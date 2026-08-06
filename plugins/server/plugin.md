# Плагин server — HTTP/JSON-RPC/WS/SSE транспорт на Fiber

## Назначение

Плагин генерирует Go-пакет HTTP-сервера по контрактам `@tg`: REST, JSON-RPC 2.0 (включая batch), WebSocket и SSE. Вы реализуете интерфейсы контрактов, регистрируете их в `transport.New` и запускаете Fiber.

Полный синтаксис аннотаций: `tg plugin doc astg`.

## Когда использовать

- Нужен серверный transport из тех же контрактов, что и клиенты/Swagger.
- Контракты помечены `http-server`, `jsonRPC-server`, `ws-server` и/или `sse-server`.
- Не используйте для `@tg kafka` — это `kafka-pub-go` / `kafka-sub-go`.
- Не правьте сгенерированный пакет вручную: меняйте контракты и перегенерируйте.

## Команда и параметры

```bash
tg server -o transport
# или
tg server --out transport --contracts-dir=contracts --contracts=UserService,OrderService
```

| Параметр | Короткая форма | Обязательный | Описание |
|----------|----------------|--------------|----------|
| `out` | `-o` | да | Каталог сгенерированного пакета (например `transport`) |
| `contracts-dir` | — | нет | Каталог контрактов относительно корня проекта (по умолчанию `contracts`) |
| `contracts` | — | нет | Имена контрактов через запятую; без фильтра — все HTTP-семейства |

Фильтр `--contracts` режет копию модели у генератора. Полный snapshot в локальной БД astg не урезается.

## Что генерируется

### Матрица: аннотации → артефакты

| На контракте | Появится |
|--------------|----------|
| `http-server` | REST handlers и маршруты Fiber |
| `jsonRPC-server` | RPC endpoint + batch (`POST /` и scoped mounts по `http-prefix`) |
| оба | одна реализация сервиса на обоих транспортах |
| `ws-server` + `stream=…` | WebSocket endpoint (multiplex JSON-RPC stream) |
| `sse-server` + `stream=server` | SSE POST endpoint (`text/event-stream`) |
| `kafka` | ничего (другой плагин) |
| `log` / `metrics` / `trace` | методы `WithLog` / `WithMetrics` / `WithTrace` на сервере |
| `security` с `http:bearer` | опция `Auth(...)`, subject в `srvctx.GetSubject` |

### Ключевые файлы и API

В `--out` появляется Go-пакет (имена зависят от проекта), которым вы пользуетесь:

| Поверхность | Назначение |
|-------------|------------|
| `New(log, options...)` | конструктор `*Server` |
| `{Contract}(impl)` | Option: регистрация реализации (имя = имя интерфейса) |
| `Fiber()` | `*fiber.App` для `Listen` |
| `WithLog` / `WithMetrics` / `WithTrace` | observability (если аннотации есть) |
| `Shutdown()` / `ShutdownWithTimeout(timeout)` | graceful stop HTTP + metrics (`Shutdown` = default 30s) |
| `ServeHealth` / `ServeMetrics` | отдельные HTTP-серверы |
| `{Contract}().WithErrorHandler` | кастом ошибок REST (только при `http-server`) |

Перед генерацией старые сгенерированные файлы в каталоге очищаются по маркеру Tool Gateway.

## Аннотации и их влияние

Полный синтаксис: `tg plugin doc astg`. Ниже — эффект **на этот плагин**.

### Транспорт и маршруты

| Аннотация | Уровень | Эффект на server |
|-----------|---------|------------------|
| `http-server` | iface | REST handlers; нужны `http-method` / `http-path` на методах |
| `jsonRPC-server` | iface | JSON-RPC: метод `{contract}.{method}` (camelCase); batch |
| `http-prefix` | пакет / iface | префикс путей REST и mount JSON-RPC; batch ещё на `POST /` |
| `http-method`, `http-path` | method | маршрут Fiber; `:param` → path args |
| `ws-server` | iface | WS upgrade endpoint |
| `sse-server` | iface | SSE endpoint |
| `stream=server\|client\|bidi` | method | направление потока; для SSE обычно `server` |
| `ws-path` / `sse-path` | method | путь stream-endpoint (с `:param`) |
| `http-success` | method | HTTP-код успешного REST-ответа |
| `requestContentType` / `responseContentType` | method | Content-Type тела |
| `http-multipart`, `http-part-name`, `http-part-content` | method / arg | multipart вместо одного raw body |
| `enableInlineSingle` | method | одно возвращаемое значение без обёртки-объекта в wire |
| `log` / `metrics` / `trace` | пакет / iface | доступны `WithLog` / `WithMetrics` / `WithTrace` |
| `log-skip` | arg / result / sub | поле не пишется в viewer-логи |
| `security` (`http:bearer…`) | пакет | включает runtime `Auth`; JWKS через RFC 8414 или `\|jwks=` |

### Режимы `http-headers` / `http-cookies` / `http-args`

Формат: `arg|key` или `arg|key|mode`. Mode: `explicit` | `implicit` | `body` (default `body`).

