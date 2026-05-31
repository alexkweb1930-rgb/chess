# GitHub Actions для Chess Platform

## Что настроено

Основной workflow лежит в `.github/workflows/ci.yml`.

Он делает сборку и доставку трех частей проекта:

1. `chess-game-service`
2. `chess-rating-service`
3. `frontend`

Go-сервисы собираются в два связанных этапа: сначала компиляция бинарников, потом упаковка этих бинарников в Docker images. Это ровно та схема, которая нужна для учебной работы про сборку и доставку микросервиса.

## Этап 1: compile-go-services

Job `compile-go-services` запускается на `ubuntu-latest`.

Что происходит:

1. Репозиторий скачивается через `actions/checkout`.
2. Устанавливается Go версии `1.24.x`.
3. Запускается проверка:

```bash
go test ./...
```

4. Собираются Linux-бинарники:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bin/chess-game-service ./services/chess-game-service
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bin/chess-rating-service ./services/chess-rating-service
```

5. Готовые файлы сохраняются как artifact `go-services`:

```text
bin/chess-game-service
bin/chess-rating-service
```

Зачем это нужно: мы фиксируем результат компиляции отдельно от Docker-сборки. Следующие jobs не пересобирают Go-код, а берут уже готовые бинарники.

## Этап 2: build-chess-game-service

Job `build-chess-game-service` зависит от `compile-go-services`.

Что происходит:

1. Репозиторий скачивается заново.
2. Artifact `go-services` скачивается в папку `bin`.
3. Бинарнику возвращается право на запуск:

```bash
chmod +x bin/chess-game-service
```

4. Через `docker/metadata-action` готовятся Docker tags.
5. Через `docker/build-push-action` собирается image по файлу:

```text
services/chess-game-service/Dockerfile.runtime
```

Runtime Dockerfile не компилирует Go-код. Он только кладет в Alpine-образ готовый бинарник и папку `config`.

## Этап 3: build-chess-rating-service

Job `build-chess-rating-service` устроен так же, как job для game-service.

Отличия:

- используется бинарник `bin/chess-rating-service`;
- используется Dockerfile `services/chess-rating-service/Dockerfile.runtime`;
- публикуется отдельный image рейтингового сервиса.

## Этап 4: build-frontend

Job `build-frontend` собирает Docker image фронтенда.

Что происходит:

1. Репозиторий скачивается.
2. Подготавливаются Docker tags.
3. Собирается image через `frontend/Dockerfile`.
4. В Docker build передаются адреса backend-сервисов:

```text
VITE_GAME_SERVICE_URL=http://localhost:8080
VITE_RATING_SERVICE_URL=http://localhost:8081
```

Внутри `frontend/Dockerfile` выполняется:

```bash
npm ci
npm run build
```

После сборки статические файлы попадают в nginx image.

## Публикация Docker images

Images публикуются в GitHub Container Registry:

```text
ghcr.io/<owner>/<repo>/chess-game-service
ghcr.io/<owner>/<repo>/chess-rating-service
ghcr.io/<owner>/<repo>/frontend
```

Публикация включается только для `push` и Git tags. Для pull request images собираются, но не публикуются.

Теги создаются автоматически:

- короткий commit SHA;
- имя ветки, например `master`;
- `latest` для default branch;
- имя Git tag, например `v1.0.0`.

## Когда запускается workflow

Workflow запускается:

- при push в `master`;
- при push в `main`;
- при pull request в `master` или `main`;
- при push Git tag вида `v*`;
- вручную через `Actions -> CI -> Run workflow`.

## Что нужно в настройках GitHub

1. Открыть репозиторий на GitHub.
2. Перейти в `Settings -> Actions -> General`.
3. Разрешить GitHub Actions.
4. В `Workflow permissions` выбрать `Read and write permissions`.
5. Сохранить настройки.

Отдельный token для публикации packages не нужен: workflow использует стандартный `secrets.GITHUB_TOKEN`.

## Как запустить опубликованные images

Для запуска готовых образов используется файл `deploy/docker-compose.registry.yml`.

Пример `deploy/.env`:

```dotenv
REGISTRY_IMAGE=ghcr.io/<owner>/<repo>
IMAGE_TAG=latest
APP_ENV=prod

FRONTEND_PORT=3000
GAME_PORT=8080
RATING_PORT=8081
```

Войти в GHCR:

```bash
docker login ghcr.io
```

Для login обычно нужен GitHub Personal Access Token с правом `read:packages`.

Запуск:

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.registry.yml up -d
```

Остановка:

```bash
docker compose -f deploy/docker-compose.registry.yml down
```
