# Плагин client-go — типобезопасный Go SDK

## Назначение

Плагин генерирует Go-клиент по контрактам `@tg`: JSON-RPC 2.0, REST, WebSocket и SSE. Один `Client` даёт доступ ко всем выбранным контрактам через методы вида `UserService()`.

Полный синтаксис аннотаций: `tg plugin doc astg`.

## Когда использовать

- Нужен SDK на Go к API, описанному tgp-контрактами.
- Контракты с `jsonRPC-server`, `http-server`, `ws-server` и/или `sse-server`.
- Не для Kafka и не как замена `server` transport.
- `--out` должен лежать **внутри** Go-модуля (`go.mod` обязателен).

## Команда и параметры

```bash
tg client go --out ./client
# опционально:
tg client go --out ./client --contracts=UserService,OrderService --doc-file=./client/readme.md
tg client go --out ./client --no-doc
```

| Параметр | Короткая форма | Обязательный | Описание |
|----------|----------------|--------------|----------|
| `out` | **нет** (только `--out`) | да | Каталог клиента внутри модуля |
| `contracts-dir` | — | нет | Каталог контрактов (default `contracts`) |
| `contracts` | — | нет | Фильтр имён через запятую (без `!Name`) |
| `doc-file` | — | нет | Путь к документации (default `<out>/readme.md`) |
| `no-doc` | — | нет | Не генерировать readme |

Перед записью старые сгенерированные файлы в `out` очищаются.

## Что генерируется

### Матрица: возможности контракта → API клиента

| На контракте / флаги | Появится в клиенте |
|----------------------|--------------------|
| `jsonRPC-server` | методы на `c.{Contract}()`, builders `Req*`, пакет `jsonrpc/` |
| `http-server` | REST-методы на том же accessor |
| оба | RPC и HTTP на одном `UserService()` |
| JSON-RPC в модели | `batch.go`, `Client.Batch` |
| `ws-server` + `stream` | channel-based WS stream-методы; файл `stream.go` |
| `sse-server` + `stream=server` | server-stream; при обоих транспортах — вариант `*SSE` |
| `metrics` | `metrics.go`, опция `WithMetrics` |
| `security` с `http:bearer` / `oauth2:` | пакет `token/`, `Auth(...)`, `TokenSource` |
| form / multipart | подпакет `schema/` |
| `kafka` | ничего |
| без `--no-doc` | `readme.md` |

### Инвентарь файлов

| Файл / пакет | Когда |
|--------------|-------|
| `client.go`, `options.go`, `error.go`, `version.go` | всегда |
| `batch.go`, `jsonrpc/` | есть JSON-RPC |
| `dto/` | типы обмена |
| `{contract}-client.go`, `{contract}-exchange.go` | на каждый контракт |
| `stream.go` | есть WS/SSE stream |
| `token/` | security bearer/oauth2 |
| `metrics.go` | `@tg metrics` |
| `schema/` | form/multipart HTTP |
| `readme.md` | если не `--no-doc` |

## Аннотации и их влияние

Полный синтаксис: `tg plugin doc astg`.

| Аннотация | Уровень | Эффект на client-go |
|-----------|---------|---------------------|
| `jsonRPC-server` | iface | RPC-методы; имя `userService.getUser` |
| `http-server` | iface | REST-методы |
| `http-prefix` | пакет / iface | входит в заявленный route; при сборке URL сворачивается с совпадающим хвостом endpoint |
| `http-method`, `http-path`, `http-args` | method | REST route и query/path |
| `http-headers`, `http-cookies` | method | header/cookie; режимы mapping |
| `ws-server` / `sse-server` / `stream` / `ws-path` / `sse-path` | iface / method | stream API на каналах |
| `enableInlineSingle` | method | одно значение в result без обёртки |
| `http-multipart`, `http-part-*` | method / arg | multipart upload/download |
| `metrics` | пакет / iface | `WithMetrics`, `LatencyBuckets`, registry |
| `log-skip` | arg / result | меньше шума при `LogRequest` (с учётом redaction) |
| `security` | пакет | генерация `token` + `Auth` / `TokenSource` |
| typed enums / `@tg enums` | type / field | константы в клиенте |
| `kafka` | iface | игнор |

