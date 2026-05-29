# Chess Platform

Простой учебный monorepo с тремя частями:

- `chess-game-service` управляет партиями;
- `chess-rating-service` хранит и пересчитывает рейтинги игроков;
- `frontend` показывает данные сервисов и позволяет отправлять запросы из браузера.

## Принципы проекта

- только стандартная библиотека Go на бэкенде;
- фронт на React vite;
- минимальный общий код в `internal/platform`;
- линейная структура: `config`, `logic`, `http`, `main`;
- in-memory хранилище для MVP;
- Docker-упаковка каждого компонента и общий `docker-compose`.

## Структура

```text
services/
  chess-game-service/
    main.go
    Dockerfile
    internal/
      config/
      game/
      httpapi/
      ratingclient/

  chess-rating-service/
    main.go
    Dockerfile
    internal/
      config/
      httpapi/
      rating/

frontend/
  index.html
  package.json
  vite.config.js
  src/
    main.jsx
    App.jsx
  nginx.conf
  Dockerfile

internal/
  platform/
    config.go
    http.go
    id.go
    stats.go

config/
  game/
    dev.yaml
    test.yaml
    prod.yaml
  rating/
    dev.yaml
    test.yaml
    prod.yaml

docker-compose.yml
docs/
  TECH_PROJECT.md
```

## Что уже реализовано

`chess-game-service`

- `GET /health`
- `GET /stats`
- `POST /api/v1/games`
- `GET /api/v1/games/{id}`
- `POST /api/v1/games/{id}/resign`

`chess-rating-service`

- `GET /health`
- `GET /stats`
- `POST /api/v1/ratings/players`
- `GET /api/v1/ratings/players/{playerId}`
- `GET /api/v1/ratings/leaderboard`
- `POST /internal/v1/rating/game-result`

`frontend`

- проверка `health` и `stats` обоих сервисов;
- отдельный демо-сценарий одной кнопкой: создать партию, завершить ее сдачей и сразу получить рейтинг;
- создание партии;
- получение партии;
- завершение партии по сдаче;
- создание рейтингового профиля;
- получение профиля игрока;
- просмотр таблицы лидеров;
- вывод JSON-ответов прямо в интерфейсе.

## Локальный запуск без Docker

Нужен установленный Go и команда `go` в `PATH`.

Сначала сервис рейтингов:

```powershell
$env:APP_ENV="dev"
go run .\services\chess-rating-service
```

Потом сервис партий:

```powershell
$env:APP_ENV="dev"
go run .\services\chess-game-service
```

Фронт можно открыть любым простым static server или через Docker, который описан ниже.

## Конфиги окружений

В корне проекта лежит `.env`. Сейчас он используется как локальная точка входа и по умолчанию содержит:

```dotenv
APP_ENV=dev
```

По умолчанию:

- `config/game/<APP_ENV>.yaml` — конфиг сервиса партий;
- `config/rating/<APP_ENV>.yaml` — конфиг сервиса рейтингов.

Как это работает сейчас:

- оба сервиса при старте читают переменную `APP_ENV`;
- если переменная не была выставлена в shell или Docker, сервисы сначала попробуют взять ее из корневого `.env`;
- если `APP_ENV` не задан, берется `dev`;
- после этого каждый сервис выбирает свой файл конфигурации;
- затем значения из файла можно точечно переопределить переменными окружения.

Порядок приоритета такой:

1. встроенные значения по умолчанию в коде;
2. файл `config/.../<env>.yaml`;
3. переменные окружения.

Что зависит от конфигов:

- `port` — на каком порту будет слушать сервис;
- `logLevel` — насколько подробными будут логи;
- `version` и `env` — что сервис покажет в `/stats`;
- `readTimeoutSeconds`, `writeTimeoutSeconds`, `shutdownTimeoutSeconds` — таймауты HTTP и graceful shutdown;
- `ratingServiceBaseURL` — куда сервис партий отправляет результат завершенной партии;
- `ratingServiceTimeoutSeconds` — сколько ждать ответ от рейтинг-сервиса;
- `initialRating` и `kFactor` — базовые параметры Elo в сервисе рейтингов.

Зачем нужны `dev`, `test`, `prod`:

- `dev` — локальная разработка, удобные localhost-адреса и более подробные логи;
- `test` — отдельное окружение для проверки, обычно с другими портами и более спокойным логированием;
- `prod` — боевое окружение, где важны стабильные адреса сервисов и консервативные настройки.

Как менять конфиги:

- если нужно поменять поведение конкретного окружения надолго, редактируешь нужный файл в `config/game` или `config/rating`;
- если нужно быстро подменить одну настройку без правки файла, задаешь env-переменную перед запуском.

Можно явно переопределить:

```powershell
$env:CHESS_GAME_CONFIG="config/game/dev.yaml"
$env:CHESS_RATING_CONFIG="config/rating/dev.yaml"
```

Примеры точечных переопределений:

```powershell
$env:APP_ENV="prod"
$env:GAME_HTTP_PORT="8088"
$env:RATING_SERVICE_BASE_URL="http://rating-service:8081"
go run .\services\chess-game-service
```

```powershell
$env:APP_ENV="test"
$env:RATING_HTTP_PORT="9091"
$env:RATING_INITIAL_RATING="1400"
go run .\services\chess-rating-service
```

## Совместный запуск через Docker

Базовый `docker-compose.yml` хранит общую схему контейнеров.
Для каждого окружения есть свой override-файл:

- `docker-compose.dev.yml`
- `docker-compose.test.yml`
- `docker-compose.prod.yml`

Запуск `dev`:

```powershell
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```

После запуска `dev` будут доступны:

- фронт: `http://localhost:3000`
- сервис партий: `http://localhost:8080`
- сервис рейтингов: `http://localhost:8081`

Запуск `test`:

```powershell
docker compose -f docker-compose.yml -f docker-compose.test.yml up --build
```

После запуска `test` будут доступны:

- фронт: `http://localhost:3000`
- сервис партий: `http://localhost:9080`
- сервис рейтингов: `http://localhost:9081`

Во фронте адреса сервисов для `test` теперь подставляются автоматически на этапе Docker-сборки.

Запуск `prod`:

```powershell
docker compose -f docker-compose.yml -f docker-compose.prod.yml up --build
```

После запуска `prod` будут доступны:

- фронт: `http://localhost:3000`
- сервис партий: `http://localhost:8080`
- сервис рейтингов: `http://localhost:8081`

Остановка любого окружения:

```powershell
docker compose down
```
