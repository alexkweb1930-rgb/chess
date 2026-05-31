import datetime as dt
import json
import os
import time
import urllib.error
import urllib.request


ELASTICSEARCH_URL = os.getenv("ELASTICSEARCH_URL", "http://elasticsearch:9200").rstrip("/")
KIBANA_URL = os.getenv("KIBANA_URL", "http://kibana:5601").rstrip("/")
POLL_INTERVAL_SECONDS = int(os.getenv("POLL_INTERVAL_SECONDS", "10"))
TARGETS = os.getenv(
    "STATS_TARGETS",
    "chess-game-service=http://game-service:8080/stats,"
    "chess-rating-service=http://rating-service:8081/stats",
)


def utc_now() -> dt.datetime:
    return dt.datetime.now(dt.timezone.utc)


def request_json(method, url, payload=None, timeout=5, headers=None):
    body = None
    request_headers = headers or {}

    if payload is not None:
        body = json.dumps(payload).encode("utf-8")
        request_headers["Content-Type"] = "application/json"

    request = urllib.request.Request(url, data=body, method=method, headers=request_headers)

    with urllib.request.urlopen(request, timeout=timeout) as response:
        data = response.read()
        if not data:
            return None
        return json.loads(data.decode("utf-8"))


def wait_for_http(name, url):
    while True:
        try:
            request_json("GET", url, timeout=5)
            print(f"{name} is ready: {url}", flush=True)
            return
        except Exception as error:
            print(f"waiting for {name}: {error}", flush=True)
            time.sleep(3)


def parse_targets(raw_targets):
    targets = []
    for item in raw_targets.split(","):
        if not item.strip():
            continue
        name, url = item.split("=", 1)
        targets.append((name.strip(), url.strip()))
    return targets


def ensure_index_template():
    template = {
        "index_patterns": ["chess-stats-*"],
        "template": {
            "mappings": {
                "dynamic": True,
                "properties": {
                    "@timestamp": {"type": "date"},
                    "project": {"type": "keyword"},
                    "event": {
                        "properties": {
                            "dataset": {"type": "keyword"},
                        }
                    },
                    "service": {
                        "properties": {
                            "name": {"type": "keyword"},
                            "version": {"type": "keyword"},
                        }
                    },
                    "stats": {
                        "properties": {
                            "started_at": {"type": "date"},
                            "uptime_seconds": {"type": "long"},
                            "total_requests": {"type": "long"},
                            "response_codes": {"type": "object", "dynamic": True},
                        }
                    },
                }
            }
        },
    }

    request_json(
        "PUT",
        f"{ELASTICSEARCH_URL}/_index_template/chess-stats-template",
        template,
        timeout=10,
    )
    print("Elasticsearch index template is ready: chess-stats-template", flush=True)


def ensure_kibana_data_view():
    payload = {
        "data_view": {
            "title": "chess-stats-*",
            "name": "Chess Stats",
            "timeFieldName": "@timestamp",
        }
    }

    headers = {"kbn-xsrf": "chess-platform-monitoring"}

    try:
        request_json(
            "POST",
            f"{KIBANA_URL}/api/data_views/data_view",
            payload,
            timeout=10,
            headers=headers,
        )
        print("Kibana data view is ready: chess-stats-*", flush=True)
    except urllib.error.HTTPError as error:
        if error.code in (400, 409):
            print("Kibana data view already exists: chess-stats-*", flush=True)
            return
        raise


def normalize_stats(service_name, payload):
    raw_response_codes = payload.get("responseCodes") or {}
    response_codes = {str(code): int(count) for code, count in raw_response_codes.items()}

    for code in ("200", "201", "204", "400", "404", "405", "500"):
        response_codes.setdefault(code, 0)

    response_code_classes = {
        "2xx": sum(count for code, count in response_codes.items() if code.startswith("2")),
        "3xx": sum(count for code, count in response_codes.items() if code.startswith("3")),
        "4xx": sum(count for code, count in response_codes.items() if code.startswith("4")),
        "5xx": sum(count for code, count in response_codes.items() if code.startswith("5")),
    }

    return {
        "@timestamp": utc_now().isoformat(),
        "project": "chess-platform",
        "event": {
            "dataset": "chess.stats",
        },
        "service": {
            "name": payload.get("serviceName") or service_name,
            "version": payload.get("version") or "unknown",
        },
        "stats": {
            "started_at": payload.get("startedAt"),
            "uptime_seconds": int(payload.get("uptimeSeconds") or 0),
            "total_requests": int(payload.get("totalRequests") or 0),
            "response_codes": response_codes,
            "response_code_classes": response_code_classes,
        },
    }


def index_document(document):
    index = f"chess-stats-{utc_now().strftime('%Y.%m.%d')}"
    request_json("POST", f"{ELASTICSEARCH_URL}/{index}/_doc", document, timeout=10)


def collect_targets_once(targets):
    indexed_count = 0

    for service_name, url in targets:
        try:
            payload = request_json("GET", url, timeout=5)
            document = normalize_stats(service_name, payload)
            index_document(document)
            indexed_count += 1
            print(
                f"indexed stats: service={document['service']['name']} "
                f"total_requests={document['stats']['total_requests']}",
                flush=True,
            )
        except Exception as error:
            print(f"failed to collect {service_name} from {url}: {error}", flush=True)

    return indexed_count


def main():
    targets = parse_targets(TARGETS)

    wait_for_http("Elasticsearch", f"{ELASTICSEARCH_URL}/_cluster/health")
    wait_for_http("Kibana", f"{KIBANA_URL}/api/status")

    ensure_index_template()

    while collect_targets_once(targets) == 0:
        print("waiting until at least one stats document is indexed", flush=True)
        time.sleep(3)

    ensure_kibana_data_view()

    while True:
        collect_targets_once(targets)
        time.sleep(POLL_INTERVAL_SECONDS)


if __name__ == "__main__":
    main()