### Режимы mapping → сигнатура клиента

| Mode | Сигнатура метода | Wire |
|------|------------------|------|
| `explicit` | аргумент **есть** | уходит в header/cookie/query |
| `implicit` | аргумент **может исчезнуть** | значение из `Headers`/контекста/опций |
| `body` (default) | в теле JSON | mapping не делает отдельный HTTP-параметр |

### Endpoint и prefix

`New(endpoint)` принимает origin или URL с path (`https://host`, `https://host/cli`, `https://host/api/v1`). REST/SSE/WS склеивают path endpoint с заявленным route (`http-prefix` + `http-path`): совпадающие сегменты на стыке сворачиваются (нет `/api/v1/api/v1`, mount вроде `/cli` сохраняется). Query endpoint для REST/SSE/WS сбрасывается. JSON-RPC ходит в сам `endpoint`.

## Использование

### Базовый вызов

```go
cli := client.New("https://api.example.com", client.Name("orders-worker"))
user, err := cli.UserService().GetUser(ctx, id)
```

### Опции

| Option | Назначение |
|--------|------------|
| `Name` | identity / `X-Client-Id` и метки метрик |
| `Headers("X-Request-ID", …)` | копировать значения из `context.Value` по тем же ключам |
| `ConfigTLS` / `ClientHTTP` / `Transport` | сеть |
| `BeforeRequest` / `AfterRequest` | хуки |
| `Auth(...)` | app2app / Resource tokens (при security) |
| `TokenSource(src)` | готовый Bearer без Resource discovery (отключает дефолт Auth transport) |
| `DecodeError` / `DecodeHTTPError` | RPC vs REST ошибки |
| `LogRequest` / `LogOnError` | slog; `Authorization` в curl → `Bearer ***` |
| `WithMetrics` / `LatencyBuckets` | Prometheus (отдельный registry) |
| `ToCurl` | отладочный curl |

### Auth и пакет `token`

Генерируется только при `@tg security` с `http:bearer` или `oauth2:`:

```go
cli := client.New(apiURL,
    client.Auth(
        client.TokenURL(tokenURL),
        client.Assertion(sub, kid, pem), // jwt-bearer → Resource + Exchange
        client.TokenRetry(client.MaxAttempts(8), client.Backoff(50*time.Millisecond, 2*time.Second)),
    ),
)
// или полный override obtain:
client.Auth(client.ResourceTokens(obtain))
// или готовый source:
client.TokenSource(src)
```

Поведение:

- skew по умолчанию ~10% TTL (`SkewFraction` / `SkewMin` / `SkewMax` / `EarlyExpiry`);
- на API **401** — один re-obtain Resource + replay, если тело rewindable;
- иначе — `token.ErrBodyNotRewindable`;
- retry IdP/metadata (429/5xx/network) через `TokenRetry`; resource API 5xx **не** ретраятся этим механизмом;
- helpers: `token.JWTBearer`, `Exchange`, `Resource`, `WellKnownURL`, `FetchMetadata`.

### Batch, streams, файлы

**Batch** (только при JSON-RPC):

```go
cli.Batch(ctx,
    userService.ReqGetUser(nil, "1"),
    userService.ReqGetUser(func(u User, err error) { /* … */ }, "2"),
)
```

**Streams**: server/client/bidi через каналы; отмена через context; при WS+SSE используйте `*SSE`-метод, чтобы форсировать SSE.

**Файлы**: один `io.Reader` / `io.ReadCloser` — raw body; несколько или multipart — schema/multipart. **Всегда закрывайте** возвращённый `io.ReadCloser`.

### Метрики

