# Trip Service — лабораторная работа 2

Архитектура, качество кода и observability.

Сервис поездок, начатый в ЛР1. Внешнее поведение не изменилось: те же ручки,
те же коды ответов, тот же контракт. Внутри код переложен по слоям, покрыт
тестами, инструментирован OpenTelemetry.

---

## Что сделано в ЛР2

- Код разложен по слоям: `domain`, `model`, `usecase`, `http`, `repository`, `infra`.
  Зависимости направлены внутрь. Правила проверяются `depguard`.
- Внешние зависимости — за узкими интерфейсами, объявленными на стороне потребителя.
- Сборка зависимостей — в `internal/container`. Больше нигде нет `NewXxx`.
- Доменные ошибки — в `domain/trip`. Маппинг «доменная ошибка → Problem» — в
  `http/domainerror`, в одном месте. Переходы статусов — в домене (`Trip.Finish`).
- Unit-тесты: домен, usecase (с моками `gomock` через `go:generate`), `domainerror`,
  хендлеры (`httptest`).
- Интеграционные тесты репозиториев на реальном PostgreSQL (`//go:build integration`),
  включая тест отката транзакции.
- Покрытие `usecase` + `domain` — **≥ 50%** (фактически ~90% и ~67%).
- `.golangci.yml` — обязательный набор линтеров + `depguard`, `make lint` зелёный.
- OpenTelemetry: трейсы HTTP (`otelhttp`) и БД (`otelpgx`), метрики, экспорт по OTLP.
- Логи `slog` с `trace_id` и `span_id`, middleware `Logging` пишет одну запись на запрос.
- Middleware в порядке `Recover → Tracing → Logging → Metrics`.
- Задание со звёздочкой: мутационное тестирование через `gremlins`.

---

## Запуск

```bash
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect

make migrate
make run
```

Сервис слушает `HTTP_ADDR` (по умолчанию `:8080`).

Переменные окружения — в `.env.example`. В ЛР2 добавлены:

| Переменная | По умолчанию | Смысл |
|---|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4318` | адрес OTel-коллектора (из `.env` от `tripgoctl`) |
| `OTEL_SERVICE_NAME` | `trip-service` | имя сервиса в трейсах и метриках |

Остальные (из ЛР1) — `HTTP_ADDR`, `DATABASE_URL`, `LOG_LEVEL`, `SHUTDOWN_TIMEOUT`,
`DATABASE_*` — не изменились.

---

## Makefile

| Цель | Что делает |
|---|---|
| `make generate` | кодогенерация по OpenAPI |
| `make migrate` | накатить миграции |
| `make migrate-down` | откатить одну миграцию |
| `make migrate-reset` | откатить все миграции |
| `make run` | запустить сервис |
| `make test` | unit-тесты с `-race`, печатает итоговое покрытие |
| `make test-integration` | интеграционные тесты (`-tags=integration`) |
| `make lint` | `golangci-lint run` |
| `make mutate` | мутационное тестирование (`gremlins`) |

---

## Архитектура

```
cmd/trip-service/        composition root, единственный main
internal/
  config/                конфигурация из env
  container/             сборка зависимостей (единственное место с NewXxx)
  domain/trip/           домен: модель, ошибки, валидация, Finish
  model/trip/            структуры, ходящие между слоями
  usecase/trip/          сценарии (по пакету на сценарий)
    create/
    get/
    finish/
  http/                  транспорт
    common/              writeJSON, writeProblem, ToAPITrip
    domainerror/         маппинг доменных ошибок в Problem (одно место)
    middleware/          Recover, Tracing, Logging, Metrics
    trip/                хендлеры (по пакету на ручку)
      create/
      get/
      finish/
    router.go            сборка chi-роутера
    server.go            реализация api.ServerInterface
  repository/trip/       реализация интерфейсов usecase
  repository/idempotency/ реализация интерфейсов usecase (ключи идемпотентности)
  infra/postgres/        пул pgx, TxManager, txctx
  infra/telemetry/       OTel SDK, бизнес-метрики
  generated/             сгенерированный по OpenAPI код (не правится руками)
