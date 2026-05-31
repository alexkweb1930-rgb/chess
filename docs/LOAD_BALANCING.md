# Балансировка нагрузки Chess Platform

## Что добавлено

Для задания по балансировке добавлены отдельные файлы запуска:

```text
docker-compose.lb.active-active.yml
docker-compose.lb.active-passive.yml
infra/haproxy/active-active.cfg
infra/haproxy/active-passive.cfg
```

Они не заменяют обычный `docker-compose.yml`. Это отдельная демонстрационная конфигурация для двух режимов балансировки.

## Общая схема

Обычный запуск проекта:

```text
frontend -> game-service -> rating-service
```

Запуск с балансировщиком:

```text
frontend -> HAProxy :8080 -> game-service-a
                         -> game-service-b

frontend -> HAProxy :8081 -> rating-service-a
                         -> rating-service-b
```

Для браузера и пользователя адреса остаются прежними:

```text
Frontend:       http://localhost:3000
Game API:       http://localhost:8080
Rating API:     http://localhost:8081
HAProxy stats:  http://localhost:8404/stats
```

Внутри Docker-сети `game-service-a` и `game-service-b` обращаются к рейтинговому сервису тоже через HAProxy:

```text
RATING_SERVICE_BASE_URL=http://load-balancer:8081
```

Так балансировка применяется не только к запросам из браузера, но и к межсервисному вызову `game-service -> rating-service`.

## Active/Active

В режиме `active/active` оба экземпляра сервиса одновременно принимают трафик.

Для `game-service`:

```text
game-service-a: active
game-service-b: active
```

Для `rating-service`:

```text
rating-service-a: active
rating-service-b: active
```

HAProxy использует алгоритм:

```text
roundrobin
```

То есть запросы примерно равномерно распределяются между экземплярами.

Фрагмент конфигурации:

```haproxy
backend game_services
  balance roundrobin
  option httpchk GET /health
  server game-a game-service-a:8080 check
  server game-b game-service-b:8080 check
```

## Active/Passive

В режиме `active/passive` основной экземпляр получает трафик, а резервный используется только при отказе основного.

Для `game-service`:

```text
game-service-a: active
game-service-b: backup
```

Для `rating-service`:

```text
rating-service-a: active
rating-service-b: backup
```

Фрагмент конфигурации:

```haproxy
backend game_services
  balance roundrobin
  option httpchk GET /health
  server game-a game-service-a:8080 check
  server game-b game-service-b:8080 check backup
```

Ключевое слово:

```text
backup
```

означает, что сервер не получает трафик, пока есть доступные не-backup серверы.

## Зачем нужен healthcheck

В HAProxy для backend-серверов включена проверка:

```haproxy
option httpchk GET /health
```

HAProxy регулярно вызывает:

```http
GET /health
```

Если контейнер перестает отвечать, HAProxy помечает его как `DOWN` и перестает отправлять туда пользовательские запросы.

Также healthcheck прописан в Docker Compose для самих контейнеров. Это удобно для проверки через:

```bash
docker compose -f docker-compose.lb.active-active.yml ps
```

## Как запустить Active/Active

Перед запуском лучше остановить обычный стек, потому что используются те же внешние порты `3000`, `8080`, `8081`:

```bash
docker compose down
docker compose -f docker-compose.yml -f docker-compose.dev.yml -f docker-compose.monitoring.yml down
```

Запуск:

```bash
docker compose -f docker-compose.lb.active-active.yml up -d --build
```

Проверка контейнеров:

```bash
docker compose -f docker-compose.lb.active-active.yml ps
```

Проверка API:

```bash
curl -i http://localhost:8080/health
curl -i http://localhost:8081/health
```

В ответе будет заголовок:

```text
X-Backend-Server
```

Он показывает, какой backend-экземпляр обработал запрос.

Для наглядной проверки выполнить несколько запросов:

```bash
curl -i http://localhost:8080/health
curl -i http://localhost:8080/health
curl -i http://localhost:8080/health
curl -i http://localhost:8080/health
```

В active/active значение `X-Backend-Server` должно чередоваться между:

```text
game-a
game-b
```

Для rating-service:

```bash
curl -i http://localhost:8081/health
curl -i http://localhost:8081/health
curl -i http://localhost:8081/health
curl -i http://localhost:8081/health
```

Ожидаемое чередование:

```text
rating-a
rating-b
```

Открыть статистику HAProxy:

