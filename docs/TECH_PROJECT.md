# Технический проект системы шахматных партий и рейтингов

## 1. Назначение

Проект представляет собой распределенную систему из двух взаимодействующих микросервисов:

- `Chess Game Service` — сервис управления шахматными партиями;
- `Chess Rating Service` — сервис расчета и хранения шахматных рейтингов игроков.

Система позволяет создавать шахматные партии, выполнять ходы, завершать игру и автоматически обновлять рейтинг игроков после появления результата партии.

Главная цель проекта — показать не один изолированный HTTP-сервис, а микросервисную архитектуру, где сервисы имеют разные зоны ответственности и обмениваются данными друг с другом.

## 2. Состав микросервисов

### 2.1 Chess Game Service

`Chess Game Service` отвечает за жизненный цикл шахматной партии:

- создание новой партии;
- хранение состояния партии;
- прием и валидация ходов;
- хранение истории ходов;
- определение статуса партии;
- завершение партии;
- передача результата партии в `Chess Rating Service`.

Сервис не хранит рейтинг игроков и не выполняет расчет рейтингов.

### 2.2 Chess Rating Service

`Chess Rating Service` отвечает за рейтинговую систему:

- хранение текущего рейтинга игроков;
- создание рейтингового профиля игрока;
- получение рейтинга игрока;
- пересчет рейтинга после завершения партии;
- хранение истории изменений рейтинга;
- предоставление таблицы лидеров.

Сервис не хранит состояние шахматной доски и не проверяет корректность ходов.

## 3. Обоснование выбора стека

Основной стек для обоих микросервисов — `Go`.

Go хорошо подходит для этого задания, потому что:

- компилируется в один бинарный файл;
- удобно используется в двухэтапной сборке;
- хорошо подходит для stateless HTTP API;
- прост в контейнеризации;
- имеет низкий overhead при запуске нескольких экземпляров сервиса;
- удобен для реализации `health`, `stats`, graceful shutdown и межсервисного HTTP-взаимодействия.

Выбранный стек:

- язык: `Go`;
- HTTP: стандартный `net/http` или `Gin`;
- база данных: `PostgreSQL`;
- конфигурация: отдельные файлы для `dev`, `test`, `prod` + переменные окружения;
- логирование: структурированные JSON-логи;
- контейнеризация: `Docker`;
- мониторинг логов: `Elastic Stack`;
- балансировка: `Nginx`.

## 4. Границы ответственности

### 4.1 В зоне ответственности Chess Game Service

- партии;
- ходы;
- состояние доски;
- статус партии;
- результат партии;
- отправка события о завершении партии.

### 4.2 В зоне ответственности Chess Rating Service

- рейтинги игроков;
- история изменения рейтингов;
- расчет нового рейтинга;
- таблица лидеров;
- прием результата завершенной партии от сервиса партий.

### 4.3 Вне зоны ответственности обоих сервисов

- регистрация и аутентификация пользователей;
- чат игроков;
- уведомления;
- платежи;
- пользовательский интерфейс;
- турнирная сетка.

## 5. Общая архитектура

Система строится как набор stateless HTTP-сервисов. Каждый сервис имеет собственную базу данных или собственную схему в PostgreSQL.

```text
Client
Load Balancer
    Chess Game Service
        Game Database
    Chess Rating Service
        Rating Database

Chess Game Service --HTTP/internal API--> Chess Rating Service
```

Основной сценарий взаимодействия:

1. клиент создает партию через `Chess Game Service`;
2. игроки выполняют ходы через `Chess Game Service`;
3. партия завершается с результатом;
4. `Chess Game Service` отправляет результат в `Chess Rating Service`;
5. `Chess Rating Service` пересчитывает рейтинги игроков;
6. клиент может запросить обновленные рейтинги и таблицу лидеров.

## 6. Межсервисное взаимодействие

Для MVP используется синхронное HTTP-взаимодействие:

- `Chess Game Service` вызывает внутренний endpoint `Chess Rating Service`;
- вызов выполняется после завершения партии;
- если рейтинг-сервис временно недоступен, результат партии остается сохраненным в сервисе партий;
- повторная отправка результата может быть реализована через фоновую задачу или outbox-паттерн на следующем этапе.

Внутренний endpoint:

`POST /internal/v1/rating/game-result`

Request:

```json
{
  "gameId": "6c5165aa-cf6b-4ceb-983a-f0e972d04e0b",
  "whitePlayerId": "player-1",
  "blackPlayerId": "player-2",
  "result": "white_win",
  "finishedAt": "2026-04-20T10:00:00Z"
}
```

Response `200 OK`:

