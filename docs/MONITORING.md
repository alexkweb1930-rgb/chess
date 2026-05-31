# Мониторинг Chess Platform

## Что добавлено

Мониторинг вынесен в отдельный compose-файл:

```text
docker-compose.monitoring.yml
```

Он добавляет к проекту три служебных контейнера:

- `elasticsearch` - хранилище метрик и технических событий;
- `kibana` - веб-интерфейс для поиска данных и построения дашбордов;
- `stats-collector` - простой Python-сборщик, который опрашивает `/stats` и пишет документы в Elasticsearch.

Ядро Elastic-стека - это два контейнера: Elasticsearch и Kibana. `stats-collector` нужен как сборщик: без него Elasticsearch будет пустым, потому что сервисы сами в него ничего не отправляют.

## Что уже есть в сервисах

У обоих backend-сервисов есть технические endpoints:

```http
GET /health
GET /stats
```

`/health` нужен для проверки живости контейнера.

`/stats` возвращает служебную информацию:

- имя сервиса;
- версию;
- дату запуска;
- uptime;
- общее количество обработанных HTTP-запросов;
- счетчики ответов по HTTP-кодам.

Пример:

```json
{
  "serviceName": "chess-game-service",
  "version": "1.0.0",
  "startedAt": "2026-05-31T10:00:00Z",
  "uptimeSeconds": 120,
  "totalRequests": 42,
  "responseCodes": {
    "200": 40,
    "404": 2
  }
}
```

## Как работает сбор данных

`stats-collector` каждые 10 секунд опрашивает:

```text
http://game-service:8080/stats
http://rating-service:8081/stats
```

Сборщик нормализует ответ сервиса в документы индекса `chess-stats-*`.

Основные поля:

```text
@timestamp
project
event.dataset
service.name
service.version
stats.started_at
stats.uptime_seconds
stats.total_requests
stats.response_codes.200
stats.response_codes.400
stats.response_codes.404
stats.response_codes.405
stats.response_codes.500
stats.response_code_classes.2xx
stats.response_code_classes.3xx
stats.response_code_classes.4xx
stats.response_code_classes.5xx
```

Все данные отправляются в Elasticsearch. Kibana подключается к Elasticsearch и показывает эти данные через Discover, Lens и Dashboard.

Состояние контейнеров проверяется Docker healthcheck. Это видно в Docker Desktop и через `docker compose ps`.

## Запуск dev-окружения с мониторингом

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml -f docker-compose.monitoring.yml up --build
```

Если нужно запускать уже готовые images из GitHub Container Registry:

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.registry.yml -f docker-compose.monitoring.yml up -d
```

После запуска будут доступны:

```text
Frontend:       http://localhost:3000
Game service:  http://localhost:8080
Rating service:http://localhost:8081
Elasticsearch: http://localhost:9200
Kibana:        http://localhost:5601
```

Первый старт Kibana может занять 1-2 минуты.

## Проверка healthcheck

Проверить состояние контейнеров:

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml -f docker-compose.monitoring.yml ps
```

У `game-service` и `rating-service` должен появиться статус `healthy`.

Ручная проверка:

```bash
curl http://localhost:8080/health
curl http://localhost:8081/health
```

## Проверка /stats

Сделать несколько запросов к сервисам:

```bash
curl http://localhost:8080/stats
curl http://localhost:8081/stats
curl http://localhost:8080/api/v1/games
```

Потом снова открыть:

```bash
curl http://localhost:8080/stats
```

Значение `totalRequests` должно увеличиться, а в `responseCodes` должны появиться коды ответов.

## Проверка данных в Elasticsearch

Открыть:

```text
http://localhost:9200/_cat/indices?v
```

Должны появиться индексы статистики, например:

```text
chess-stats-*
```

Можно проверить документы:

```text
http://localhost:9200/chess-stats-*/_search?pretty
```

## Настройка Kibana

1. Открыть `http://localhost:5601`.
2. Перейти в `Stack Management -> Data Views`.
3. Data view обычно создается автоматически контейнером `stats-collector`. Если его нет, создать вручную:

```text
chess-stats-*
```

4. В качестве time field выбрать:

```text
@timestamp
```

5. Перейти в `Analytics -> Discover`.
6. Выбрать data view `Chess Stats` или `chess-stats-*`.
7. Проверить, что появляются документы с полями:

```text
service.name
stats.total_requests
stats.uptime_seconds
stats.response_codes.200
```

## Какие дашборды сделать

Минимальный набор для задания:

1. `Total requests by service`
   - источник: `chess-stats-*`;
   - visualization: Lens line chart;
   - Y-axis: `stats.total_requests`, function `Max`;
   - Break down by: `service.name`;
   - X-axis: `@timestamp`.

2. `Response codes`
   - источник: `chess-stats-*`;
   - visualization: Lens line chart или bar chart;
   - Y-axis: `stats.response_codes.200`, function `Max`;
   - Break down by: `service.name`;
   - при необходимости добавить отдельные слои для `stats.response_codes.404` и `stats.response_codes.500`.

3. `Service uptime`
   - источник: `chess-stats-*`;
   - visualization: Lens line chart;
   - Y-axis: `stats.uptime_seconds`, function `Max`;
   - Break down by: `service.name`.

4. `Healthcheck status`
   - источник: `docker compose ps`;
   - для отчета можно приложить скрин Docker Desktop или вывод команды с состоянием `healthy`.

## Как создать Dashboard в Kibana

Перед созданием дашборда убедиться, что выбран правильный data view:

```text
Chess Stats
```

или:

```text
chess-stats-*
```

Если выбран старый `metricbeat-*`, данных по текущей схеме не будет.

Поля `http.chess_game.*` и `http.chess_rating.*` относились к старой схеме Metricbeat. В текущей схеме отдельные наборы полей для game/rating не нужны. Оба сервиса пишутся в один индекс `chess-stats-*`, а конкретный сервис выбирается через поле:

```text
service.name
```

Например:

```text
service.name : "chess-rating-service"
```

### Панель 1: Total requests by service

1. Открыть `Analytics -> Dashboard`.
2. Нажать `Create dashboard`.
3. Нажать `Create visualization`.
4. Выбрать data view `Chess Stats`.
5. В типе визуализации выбрать `Line`.
6. В `Vertical axis` выбрать поле:

```text
stats.total_requests
```

7. Функция агрегации:

```text
Maximum
```

8. В `Horizontal axis` выбрать:

```text
@timestamp
```

9. В `Break down by` выбрать:

```text
service.name
```

10. Нажать `Save and return`.
11. Название панели:

```text
Total requests by service
```

### Панель 2: Service uptime

1. На dashboard нажать `Create visualization`.
2. Data view: `Chess Stats`.
3. Тип: `Line`.
4. `Vertical axis`:

```text
stats.uptime_seconds
```

5. Функция:

```text
Maximum
```

6. `Horizontal axis`:

```text
@timestamp
```

7. `Break down by`:

```text
service.name
```

8. Сохранить как:

```text
Service uptime
```

### Панель 3: HTTP 200 responses

1. На dashboard нажать `Create visualization`.
2. Data view: `Chess Stats`.
3. Тип: `Bar vertical stacked` или `Line`.
4. `Vertical axis`:

```text
stats.response_codes.200
```

5. Функция:

```text
Maximum
```

6. `Horizontal axis`:

```text
@timestamp
```

7. `Break down by`:

```text
service.name
```

8. Сохранить как:

```text
HTTP 200 responses
```

Если появятся ошибки `404` или `500`, аналогично можно добавить панели:

```text
stats.response_codes.404
stats.response_codes.500
```

Также можно строить графики по классам ответов:

```text
stats.response_code_classes.2xx
stats.response_code_classes.4xx
stats.response_code_classes.5xx
```

### Фильтр по конкретному сервису

В верхней строке KQL можно фильтровать данные:

```text
service.name : "chess-rating-service"
```

или:

```text
service.name : "chess-game-service"
```

### Если Kibana показывает No results

Проверить три вещи:

1. Data view должен быть `Chess Stats`, а не `metricbeat-*`.
2. Time range справа сверху должен включать текущий момент, например `Last 15 minutes`.
3. В Elasticsearch должен быть индекс:

```text
chess-stats-*
```

Проверка:

```bash
docker exec chess-elasticsearch curl -s "http://localhost:9200/chess-stats-*/_search?size=1&pretty"
```

## Остановка

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml -f docker-compose.monitoring.yml down
```

Если нужно удалить данные Elasticsearch:

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml -f docker-compose.monitoring.yml down -v
```
