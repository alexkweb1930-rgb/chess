# GitLab CI/CD для проекта Chess Platform

## Что добавлено

- `.gitlab-ci.yml` - основной pipeline GitLab.
- `services/chess-game-service/Dockerfile.runtime` - runtime-образ сервиса партий.
- `services/chess-rating-service/Dockerfile.runtime` - runtime-образ сервиса рейтингов.
- `deploy/docker-compose.registry.yml` - запуск уже собранных образов из GitLab Container Registry.
- `deploy/.env.example` - пример параметров для deploy-запуска.
- `.dockerignore` - уменьшает Docker build context и не отправляет в сборку локальный мусор.

## Идея pipeline

Задание просит два связанных build plan.

Первый plan - `compile:go-services`.
Он запускается в контейнере `golang:1.24-alpine`, проверяет Go-код через `go test ./...`, компилирует два Linux-бинарника и сохраняет их как artifact:

- `bin/chess-game-service`
- `bin/chess-rating-service`

Второй plan - Docker-сборки.
Jobs `build:chess-game-service` и `build:chess-rating-service` зависят от `compile:go-services` через `needs`, получают его artifacts и собирают маленькие runtime-образы на базе Alpine. В эти образы кладутся только готовый бинарник и папка `config`.

Фронтенд собирается отдельным Docker job `build:frontend`, потому что для него artifact-бинарника нет: Vite-сборка уже описана внутри `frontend/Dockerfile`.

## Какие образы появятся в GitLab Container Registry

После успешного pipeline GitLab отправит в registry:

- `$CI_REGISTRY_IMAGE/chess-game-service:$CI_COMMIT_SHORT_SHA`
- `$CI_REGISTRY_IMAGE/chess-rating-service:$CI_COMMIT_SHORT_SHA`
- `$CI_REGISTRY_IMAGE/frontend:$CI_COMMIT_SHORT_SHA`

Если pipeline идет по default branch, дополнительно публикуется тег `latest`.

Если pipeline идет по Git tag, дополнительно публикуется тег с именем Git tag, например `v1.0.0`.

## Что сделать в GitLab

1. Залить эти файлы в репозиторий и сделать push.
2. В проекте GitLab открыть `Settings -> General -> Visibility, project features, permissions` и убедиться, что `Container Registry` включен.
3. Открыть `Settings -> CI/CD -> Runners`.
4. Подключить runner с Docker executor или включить shared runners, если они доступны в вашем GitLab.
5. Если используете свой runner, для Docker-in-Docker ему нужен `privileged = true` в `config.toml`.
6. Открыть `Settings -> CI/CD -> Variables` и при необходимости добавить переменные:

| Variable | Зачем |
| --- | --- |
| `VITE_GAME_SERVICE_URL` | URL game-service, который будет зашит во frontend при сборке |
| `VITE_RATING_SERVICE_URL` | URL rating-service, который будет зашит во frontend при сборке |

GitLab сам предоставляет переменные для публикации образов:

- `CI_REGISTRY`
- `CI_REGISTRY_IMAGE`
- `CI_REGISTRY_USER`
- `CI_REGISTRY_PASSWORD`
- `CI_COMMIT_SHORT_SHA`
- `CI_COMMIT_BRANCH`
- `CI_DEFAULT_BRANCH`
- `CI_COMMIT_TAG`

Их руками добавлять не нужно.

## Как запустить готовые образы локально или на сервере

1. Скопировать пример env-файла:

```bash
cp deploy/.env.example deploy/.env
```

2. В `deploy/.env` заменить `REGISTRY_IMAGE` на адрес registry вашего проекта. Он обычно выглядит так:

```dotenv
REGISTRY_IMAGE=registry.gitlab.com/<group>/<project>
```

3. Войти в registry:

```bash
docker login registry.gitlab.com
```

4. Запустить:

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.registry.yml up -d
```

5. Остановить:

```bash
docker compose -f deploy/docker-compose.registry.yml down
```

## Почему оставлены старые Dockerfile

Файлы `services/*/Dockerfile` удобны для локальной разработки: Docker сам компилирует сервис из исходников.

Файлы `services/*/Dockerfile.runtime` нужны именно для CI/CD: они не компилируют код, а берут готовый artifact из первого plan. Так хорошо видно разделение:

- compile plan отвечает за сборку и проверку бинарников;
- package plan отвечает за упаковку и доставку Docker images.

Это проще проверять в учебной работе и ближе к промышленной схеме, где один результат сборки переиспользуется на следующих шагах pipeline.