```json
{
  "whitePlayerId": "player-1",
  "whiteRatingBefore": 1200,
  "whiteRatingAfter": 1216,
  "blackPlayerId": "player-2",
  "blackRatingBefore": 1200,
  "blackRatingAfter": 1184
}
```

На более зрелом этапе HTTP-вызов можно заменить или дополнить асинхронным обменом через брокер сообщений, например Kafka или RabbitMQ. Для текущего задания HTTP-вариант проще реализовать и легче показать в документации.

## 7. Функциональные требования

### 7.1 Chess Game Service

Сервис должен поддерживать:

- создание новой партии;
- получение партии по идентификатору;
- выполнение хода;
- получение истории ходов;
- получение текущего состояния доски;
- завершение партии по мату, пату, ничьей или сдаче;
- отправку результата партии в рейтинг-сервис;
- сервисные методы `health` и `stats`.

### 7.2 Chess Rating Service

Сервис должен поддерживать:

- создание рейтингового профиля игрока;
- получение рейтинга игрока;
- прием результата завершенной партии;
- пересчет рейтинга двух игроков;
- получение истории изменения рейтинга;
- получение таблицы лидеров;
- сервисные методы `health` и `stats`.

## 8. Нефункциональные требования

- REST API по HTTP/JSON;
- отдельная зона ответственности для каждого сервиса;
- независимая сборка и доставка каждого сервиса;
- конфиги `dev`, `test`, `prod` для каждого сервиса;
- централизованное логирование;
- метод `stats` в каждом сервисе;
- поддержка горизонтального масштабирования;
- возможность работы за балансировщиком;
- поддержка режимов `active/active` и `active/passive`.

## 9. API Chess Game Service

Базовый префикс: `/api/v1`

### 9.1 Создать партию

`POST /api/v1/games`

Request:

```json
{
  "whitePlayerId": "player-1",
  "blackPlayerId": "player-2"
}
```

Response `201 Created`:

```json
{
  "id": "6c5165aa-cf6b-4ceb-983a-f0e972d04e0b",
  "status": "in_progress",
  "currentTurn": "white",
  "boardFEN": "startpos"
}
```

### 9.2 Получить партию

`GET /api/v1/games/{id}`

### 9.3 Сделать ход

`POST /api/v1/games/{id}/moves`

Request:

```json
{
  "playerColor": "white",
  "notation": "e2e4"
}
```

### 9.4 Получить историю ходов

`GET /api/v1/games/{id}/moves`

### 9.5 Получить состояние доски

`GET /api/v1/games/{id}/state`

### 9.6 Сдаться

`POST /api/v1/games/{id}/resign`

Request:

```json
{
  "playerColor": "black"
}
```

Response `200 OK`:

```json
{
  "gameId": "6c5165aa-cf6b-4ceb-983a-f0e972d04e0b",
  "status": "resigned",
  "result": "white_win",
  "ratingUpdateStatus": "sent"
}
```

## 10. API Chess Rating Service

Базовый префикс: `/api/v1`

### 10.1 Создать рейтинговый профиль

`POST /api/v1/ratings/players`

Request:

```json
{
  "playerId": "player-1",
  "initialRating": 1200
}
```

Response `201 Created`:

```json
{
  "playerId": "player-1",
  "rating": 1200,
  "gamesPlayed": 0
}
```

### 10.2 Получить рейтинг игрока

`GET /api/v1/ratings/players/{playerId}`

Response `200 OK`:

```json
{
  "playerId": "player-1",
  "rating": 1216,
  "gamesPlayed": 1,
  "wins": 1,
  "losses": 0,
  "draws": 0
}
```

### 10.3 Получить историю изменения рейтинга

`GET /api/v1/ratings/players/{playerId}/history`

### 10.4 Получить таблицу лидеров

`GET /api/v1/ratings/leaderboard?limit=50`

### 10.5 Принять результат партии

Внутренний endpoint для вызова из `Chess Game Service`:

`POST /internal/v1/rating/game-result`

## 11. Сервисные методы

Оба микросервиса должны иметь одинаковые сервисные endpoints:

- `GET /health`;
- `GET /stats`.

`GET /health` возвращает:

```json
{
  "status": "ok"
}
```

`GET /stats` возвращает:

```json
{
  "serviceName": "chess-game-service",
  "version": "1.0.0",
  "startedAt": "2026-04-20T10:00:00Z",
  "uptimeSeconds": 3600,
  "totalRequests": 1520,
  "responseCodes": {
    "200": 1400,
    "400": 50,
    "404": 20,
    "500": 50
  }
}
```

Для `Chess Rating Service` поле `serviceName` будет равно `chess-rating-service`.

## 12. Доменная модель Chess Game Service

### 12.1 Game