| Mode | На сервере |
|------|------------|
| `explicit` | значение читается/пишется только через header/cookie/query/path, в тело не дублируется |
| `implicit` | как explicit на wire; значение часто из middleware/контекста (клиенты могут скрыть аргумент) |
| `body` | основное значение в теле; mapping — overlay/override (в JSON-RPC batch — через snapshot overlay в context) |

Для **SSE** headers/cookies/args берутся из HTTP-запроса метода. Для **WS** — из upgrade (connection-scoped на весь multiplex). Браузерный клиент может дублировать headers в query — сервер читает query как fallback.

### Преобразование header/cookie/query

Единый конвейер для HTTP и JSON-RPC (меняется только источник строки):

- встроенные скаляры — `strconv`;
- именованные alias на скаляр — по базовому типу;
- слайсы в header/query — **CSV** (`join(",")` + поэлементный parse);
- иначе — JSON-совместимое преобразование из строки.

По умолчанию **lenient**: неудачный parse → zero-value. Для строгой валидации задайте у доменного типа `Parse(string)` / `Parse{TypeName}(string)` с возвратом `(T)` / `(T, error)` (или указатель).

### Файлы и multipart

| Контракт | Поведение сервера |
|----------|-------------------|
| один `io.Reader` | тело запроса потоком |
| один `io.ReadCloser` | тело ответа потоком |
| несколько reader/closer или `http-multipart` | `multipart/form-data` |
| несколько `io.Reader` в multipart | **строго по порядку** частей до EOF; параллель/чтение «вперёд» — ошибка |

`io.Reader` / `io.ReadCloser` допустимы только при `http-server`.

## Использование

### Базовый запуск

```go
srv := transport.New(slog.Default(), transport.UserService(impl))
srv.WithLog()
srv.WithMetrics()
if err := srv.Fiber().Listen(":8080"); err != nil {
    log.Fatal(err)
}
```

### Опции `New`

| Option | Когда / что |
|--------|-------------|
| `{Contract}(impl)` | регистрация реализации |
| `Service(svc)` | произвольный `ServiceRoute` |
| `SetFiberCfg` | конфиг Fiber (`Immutable`, `StreamRequestBody` и др. форсируются) |
| `SetReadBufferSize` / `SetWriteBufferSize` / `MaxBodySize` | буферы и лимит тела |
| `ReadTimeout` / `WriteTimeout` | таймауты |
| `MaxBatchSize` / `MaxBatchWorkers` | JSON-RPC batch (default 100 / 10) |
| `MaxOpenStreams(n)` | WS: лимит multiplex stream на соединение (default 64) |
| `MaxWSConnections(n)` | WS: лимит соединений (default **512**; `0` = без лимита); при насыщении `503` до upgrade |
| `MaxSSEConnections(n)` | SSE: лимит соединений (default **512**; `0` = off); `503` до parse/stream |
| `MaxWSMessageSize` | лимит размера WS-кадра |
| `StreamWriteTimeout` | write deadline для stream |
| `SetSSEHeartbeat` | heartbeat SSE |
| `MaxClientIDs(n)` | cardinality `client_id` после normalize (default 64; `0` = off → overflow `other`) |
| `DurationBuckets` / `LatencyBuckets` | бакеты гистограмм **до** `WithMetrics()` |
| `WithRequestID` / `WithHeader` | request metadata |
| `Auth(opts...)` | bearer JWT fail-closed |
| `Use(...)` | Fiber middleware |

### Auth

При `@tg security` с `http:bearer`:

```go
transport.Auth(
    transport.AuthorizationServer(asURL),
    transport.Resource(resourceURL),
    transport.Audience(aud...),
    // опционально: ClientHTTP, ConfigTLS, Bind, Allow, Verify
)
```

- Subject после Verify: `srvctx.GetSubject`.
- `missing`/`invalid` → 401 + `WWW-Authenticate`; отказ `Allow` → 403 без challenge.
- Неполная AS/Audience отклоняет на `Verify`, не на `New`.
- При одновременном `ClientHTTP` и `ConfigTLS` побеждает `ClientHTTP` (TLS на custom client).
- JWKS: RFC 8414 от AS или `|jwks=` в аннотации.

### Health и shutdown

```go
hs := transport.ServeHealth(log, "/health", ":8081",
    transport.WithResponse(map[string]string{"status": "ok"}),
)
defer hs.Stop()

ready := transport.ServeHealth(log, "/ready", ":8082",
    transport.WithCheck(func(ctx context.Context) (err error) {
        return db.PingContext(ctx)
    }),
)
defer ready.Stop()

// остановка API+metrics:
_ = srv.ShutdownWithTimeout(10 * time.Second)
// или с дефолтным budget 30s:
_ = srv.Shutdown()
```

- **`ServeHealth(log, path, address, opts...)`** — отдельный сервер; опции `WithResponse` / `WithCheck` (probe timeout 5s → `503`). Без `WithResponse` тело успеха — `"ok"`.
- **`Shutdown` / `ShutdownWithTimeout`** — graceful stop; `Shutdown()` делегирует в `ShutdownWithTimeout(defaultShutdownTimeout)` (30s); HTTP и metrics гасятся параллельно (`errors.Join`).