`WithMetrics()` создаёт отдельный registry (`GetMetricsRegistry()`), не глобальный. `LatencyBuckets` — только до `WithMetrics()`, строго возрастающие `> 0`.

```go
cli := client.New(url,
    client.LatencyBuckets(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 5, 10),
    client.WithMetrics(),
)
reg := cli.GetMetricsRegistry()
http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
```

Несколько клиентов можно объединить через `prometheus.Gatherers`.

### Ошибки

| Транспорт | Default | Кастом |
|-----------|---------|--------|
| JSON-RPC | разбор `error` из ответа | `DecodeError` (`json.RawMessage`) |
| REST | `*ResponseError` при неуспешном статусе | `DecodeHTTPError(status, contentType, body)` |

Держите доменный decoder в handwritten коде потребителя — регенерация клиента его не должна затирать.

### Примеры HTTP / upload

```go
// GET /users/123  (http-path=/users/:id, http-args=id|…)
user, err := cli.UserService().GetUser(ctx, "123")

f, _ := os.Open("doc.pdf")
defer f.Close()
id, err := cli.FileService().Upload(ctx, "doc.pdf", f)

body, contentType, err := cli.FileService().Download(ctx, id)
if err != nil { return err }
defer body.Close()
```

### Streams подробнее

| Направление | Поведение клиента |
|-------------|-------------------|
| server stream | читать из канала до close / cancel ctx |
| client stream | писать во входной канал, получить финальный result |
| bidi | координировать оба направления и отмену |
| WS + SSE | метод без суффикса / с `SSE` — выбор транспорта |

Headers на WS задаются при dial (connection-scoped). SSE metadata — на HTTP-запросе метода.

### Когда появляется `stream.go` / `token/`

| Условие в модели | Артефакт |
|------------------|----------|
| хотя бы один WS/SSE stream-метод | `stream.go` |
| `@tg security` с `http:bearer` или `oauth2:` | каталог `token/` + опции Auth |
| нет security | `Auth` / `token` отсутствуют — не ищите их в options |
| form/multipart HTTP | `schema/` |

### Заголовки из контекста

```go
cli := client.New(url, client.Headers("X-Request-ID", "X-User-ID"))
ctx = context.WithValue(ctx, "X-Request-ID", "req-123")
ctx = context.WithValue(ctx, "X-User-ID", "user-456")
```

Ключи `context.Value` должны совпадать со строками в `Headers(...)`.

## На что обратить внимание

1. **`go.mod not found`** — `--out` вне модуля.
2. Только **`--out`**, не `-o`.
3. **`implicit`** → «пропал» аргумент в сигнатуре — смотрите mapping в astg-json.
4. **Стык endpoint и route**: mount (`/cli`) сохраняется; совпадающий хвост с заявленным route (`/api/v1`) сворачивается — клиент сам не удваивает prefix.
5. Не кладите handwritten код в пакет клиента; после регенерации обновляйте вызывающий код.
6. Kafka в клиенте не появится.
7. `--contracts` без `!Name` (в отличие от swagger).
8. Upload с Auth: не rewindable body + 401 → `ErrBodyNotRewindable`.
9. Смена `enableInlineSingle` / enums / mapping ломает компиляцию потребителей — ожидаемо.

## Ограничения

1. Сигнатуры контрактов: `context.Context` первым, `error` последним, именованные args/results.
2. Только экспортируемые интерфейсы.
3. REST требует `http-method` + `http-path`.
4. Вызывающий закрывает `io.ReadCloser`.
5. Без security-аннотаций нет пакета `token` и `Auth`.

## Связь с другими плагинами

| Плагин | Связь |
|--------|-------|
| `astg` / `astg-json` | модель и диагностика mapping |
| `server` | совместимый wire |
| `client-ts` | тот же контракт, другой runtime |
| `swagger` | OpenAPI для людей/инструментов |
| `tgp-contracts` | правка аннотаций до генерации |
