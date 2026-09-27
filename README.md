# TripGo — репозиторий для лабораторных работ

Заготовка курса «Разработка микросервисов на Go». Здесь вы делаете все пять
работ: каждая следующая продолжает предыдущую, переписывать сервис с нуля не
нужно.

## Что делать сразу

1. **Fork** этого репозитория к себе. Форк нужен, чтобы преподаватели видели
   список всех работ курса одной страницей.
2. Заведите модуль:

```bash
git clone git@github.com:<ваш-логин>/<ваш-репозиторий>.git
cd <ваш-репозиторий>
go mod init github.com/<ваш-логин>/<ваш-репозиторий>
```

Путь модуля потом не меняется — иначе придётся править все импорты. Проще всего
взять адрес своего репозитория, каким бы он ни был.

Дальше — [`homework/docs/getting-started.md`](https://github.com/course-go-autumn-2026/course/blob/main/homework/docs/getting-started.md)
в репозитории курса: инструменты, окружение, миграции, вид сданной работы.

## Где что лежит

| Что | Где |
|---|---|
| Задания, документация, контракты | [`course-go-autumn-2026/course`](https://github.com/course-go-autumn-2026/course) |
| Слайды и записи лекций | [`lections/`](https://github.com/course-go-autumn-2026/course/tree/main/lections) |
| Как оценивают, дедлайны, порядок сдачи | [`homework/docs/grading.md`](https://github.com/course-go-autumn-2026/course/blob/main/homework/docs/grading.md) |
| Локальное окружение и утилита `tripgoctl` | [`course-go-autumn-2026/course-infra`](https://github.com/course-go-autumn-2026/course-infra) |

Задания появляются по мере курса, каждое — после своей пары лекций.

## Как сдавать

Ветка `homework/NN` от `main`, pull request в `main` своего форка, ссылка
ментору до дедлайна. Подробно — в `grading.md` репозитория курса.

## Чужие работы

Форки видны всем, включая ваши. Смотреть чужие решения, пока идёт курс, —
плохая идея: одинаковый код виден сразу, а разбираться на защите придётся
самому.

# ЛР 1. HTTP API и PostgreSQL

Принимает HTTP-запросы по контракту, создаёт поездку, отдаёт её и завершает.
Данные лежат в PostgreSQL, схема накатывается миграциями, создание поездки
атомарно пишет в две таблицы в одной транзакции.

## Требования

- Go 1.24+ (проект собирается на `go 1.27.1` из `go.mod`)
- [tripgoctl](https://github.com/course-go-autumn-2026/course-infra) - утилита локального окружения курса
- PostgreSQL - поднимается `tripgoctl` в локальном Kubernetes
- `psql` - для ручной проверки

## Запуск

1. Клонировать репозиторий и перейти в него:

   ```bash
   git clone <URL-репозитория>
   cd go_labs
   ```

2. Поднять локальное окружение и получить адреса:

   ```bash
   tripgoctl cluster start
   tripgoctl environment start
   tripgoctl connect
   ```

   После `environment start` в корне появится `.env` с готовым `DATABASE_URL`.

3. Накатить миграции:

   ```bash
   make migrate
   ```

4. Запустить сервис:

   ```bash
   make run
   ```

   Сервис слушает адрес из `HTTP_ADDR` (по умолчанию `:8080`).

## Переменные окружения

Все параметры — из переменных окружения. Обязательные проверяются на старте:
без них сервис падает с понятной ошибкой. Значения по умолчанию — в
`.env.example`.

| Переменная | Обязательная | По умолчанию | Смысл |
|---|---|---|---|
| `HTTP_ADDR` | да | — | адрес HTTP-сервера, например `:8080` |
| `DATABASE_URL` | да | — | DSN PostgreSQL |
| `LOG_LEVEL` | нет | `info` | уровень логов (`debug`/`info`/`warn`/`error`) |
| `SHUTDOWN_TIMEOUT` | нет | `10s` | бюджет graceful shutdown |
| `DATABASE_MAX_CONNS` | нет | `10` | максимум соединений в пуле |
| `DATABASE_MIN_CONNS` | нет | `2` | минимум соединений в пуле |
| `DATABASE_MAX_CONN_LIFETIME` | нет | `30m` | время жизни соединения |
| `DATABASE_CONNECT_TIMEOUT` | нет | `5s` | таймаут подключения к бд |
| `DATABASE_QUERY_TIMEOUT` | нет | `3s` | таймаут запроса |

`.env` не коммитится (в `.gitignore`). В репозитории лежит `.env.example` со
всеми переменными и безопасными дефолтами.

## Makefile

| Цель | Что делает |
|---|---|
| `make generate` | генерирует код по OpenAPI из `contracts/openapi/trip-service.openapi.yaml` |
| `make migrate` | накатывает миграции через goose |
| `make migrate-down` | откатывает одну миграцию |
| `make migrate-reset` | откатывает все миграции (`down-to 0`) |
| `make run` | запускает сервис локально |

## Проверка

### Liveness и readiness

```bash
curl -i localhost:8080/health

curl -i localhost:8080/ready
```

### Создание поездки

```bash
curl -i -X POST localhost:8080/api/v1/trips \
  -H 'content-type: application/json' \
  -d '{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c",
       "driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5",
       "start_point":{"latitude":59.9398,"longitude":30.3146},
       "end_point":{"latitude":59.9290,"longitude":30.3626},
       "price":1450}'
```

Повторный `POST` на того же водителя вернёт `409 driver_busy` - у водителя уже
есть активная поездка.

### Получение и завершение

```bash
TRIP_ID=<id из ответа>

curl -i localhost:8080/api/v1/trips/$TRIP_ID
# 200

curl -i -X POST localhost:8080/api/v1/trips/$TRIP_ID/finish
# 200, status: completed

curl -i -X POST localhost:8080/api/v1/trips/$TRIP_ID/finish
# 409, {"code":"trip_completed"}
```

### Ошибки

Все ошибки со стабильным полем `code`:

- `400 invalid_request` - не прошла валидация тела или `tripId` не UUID;
- `404 trip_not_found` - поездки с таким `id` нет;
- `409 trip_completed` - попытка завершить уже завершённую поездку;
- `409 driver_busy` - у водителя уже есть активная поездка;
- `500 internal_error` - всё остальное.

## Проверка конкурентности

Создание 20 параллельных запросов на одного водителя — ровно один `201`,
остальные `409`:

```bash
echo '{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c",
       "driver_id":"11111111-1111-1111-1111-111111111111",
       "start_point":{"latitude":59.9,"longitude":30.3},
       "end_point":{"latitude":59.9,"longitude":30.4},
       "price":100}' > /tmp/trip.json

seq 20 | xargs -P20 -I{} curl -s -o /dev/null -w '%{http_code}\n' \
  -X POST localhost:8080/api/v1/trips \
  -H 'content-type: application/json' \
  -d @/tmp/trip.json | sort | uniq -c
#       1 201
#      19 409
```

Конкурентное завершение — один `200`, один `409`:

```bash
seq 2 | xargs -P2 -I{} curl -s -o /dev/null -w '%{http_code}\n' \
  -X POST localhost:8080/api/v1/trips/$TRIP_ID/finish | sort | uniq -c
#       1 200
#       1 409
```
### Конкурентный create с одним Idempotency-Key

5 параллельных запросов с одним ключом на одного водителя — один `201`,
остальные `200`, вторая поездка не создаётся:

```bash
CONC_DRIVER=$(uuidgen)
KEY2=$(uuidgen)
echo "{\"user_id\":\"5cb72c04-7650-45c9-a79b-bcdba0631e0c\",\"driver_id\":\"$CONC_DRIVER\",\"start_point\":{\"latitude\":59.9,\"longitude\":30.3},\"end_point\":{\"latitude\":59.9,\"longitude\":30.4},\"price\":100}" > /tmp/body.json

seq 5 | xargs -P5 -I{} curl -s -o /dev/null -w '%{http_code}\n' \
  -X POST localhost:8080/api/v1/trips \
  -H 'content-type: application/json' \
  -H "Idempotency-Key: $KEY2" \
  -d @/tmp/body.json | sort | uniq -c
#       1 201
#       4 200

psql "$DATABASE_URL" -c "SELECT COUNT(*) FROM trips WHERE driver_id = '$CONC_DRIVER';"
# 1

---

## Решения

### Уровень изоляции транзакций

Выбран **`ReadCommitted`** - уровень по умолчанию в PostgreSQL. Задаётся явно
в `txManager.Do` через `pgx.TxOptions{IsoLevel: pgx.ReadCommitted}`.

Обоснование:

- Запрет двух активных поездок держится **частичным уникальным индексом**
  `trips_one_active_per_driver_uniq`, а не логикой в коде. Под конкурентной
  нагрузкой PostgreSQL сам разведёт два параллельных `INSERT`: один пройдёт,
  второй получит `23505`. Уровень изоляции тут ни при чём.
- Завершение поездки использует **условный** `UPDATE ... WHERE status = 'active'`.
  Это атомарная операция: даже при двух параллельных `finish` только один
  запрос изменит строку, второй получит `RowsAffected = 0`. Этого достаточно
  на `ReadCommitted`.
- `Serializable` не даёт преимуществ для этих сценариев и потребовал бы
  ретраев на каждом конфликте - лишняя сложность без выгоды.

### Менеджер транзакций

Реализован интерфейс:

```go
    type TxManager interface {
        Do(ctx context.Context, fn func(ctx context.Context) error) error
    }
```

Как устроен:

- `Do` открывает транзакцию через `pool.BeginTx` с `IsoLevel: ReadCommitted`;
- кладёт `pgx.Tx` в `context` через приватный ключ `txKey{}`;
- вызывает `fn` с обогащённым контекстом;
- если `fn` вернула `nil` - `COMMIT`;
- если `fn` вернула ошибку - `ROLLBACK`, ошибка пробрасывается наружу;
- если `fn` запаниковала - `ROLLBACK` и `panic` пробрасывается дальше.

**Вложенный `Do`** внутри `Do` видит, что в контексте уже лежит `pgx.Tx`, и
переиспользует её - вторая транзакция не открывается.

**Репозиторий не получает транзакцию аргументом.** Он достаёт исполнителя из
контекста в `exec(ctx)`:

```go
    func (r *TripRepository) exec(ctx context.Context) DBTX {
        if tx, ok := txFromContext(ctx); ok {
            return tx
        }
        return r.pool
    }
```

Если транзакция в контексте есть - работаем через неё; если нет - через пул.
Бизнес-код (`Handlers`) не знает ни про `pgx`, ни про транзакции, ни про пул -
он просто вызывает `txManager.Do` и внутри него методы репозитория.

**Создание поездки атомарно.** `CreateTrip` вызывает один `Do`, внутри которого
два `INSERT`: в `trips` и в `trip_status_history`. Если второй падает - первый
откатывается, «висящей» поездки не остаётся.

**Завершение поездки атомарно.** Аналогично: `Finish` (условный `UPDATE`) и
`AppendStatusChange` — в одном `Do`.

### Запрет двух активных поездок у водителя

Закреплён **в схеме бд** - частичным уникальным индексом:

```sql
CREATE UNIQUE INDEX trips_one_active_per_driver_uniq
    ON trips (driver_id)
    WHERE status = 'active';
```

Индекс уникален **только для строк со `status = 'active'`**. У завершённых
поездок ограничение не действует - их у водителя может быть сколько угодно.

При конкурентных запросах: два параллельных `INSERT` с одним `driver_id` и
`status = 'active'` - один проходит, второй получает `23505` с именем индекса
`trips_one_active_per_driver_uniq`. Репозиторий распознаёт этот код через
`pgconn.PgError` и возвращает доменную ошибку `trip.ErrDriverBusy`, транспорт
переводит её в `409 driver_busy`.

Проверка `SELECT` перед `INSERT` **не годится**: оба параллельных запроса
пройдут проверку до того, как любой из них вставит строку. Ограничение в бд
этого недостатка лишено - оно атомарно на уровне индекса.

### Идемпотентность создания (задание со звёздочкой)

`POST /api/v1/trips` поддерживает заголовок `Idempotency-Key`:

- новый ключ `201`, поездка создаётся;
- тот же ключ + то же тело `200`, возвращается та же поездка;
- тот же ключ + другое тело `409 idempotency_conflict`;
- без заголовка `201` как обычно.

Ключи хранятся в таблице `idempotency_keys` (`key UUID PRIMARY KEY`,
`trip_id`, `request_hash BYTEA`, `created_at`). `INSERT` ключа, `INSERT` поездки
и запись в журнал — в **одной транзакции**. Гонка защищена первичным ключом:
два параллельных запроса с одним ключом — один займёт ключ и создаст поездку,
второй получит `23505` и прочитает уже созданную поездку.

**TTL ключей** - 24 часа с момента создания. По истечении повтор с тем же ключом
считается новым и создаёт новую поездку. В ЛР1 фоновой очистки нет - таблица
растёт; поле `created_at` и индекс по нему уже есть, чтобы реализовать очистку
в следующих работах.

**Почему SAVEPOINT.** В PostgreSQL любая ошибка внутри транзакции
(включая `23505` — нарушение уникальности) переводит её в состояние
`aborted`: до `ROLLBACK` или `ROLLBACK TO SAVEPOINT` ни одна команда
в ней не выполняется — попытка получит `SQLSTATE 25P02`.

Сценарий гонки: два параллельных запроса с одним ключом. Первый занимает
ключ, второй при `INSERT` получает `23505`. Если после этого сразу
попытаться прочитать запись через `SELECT` в той же транзакции — получим
`25P02`, а не запись.

Поэтому перед `INSERT` ключа ставится `SAVEPOINT idem_insert`,
а при `23505` — `ROLLBACK TO SAVEPOINT idem_insert`. После отката
до точки транзакция снова рабочая, `SELECT` в ней выполняется корректно,
и мы возвращаем существующую поездку (`200`).

Вся логика идемпотентности выполняется в одной транзакции: `SAVEPOINT`,
`INSERT` ключа, `INSERT` поездки, запись в журнал, `UPDATE trip_id`,
`COMMIT`. Если что-то падает, откатывается всё, ключ освобождается.

## Docker (задание со звёздочкой)

Сервис собирается многостадийно: компиляция в `golang:1.24-alpine`,
запуск — в `gcr.io/distroless/static-debian12:nonroot`. В итоговом образе нет
исходников, go toolchain и shell; процесс работает не от root.

### Сборка

```bash
docker build -t trip-service:local .
```

---

## Структура репозитория

```
    cmd/trip-service/        composition root, единственный main
    internal/
      api/                   сгенерированный по OpenAPI код (не правится руками)
      config/                конфигурация из env
      domain/trip/           доменные типы и ошибки
      repository/postgres/   pgxpool, TripRepository, TxManager
      transport/http/        HTTP-сервер на chi, хендлеры, problem+json
    migrations/              SQL-миграции goose
    contracts/               контракты курса (не редактируются)
    config/                  oapi-codegen.yaml, trip-service.env.example
    Makefile
    .env.example
    README.md
```

## Кодогенерация

Типы и серверные интерфейсы генерируются из OpenAPI:

```bash
    make generate
```

Конфигурация - в `config/oapi-codegen.yaml`, выход - `internal/api/server.gen.go`.
Файл коммитится, но руками не правится: при следующем `make generate` правки
затрутся.

## Тесты

Тесты появятся в лабораторной работе 2. На ЛР1 цель `make test` ещё не
реализована.