### JSON-RPC batch и prefix

- Общий batch: **`POST /`**.
- Дополнительно — mount’ы по `http-prefix` (и предкам) со scoped картой методов.
- Headers/cookies в batch берутся из overlay контекста на входе batch/method (не из «чужого» Fiber Ctx).

### Метрики и identity

`X-Client-Id` нормализуется (trim, max 64, charset); пустой → `unknown`, мусор → `invalid`. Лейбл `path` — шаблон маршрута Fiber (`unmatched` при промахе). Смена `DurationBuckets`/`LatencyBuckets` ломает дашборды, завязанные на `le`.

Экспорт метрик — отдельный сервер:

```go
srv.WithMetrics()
_ = srv.ServeMetrics(log, "/metrics", ":9090")
// ServeMetrics тоже останавливается в ShutdownWithTimeout
```

### Streams (WS / SSE)

| Аспект | WS | SSE |
|--------|----|-----|
| Контракт | `ws-server` + `stream=…` | `sse-server` + `stream=server` |
| Соединение | одно upgrade, multiplex RPC-stream | один HTTP-запрос на server-stream |
| Лимит соединений | `MaxWSConnections` (512 / `0`) | `MaxSSEConnections` (512 / `0`) |
| Лимит stream | `MaxOpenStreams` (64 на соединение) | — |
| Размер / deadline | `MaxWSMessageSize`, `StreamWriteTimeout` | `StreamWriteTimeout`, `SetSSEHeartbeat` |
| Headers/args | из upgrade, connection-scoped | из HTTP-запроса метода |
| Wire | JSON-RPC notifications `$/stream`, `$/stream.end`, `$/cancel` | `text/event-stream` |

Повторный JSON-RPC `id` среди открытых stream → ошибка duplicate stream id. Переполнение per-stream incoming buffer закрывает этот stream, сессия остальных продолжается.

### Ошибки REST

Для контрактов с `http-server`:

```go
srv.UserService().WithErrorHandler(func(err error) error {
    // обернуть/подменить до кодирования ответа
    return err
})
```

Только до `Listen`. У контрактов «только JSON-RPC» такого accessor нет — RPC errors идут в JSON-RPC error object.

### Типичные сценарии запуска

**Только REST** (`http-server` + `http-prefix` + method/path):

```go
srv := transport.New(log, transport.UserService(impl))
_ = srv.Fiber().Listen(":8080") // GET /api/v1/users/:id
```

**Только JSON-RPC** (`jsonRPC-server`):

```go
// POST /userService  method=userService.getUser
// batch: POST /  и scoped mounts по prefix
```

**HTTP + RPC** на одном iface — одна реализация, два wire.

**Graceful stop:**

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
go func() { _ = srv.Fiber().Listen(":8080") }()
<-ctx.Done()
_ = srv.ShutdownWithTimeout(10 * time.Second)
```

### Enums и CSV

- Именованный скаляр с typed `const` (≥2) попадает в модель enums без обязательного `@tg enums` на поле.
- Слайсы в header/query на wire — CSV; сервер парсит поэлементно.

## На что обратить внимание

1. **Владение кодом**: правьте `contracts/` и реализации сервисов; `transport/` только регенерируйте.
2. **`--contracts`** фильтрует генерацию, но не «половину проекта» в БД.
3. **Kafka** в этом плагине не появляется.
4. **`implicit` args** на клиентах могут исчезнуть из сигнатуры — на сервере wire всё равно нужен.
5. **Lenient parse** header/query: «тихий» zero-value → задайте `Parse(string)`.
6. **CSV slices** в header/query — не Go `fmt` slice; клиенты должны слать CSV.
7. **Multipart** — только последовательное чтение частей.
8. **Лимиты WS/SSE** по умолчанию 512 соединений; `0` — явно без лимита.
9. **Observability** на чистом WS/SSE без HTTP/JSON-RPC флагов не ждите тех же Fiber-хуков.
10. После смены контракта пересоберите модуль и smoke-test маршрутов/stream/cancel.

## Ограничения

1. Первый аргумент метода — `context.Context`, последний результат — `error`; остальные имена обязательны.
2. Только экспортируемые интерфейсы.
3. REST требует `http-method` + `http-path`; при `http-prefix` в path указывайте ведущий `/`.
4. JSON-RPC params только в теле (path params не используются как RPC-параметры).
5. `io.Reader` / `io.ReadCloser` — только с `http-server`.
6. OpenAPI/клиенты — отдельные плагины; server не пишет Swagger-файл.

## Связь с другими плагинами

| Плагин | Связь |
|--------|-------|
| `astg` / `astg-json` | модель контрактов; проверка resolved annotations |
| `init-go` | scaffold с первым `transport/` |
| `client-go` / `client-ts` | совместимые клиенты по тем же аннотациям |
| `swagger` | OpenAPI по HTTP-семейству |
| `kafka-pub-go` / `kafka-sub-go` | события Kafka (не Fiber) |