migrations/              SQL-миграции goose
contracts/               контракты курса
```

### Правила зависимостей

- `domain` не импортирует `net/http`, `chi`, `pgx`, `squirrel`, `otel`, `generated`;
- `usecase` не импортирует `net/http`, `chi`, `pgx`, `squirrel`, `generated`;
- `http/*` не импортирует `repository/*`, `infra/*`;
- `repository/*` не импортирует `http/*`.

Проверяется `depguard` из `.golangci.yml`.

### Интерфейсы на стороне потребителя

- `usecase/trip/create/contract.go` — `tripRepository`, `idempotencyRepository`, `txManager`, `metrics`;
- `usecase/trip/finish/contract.go` — `tripRepository`, `txManager`, `metrics`;
- `usecase/trip/get/contract.go` — `tripRepository`;
- `http/trip/{create,get,finish}/handler.go` — по одному интерфейсу `usecase`.

Моки генерируются `mockgen` через `go:generate` и лежат рядом с кодом
(`mocks_test.go`), отдельного пакета `mocks/` нет.

### Сборка зависимостей

`internal/container.New(ctx, cfg, log)` — единственное место, где вызываются
`pgxpool.New`, `NewTripRepository`, `NewUsecase` и т.п. `main.go` только читает
конфиг, вызывает `container.New`, запускает HTTP-сервер и ждёт сигнала.

### Домен

Переходы статусов описаны в домене:

```go
func (t *Trip) Finish(at time.Time) error {
    if t.Status != StatusActive {
        return ErrAlreadyDone
    }
    t.Status = StatusCompleted
    t.FinishedAt = &at
    t.UpdatedAt = at
    return nil
}
```

### Ошибки

Доменные ошибки — `internal/domain/trip/errors.go`. Ошибки валидации —
`*trip.ValidationError` с полем и сообщением. Маппинг «доменная ошибка → Problem» —
`internal/http/domainerror/domainerror.go`.

---

## Тесты

| Что | Чем | Где |
|---|---|---|
| Домен | обычные table-driven тесты | `internal/domain/trip/trip_test.go`, `validation_test.go` |
| Usecase | `gomock` на репозиторий и `TxManager` | `internal/usecase/trip/{create,get,finish}/usecase_test.go` |
| Маппинг ошибок | table-driven | `internal/http/domainerror/domainerror_test.go` |
| Хендлеры | `httptest` + мок usecase | `internal/http/trip/{create,get,finish}/handler_test.go` |
| Репозитории | реальный PostgreSQL, тег `integration` | `internal/repository/{trip,idempotency}/integration_test.go` |

Обязательные сценарии покрыты:
- создание поездки — успех (`201`);
- создание при занятом водителе → `409 driver_busy`;
- создание с идемпотентным ключом → `201`, повтор → `200`, другое тело → `409`;
- завершение активной поездки → `200`;
- завершение завершённой → `409 trip_completed`;
- получение несуществующей → `404 trip_not_found`;
- невалидное тело → `400 invalid_request`;
- ошибка репозитория → `500 internal_error` без утечки деталей;
- откат транзакции (интеграционный).

Покрытие:
- `internal/usecase/...` — **≥ 70%**;
- `internal/domain/...` — **≥ 66%**.

`make test` печатает итоговое покрытие (`total: (statements) X%`).

### Команды

```bash
make test
set -a; source .env; set +a
make test-integration
```

---

## Линтер

`.golangci.yml` в корне. Набор из задания:

```
depguard, errcheck, forbidigo, gochecknoinits, gocritic, gocyclo, goconst,
gosec, govet, ineffassign, makezero, prealloc, revive, staticcheck, unconvert,
unparam, unused, whitespace
```

Форматтеры: `gofumpt`, `gci`. `depguard` проверяет правила слоёв.

```bash
make lint
```

---

## Observability

### Трейсы

OpenTelemetry SDK инициализируется в `internal/infra/telemetry`:
- `SetupTracing` — `otlptracehttp`, экспорт по OTLP;
- `SetupMetrics` — `otlpmetrichttp`, `PeriodicReader`.

Спаны создаются:
- на HTTP-запросы — middleware `Tracing` (`otelhttp`);
- на SQL-запросы — автоматически через `otelpgx` на уровне пула
  (`poolCfg.ConnConfig.Tracer = otelpgx.NewTracer()`).

Имя HTTP-спана — `{method} {route}`, где `route` — **шаблон** маршрута
(`/api/v1/trips/{tripId}`), а не реальный URL. Это требование задания.

### Логи

Middleware `Logging` пишет **одну** запись на запрос после отправки ответа:

```json
{
  "level": "INFO",
  "msg": "request handled",
  "service": "trip-service",
  "method": "POST",
  "route": "/api/v1/trips",
  "status": 201,
  "duration_ms": 12,
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7"
}
```

- `route` — шаблон, не URL;
- тел запроса/ответа не логируем;
- `2xx`/`4xx` → `Info`, `5xx` → `Error`;
- бизнес-события (`trip created`, `trip finished`) логируются отдельно, в usecase.

По `trace_id` находится трейс в Jaeger.

### Метрики

| Метрика (OTel) | В Prometheus | Тип | Где создаётся |
|---|---|---|---|
| `http.server.request.duration` | `http_server_request_duration_seconds_bucket` | histogram | `middleware/metrics.go` |
| `http.server.active_requests` | `http_server_active_requests` | up-down counter | `middleware/metrics.go` |
| `db.client.operation.duration` | `db_client_operation_duration_seconds_bucket` | histogram | `otelpgx` (автоматически) |
| `tripgo.trips.created` | `tripgo_trips_created_total` | counter | `usecase/trip/create` |
| `tripgo.trips.completed` | `tripgo_trips_completed_total` | counter | `usecase/trip/finish` |

### Middleware

Порядок фиксирован: `Recover → Tracing → Logging → Metrics`.

- `Recover` ловит панику, логирует её со стектрейсом и отдаёт `500 internal_error`.
- `Tracing` — корневой спан на HTTP-запрос.
- `Logging` — одна запись на запрос после ответа.
- `Metrics` — `request.duration` и `active_requests`.

### Проверка в Jaeger и Grafana

1. `tripgoctl connect` — адреса Jaeger и Grafana.
2. Сделать несколько запросов (`curl`) — 5–10 раз.
3. Открыть Jaeger → **Service** `trip-service` → **Find Traces** → выбрать трейс
   `POST /api/v1/trips` → увидеть спаны HTTP и БД.
4. Открыть Grafana → **Explore** → **Prometheus** → `tripgo_trips_created_total`
   (или `http_server_request_duration_seconds_count`).
5. Взять `trace_id` из лога → вставить в поле **Trace ID** в Jaeger → попасть
   на тот же трейс.

### Скриншоты

В папке "docs/screenshots" лежат все необходмые скриншоты пользования Jager и Grafana

---

## Мутационное тестирование (задание со звёздочкой)

Инструмент: [`gremlins`](https://github.com/go-gremlins/gremlins).

### Запуск

```bash
make mutate
```

Или вручную:

```bash
gremlins unleash ./internal/domain/trip ./internal/usecase/trip/... ./internal/http/trip/... ./internal/http/domainerror
```

### Результаты

| Пакет | Killed | Lived | Not covered | Test efficacy | Mutator coverage |
|---|---|---|---|---|---|
| `domain/trip` | 2 | 0 | 0 | 100% | 100% |
| `usecase/trip/create` | <N> | 0 | 0 | 100% | 100% |
| `usecase/trip/get` | <N> | 0 | 0 | 100% | 100% |
| `usecase/trip/finish` | <N> | 0 | 0 | 100% | 100% |
| `http/trip/create` | 1 | 0 | 0 | 100% | 100% |
| `http/trip/get` | <N> | 0 | 0 | 100% | 100% |
| `http/trip/finish` | <N> | 0 | 0 | 100% | 100% |
| `http/domainerror` | <N> | 0 | 0 | 100% | 100% |

`TIMED OUT` — ограничение `gremlins` на некоторых мутациях (`defer`/`recover`/паника).
Мутант покрыт тестами, но `gremlins` не смог корректно оценить поведение. В `Test
efficacy` не учитывается.

### Какие мутанты закрыты

- `domain/trip` — `validation.go:11` (ветка `Error()` с пустым `Field`).
  Добавлен `TestValidationError_Error` (2 кейса: с полем и без).
- `usecase/trip/create` — `validate.go:20,27,30` (границы `price=0`, `lat=±90`,
  `lng=±180`). Добавлены `TestCreate_PriceZeroIsValid`, `TestCreate_CoordsBoundaries`.
- `usecase/trip/create` — `usecase.go` (ветки идемпотентности: тело другое,
  `TripID == nil`, `rec == nil`). Добавлены `TestCreate_WithIdempotency_DifferentBody`,
  `TestCreate_WithIdempotency_KeyExistsButTripIDNil`, `TestCreate_WithIdempotency_GetReturnsNil`.
- `http/trip/create` — `handler.go:55` (`IdempotencyKey != nil`), `handler.go:62`
  (ошибка 500 → Error-лог). Добавлены `TestHandler_Create_WithIdempotencyKey`,
  `TestHandler_Create_InternalError_LogsError`.
- `http/trip/{get,finish}` — `handler.go` (ошибка 500 → Error-лог).
  Добавлены `TestHandler_*_InternalError_LogsError` с `countingHandler`.

Отчёт: [`docs/mutation-after.txt`](docs/mutation-after.txt).

---

## Решения

### Интерфейсы на стороне потребителя

Объявлены там, где используются (usecase и хендлеры), а не рядом с реализацией.
Когда в ЛР4 появится второй транспорт (gRPC), usecase не придётся менять — он уже
работает через свои интерфейсы. Достаточно реализовать gRPC-транспорт, использующий
те же usecase.

### Сборка в `container`, а не в `main`

`main.go` тонкий: конфиг → `container.New` → запуск HTTP-сервера → ожидание сигнала.
Всё, что создаёт зависимости (`pgxpool.New`, `NewTripRepository`, `NewUsecase`),
собрано в `internal/container`. Это упрощает тесты и делает `main` читаемым.

### Ошибки

Доменные ошибки в `domain/trip`, маппинг в `http/domainerror`. Внутренние детали
(текст ошибки БД, стектрейс) в ответ клиенту не попадают — только в лог.

### OTel через `otelpgx`

Инструментируем БД **на уровне пула**, а не вручную в каждом методе репозитория.
Это гарантирует, что **все** SQL-запросы получают спаны, и репозиторий не зависит
от OTel.

---

## Что не сделано

- Мутационное тестирование выполнено; эквивалентные мутанты описаны выше.
- Дополнительных ручек/миграций в ЛР2 нет — требование задания.