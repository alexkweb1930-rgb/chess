# GitHub Actions для Chess Platform

## Что добавлено

Workflow лежит в `.github/workflows/ci.yml`.

Он повторяет учебную схему из GitLab CI/CD:

1. `compile-go-services` компилирует два Go-бинарника и сохраняет их как artifact.
2. `build-chess-game-service` скачивает artifact и собирает Docker image для сервиса партий.
3. `build-chess-rating-service` скачивает artifact и собирает Docker image для сервиса рейтингов.
4. `build-frontend` собирает Docker image фронтенда через `frontend/Dockerfile`.

## Когда запускается

Workflow запускается:

- при push в `master`;
- при push в `main`;
- при pull request в `master` или `main`;
- при push Git tag вида `v*`, например `v1.0.0`;
- вручную через `Actions -> CI -> Run workflow`.

На pull request образы только собираются, но не публикуются.
На push и tag образы собираются и публикуются в GitHub Container Registry.

## Куда публикуются Docker images

GitHub Actions публикует образы в GHCR:

```text
ghcr.io/<owner>/<repo>/chess-game-service
ghcr.io/<owner>/<repo>/chess-rating-service
ghcr.io/<owner>/<repo>/frontend
```

Теги создаются автоматически:

- короткий commit SHA;
- имя ветки, например `master`;
- `latest` для default branch;
- имя Git tag, например `v1.0.0`.

## Что нужно включить в GitHub

1. Открыть репозиторий на GitHub.
2. Перейти в `Settings -> Actions -> General`.
3. Включить `Allow all actions and reusable workflows` или разрешить actions:
   - `actions/checkout`
   - `actions/setup-go`
   - `actions/upload-artifact`
   - `actions/download-artifact`
   - `docker/login-action`
   - `docker/metadata-action`
   - `docker/build-push-action`
4. В этом же разделе открыть `Workflow permissions`.
5. Выбрать `Read and write permissions`.
6. Сохранить настройки.

Отдельный token добавлять не нужно: workflow использует стандартный `secrets.GITHUB_TOKEN`.

## Как запустить

Обычный вариант:

```bash
git add .
git commit -m "Add GitHub Actions CI"
git push origin master
```

Если основная ветка называется `main`, push делается в `main`.

После push:

1. Открыть вкладку `Actions`.
2. Выбрать workflow `CI`.
3. Дождаться прохождения jobs.
4. Открыть `Packages` в профиле/организации или в репозитории и проверить опубликованные images.

## Как запускать опубликованные images

Для `deploy/docker-compose.registry.yml` можно использовать GHCR вместо GitLab Registry.

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