- `id` — UUID партии;
- `white_player_id` — идентификатор белых;
- `black_player_id` — идентификатор черных;
- `status` — статус партии;
- `result` — результат партии;
- `current_turn` — чей ход;
- `board_fen` — текущее состояние доски;
- `rating_update_status` — статус отправки результата в рейтинг-сервис;
- `created_at` — дата создания;
- `updated_at` — дата обновления;
- `finished_at` — дата завершения.

### 12.2 Move

- `id` — UUID хода;
- `game_id` — идентификатор партии;
- `move_number` — номер хода;
- `player_color` — цвет игрока;
- `notation` — ход в SAN или UCI;
- `fen_after_move` — состояние доски после хода;
- `is_check` — признак шаха;
- `is_checkmate` — признак мата;
- `created_at` — дата фиксации хода.

## 13. Доменная модель Chess Rating Service

### 13.1 RatingProfile

- `player_id` — идентификатор игрока;
- `rating` — текущий рейтинг;
- `games_played` — количество сыгранных партий;
- `wins` — победы;
- `losses` — поражения;
- `draws` — ничьи;
- `created_at` — дата создания профиля;
- `updated_at` — дата обновления профиля.

### 13.2 RatingHistory

- `id` — UUID записи;
- `player_id` — идентификатор игрока;
- `game_id` — идентификатор партии;
- `rating_before` — рейтинг до партии;
- `rating_after` — рейтинг после партии;
- `rating_delta` — изменение рейтинга;
- `result` — результат игрока в партии;
- `created_at` — дата изменения рейтинга.

## 14. Модель данных

### 14.1 Таблицы Chess Game Service

`games`:

- `id uuid primary key`;
- `white_player_id varchar(64) not null`;
- `black_player_id varchar(64) not null`;
- `status varchar(32) not null`;
- `result varchar(32) not null default 'none'`;
- `current_turn varchar(8) not null`;
- `board_fen text not null`;
- `rating_update_status varchar(32) not null default 'not_required'`;
- `created_at timestamptz not null`;
- `updated_at timestamptz not null`;
- `finished_at timestamptz null`.

`moves`:

- `id uuid primary key`;
- `game_id uuid not null references games(id) on delete cascade`;
- `move_number int not null`;
- `player_color varchar(8) not null`;
- `notation varchar(32) not null`;
- `fen_after_move text not null`;
- `is_check boolean not null default false`;
- `is_checkmate boolean not null default false`;
- `created_at timestamptz not null`.

### 14.2 Таблицы Chess Rating Service

`rating_profiles`:

- `player_id varchar(64) primary key`;
- `rating int not null`;
- `games_played int not null default 0`;
- `wins int not null default 0`;
- `losses int not null default 0`;
- `draws int not null default 0`;
- `created_at timestamptz not null`;
- `updated_at timestamptz not null`.

`rating_history`:

- `id uuid primary key`;
- `player_id varchar(64) not null`;
- `game_id uuid not null`;
- `rating_before int not null`;
- `rating_after int not null`;
- `rating_delta int not null`;
- `result varchar(32) not null`;
- `created_at timestamptz not null`.

## 15. Алгоритм расчета рейтинга

Для MVP используется упрощенный Elo-подход:

- начальный рейтинг игрока: `1200`;
- коэффициент изменения: `K = 32`;
- победитель получает положительное изменение рейтинга;
- проигравший получает отрицательное изменение рейтинга;
- при ничьей изменение зависит от разницы рейтингов.

Формула:

```text
expectedScore = 1 / (1 + 10 ^ ((opponentRating - playerRating) / 400))
newRating = oldRating + K * (actualScore - expectedScore)
```

Где:

- `actualScore = 1` для победы;
- `actualScore = 0.5` для ничьей;
- `actualScore = 0` для поражения.

## 16. Конфигурация

Для каждого микросервиса создаются отдельные конфиги:

```text
config/
  game/
    dev.yaml
    test.yaml
    prod.yaml
  rating/
    dev.yaml
    test.yaml
    prod.yaml
```

Дополнительные параметры `Chess Game Service`:

- `rating_service.base_url`;
- `rating_service.timeout`;
- `rating_service.retry_count`.

Дополнительные параметры `Chess Rating Service`:

- `rating.initial_rating`;
- `rating.k_factor`.

## 17. Сборка и доставка

Для каждого микросервиса используется одинаковый двухэтапный цикл сборки.

### 17.1 Билд-план 1

Первый билд-план:

1. забирает исходный код сервиса из репозитория;
2. запускает сборку в Docker-контейнере с нужной версией Go;
3. выполняет тесты;
4. компилирует бинарный файл;
5. сохраняет бинарник как артефакт.

Артефакты:

- `chess-game-service`;
- `chess-rating-service`.

### 17.2 Билд-план 2

Второй билд-план:

1. стартует после успешного завершения первого плана;
2. получает бинарник сервиса;
3. берет конфиги и параметры деплоя из отдельного репозитория;
4. собирает Docker image;
5. публикует image в registry.

Для каждого сервиса собирается отдельный Docker image.

## 18. Мониторинг

Оба микросервиса пишут структурированные JSON-логи в stdout.

Elastic Stack используется так:

1. сервисы пишут логи в stdout;
2. Filebeat собирает логи контейнеров;
3. Elasticsearch хранит события;
4. Kibana отображает дашборды.

Базовые дашборды:

- количество запросов по сервисам;
- распределение HTTP-кодов;
- ошибки `5xx`;
- среднее время ответа;
- количество завершенных партий;
- количество пересчетов рейтинга;
- ошибки межсервисного взаимодействия.

## 19. Балансировка нагрузки

### 19.1 Active/active

В режиме `active/active` одновременно запущены несколько экземпляров каждого сервиса:

```text
Load Balancer
  chess-game-service-1
  chess-game-service-2

Load Balancer
  chess-rating-service-1
  chess-rating-service-2
```

Особенности:

- каждый сервис stateless;
- состояние хранится в БД;
- отказ одного экземпляра не останавливает систему;
- сервис партий может отправлять запросы в рейтинг-сервис через внутренний балансировщик.

### 19.2 Active/passive

В режиме `active/passive` один экземпляр активен, второй находится в резерве:

```text
active:  chess-game-service-1
passive: chess-game-service-2

active:  chess-rating-service-1
passive: chess-rating-service-2
```

При отказе active-инстанса трафик переключается на passive-инстанс.

## 20. Предлагаемая структура каталогов

```text
chess-platform/
  services/
    chess-game-service/
      cmd/
      internal/
      config/
      migrations/
      Dockerfile
    chess-rating-service/
      cmd/
      internal/
      config/
      migrations/
      Dockerfile
  deploy/
    nginx/
      active-active.conf
      active-passive.conf
    docker-compose.yml
  docs/
    TECH_PROJECT.md
```

## 21. Этапы реализации

### Этап 1. Базовая инфраструктура

- подготовить каркас двух Go-сервисов;
- добавить конфиги окружений;
- реализовать `health` и `stats` в обоих сервисах;
- добавить structured logging.

### Этап 2. Chess Game Service

- реализовать создание партии;
- реализовать получение партии;
- реализовать выполнение хода;
- реализовать завершение партии;
- сохранить результат партии.

### Этап 3. Chess Rating Service

- реализовать профиль рейтинга;
- реализовать получение рейтинга;
- реализовать прием результата партии;
- реализовать пересчет Elo;
- реализовать таблицу лидеров.

### Этап 4. Межсервисное взаимодействие

- добавить HTTP-клиент в `Chess Game Service`;
- настроить вызов `Chess Rating Service` после завершения партии;
- обработать ошибки и retry;
- добавить статус отправки результата рейтинга.

### Этап 5. Эксплуатация

- добавить Dockerfile для обоих сервисов;
- описать двухэтапную сборку;
- добавить конфигурации балансировщика;
- описать Elastic Stack и дашборды.

## 22. Минимально жизнеспособная версия

MVP должен включать:

- два отдельных микросервиса;
- `GET /health` и `GET /stats` в каждом сервисе;
- создание и завершение партии в `Chess Game Service`;
- создание и получение рейтинга в `Chess Rating Service`;
- вызов рейтинг-сервиса после завершения партии;
- пересчет рейтинга по упрощенной Elo-формуле;
- конфиги `dev/test/prod`;
- Dockerfile для каждого сервиса;
- описание active/active и active/passive.

## 23. Зафиксированные решения

- проект состоит из двух микросервисов;
- `Chess Game Service` управляет партиями;
- `Chess Rating Service` управляет рейтингами;
- сервисы взаимодействуют по HTTP;
- результат партии передается из сервиса партий в сервис рейтингов;
- оба сервиса реализуются на Go;
- оба сервиса имеют `health` и `stats`;
- каждый сервис собирается в отдельный бинарник;
- каждый сервис упаковывается в отдельный Docker image;
- масштабирование и отказоустойчивость проектируются для обоих сервисов.

## 24. Следующий шаг

Следующий шаг после обновления документации — привести кодовую структуру к двум сервисам:

1. выделить текущий каркас в `services/chess-game-service`;
2. создать второй каркас `services/chess-rating-service`;
3. реализовать общий формат `health` и `stats`;
4. добавить внутренний HTTP-вызов из сервиса партий в сервис рейтингов.