```text
http://localhost:8404/stats
```

Там видно:

- статус backend-серверов;
- количество запросов;
- active/down состояние;
- какой сервер получает трафик.

## Как остановить Active/Active

```bash
docker compose -f docker-compose.lb.active-active.yml down
```

## Как запустить Active/Passive

Запуск:

```bash
docker compose -f docker-compose.lb.active-passive.yml up -d --build
```

Проверка:

```bash
curl -i http://localhost:8080/health
curl -i http://localhost:8080/health
curl -i http://localhost:8081/health
curl -i http://localhost:8081/health
```

В нормальном состоянии все запросы должны идти на основные серверы:

```text
game-a
rating-a
```

Резервные серверы `game-b` и `rating-b` будут видны в HAProxy stats как backup.

## Демонстрация отказа в Active/Passive

Остановить основной game-service:

```bash
docker stop chess-lb-game-service-a
```

Подождать несколько секунд, чтобы HAProxy healthcheck пометил сервер как `DOWN`.

Проверить:

```bash
curl -i http://localhost:8080/health
```

Ожидаемый backend:

```text
X-Backend-Server: game-b
```

Вернуть основной контейнер:

```bash
docker start chess-lb-game-service-a
```

Через несколько секунд HAProxy снова увидит его как `UP`.

Аналогично для rating-service:

```bash
docker stop chess-lb-rating-service-a
curl -i http://localhost:8081/health
docker start chess-lb-rating-service-a
```

Ожидаемый backend после остановки primary:

```text
X-Backend-Server: rating-b
```

## Как остановить Active/Passive

```bash
docker compose -f docker-compose.lb.active-passive.yml down
```

## Важное ограничение текущего проекта

Сейчас оба backend-сервиса хранят данные в памяти процесса.

Это означает:

```text
game-service-a memory != game-service-b memory
rating-service-a memory != rating-service-b memory
```

Из-за этого active/active технически распределяет HTTP-запросы, но бизнес-сценарии с состоянием могут работать нестабильно.

Пример:

```text
POST /api/v1/games       -> попал на game-service-a
GET /api/v1/games/{id}   -> попал на game-service-b
```

`game-service-b` может не найти игру, потому что она была создана в памяти `game-service-a`.

Поэтому для текущей in-memory реализации:

- `active/active` хорошо показывает саму балансировку;
- `active/passive` лучше подходит для бизнес-сценариев с состоянием;
- production-решение требует вынести состояние из памяти контейнера.

## Как сделать состояние правильно

Правильный production-подход: backend-контейнеры должны быть stateless.

Это значит:

```text
контейнер можно остановить, перезапустить или заменить без потери данных
```

Для этого состояние нужно вынести во внешнее хранилище.

### Вариант для game-service

Игры лучше хранить в PostgreSQL.

Минимальные таблицы:

```text
games
game_events
```

`games` хранит текущее состояние партии:

```text
id
white_player_id
black_player_id
status
winner_color
created_at
updated_at
```

`game_events` хранит историю событий:

```text
id
game_id
event_type
payload
created_at
```

Тогда любой экземпляр `game-service` сможет прочитать одну и ту же игру из PostgreSQL.

### Вариант для rating-service

Рейтинги тоже лучше хранить в PostgreSQL.

Минимальные таблицы:

```text
players
rating_history
```

`players`:

```text
player_id
rating
created_at
updated_at
```

`rating_history`:

```text
id
player_id
old_rating
new_rating
reason
created_at
```

Так оба экземпляра `rating-service` будут работать с одной общей базой.

### Что добавить в проект при следующем шаге

1. Добавить PostgreSQL в `docker-compose.yml`.
2. Добавить миграции SQL.
3. Заменить in-memory repositories на SQL repositories.
4. Передавать DSN через env:

```text
DATABASE_URL=postgres://...
```

5. После этого active/active станет корректным не только технически, но и бизнес-логически.

## Что использовать в отчете

Для отчета достаточно показать:

1. Запущенный active/active compose.
2. HAProxy stats с двумя активными backend-серверами.
3. Несколько `curl -i`, где `X-Backend-Server` чередуется.
4. Запущенный active/passive compose.
5. HAProxy stats, где второй сервер помечен как backup.
6. Остановку primary-контейнера и переход трафика на backup.
7. Пояснение, что state пока in-memory, поэтому для production нужно PostgreSQL/общее хранилище.
