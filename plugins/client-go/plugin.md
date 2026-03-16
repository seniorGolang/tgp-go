# Плагин client-go

## Назначение

Плагин генерирует **Go-клиент** для вызова вашего API. Клиент умеет ходить по **JSON-RPC 2.0** и по **HTTP REST** — в зависимости от того, как помечены контракты в проекте. Вы описываете контракты (интерфейсы) в коде, плагин по ним строит типобезопасные методы, обработку ошибок, при необходимости — метрики и логирование.

Подробнее про контракты и аннотации — в документации плагина `astg`: `tg plugin doc astg`.

## Возможности клиента

- **JSON-RPC 2.0** — вызовы методов и batch-запросы.
- **HTTP REST** — GET, POST, PUT, PATCH, DELETE с настраиваемыми путями, заголовками и cookie.
- **Типобезопасность** — методы генерируются по сигнатурам интерфейсов.
- **Ошибки** — единая обработка и при желании свой декодер ошибок.
- **Метрики** — опциональный сбор метрик запросов (Prometheus).
- **Логирование** — опциональный вывод запросов/ответов или только ошибок.
- **Гибкая настройка** — свой HTTP-клиент, TLS, заголовки из контекста, хуки до/после запроса.

## Как запускать

Команда генерации: `tg client go` (или через общий пайплайн с плагином astg).

**Обязательный параметр:**

- **`out`** — каталог, куда положить сгенерированный код (например, `./client`).

**Необязательные параметры:**

- **`contracts-dir`** — каталог с контрактами относительно корня проекта (по умолчанию `contracts`). Используется при разборе проекта (плагин astg).
- **`contracts`** — список имён контрактов через запятую; генерируется клиент только по ним (например: `UserService,OrderService`). Если не указан — берутся все контракты.
- **`doc-file`** — путь к файлу с документацией по клиенту. По умолчанию при включённой документации: `<out>/readme.md`.
- **`no-doc`** — не генерировать документацию (по умолчанию документация создаётся).

Перед каждой генерацией старые сгенерированные файлы в `out` удаляются; затем создаются новые. Файл документации (например, `readme.md`) при следующем запуске перезаписывается.

## Что появляется в каталоге `out`

В указанном каталоге создаются следующие файлы:

- **client.go** — конструктор `New`, структура `Client`, методы вида `UserService()` для доступа к клиенту контракта;
- **options.go** — тип `Option` и все опции (Name, Headers, LogRequest, WithMetrics и т.д.);
- **error.go** — типы для обработки ошибок и декодер по умолчанию;
- **version.go** — версия генератора;
- **batch.go** — тип `RequestRPC` и метод `Batch()` (только при наличии JSON-RPC-контрактов);
- **metrics.go** — метрики клиента (только при аннотации `@tg metrics` в контракте);
- **jsonrpc/** — подпакет для JSON-RPC;
- **dto/** — типы запросов и ответов и общие типы по контрактам;
- **schema/** — появляется только при использовании HTTP с form/multipart;
- **\<имя_контракта>-exchange.go** и **\<имя_контракта>-client.go** — структуры обмена и методы по каждому контракту (JSON-RPC и HTTP);
- **readme.md** — документация по клиенту (если не отключена через `no-doc`).

## Использование сгенерированного клиента

### Базовый пример

```go
package main

import (
    "context"
    "fmt"

    "your-module/client"
)

func main() {
    cli := client.New("https://api.example.com")
    userService := cli.UserService()

    user, err := userService.GetUser(context.Background(), "user-id-123")
    if err != nil {
        panic(err)
    }
    fmt.Printf("User: %+v\n", user)
}
```

Один клиент (`cli`) даёт доступ ко всем контрактам через методы вида `UserService()`, `OrderService()` и т.д. У каждого такого «сервиса» есть и JSON-RPC-, и HTTP-методы (если они заданы в контракте).

### Настройка клиента

```go
cli := client.New("https://api.example.com",
    client.Name("my-service"),             // Имя клиента (по умолчанию: hostname_astg_go_<версия>; используется в X-Client-Id и в метриках)
    client.LogRequest(),                   // Логировать все запросы
    client.LogOnError(),                   // Логировать только ошибки
    client.Headers("X-Request-ID"),         // Подставлять заголовки из контекста
    client.ConfigTLS(&tls.Config{...}),    // Настройка TLS
    client.ClientHTTP(myHttpClient),       // Свой *http.Client
    client.Transport(myRoundTripper),      // Свой http.RoundTripper
    client.BeforeRequest(func(ctx context.Context, req *http.Request) context.Context {
        return ctx
    }),
    client.AfterRequest(func(ctx context.Context, resp *http.Response) error {
        return nil
    }),
    client.DecodeError(customErrorDecoder), // Свой декодер ошибок
    client.WithMetrics(),                   // Включить метрики (если в контракте есть @tg metrics)
)
```

### HTTP-методы

Для методов, помеченных в контракте как HTTP (`@tg http-method=GET` и т.д.), клиент:

- шлёт указанный HTTP-метод и путь (`http-path`, при необходимости с параметрами типа `:id`);
- подставляет параметры пути через `http-args`, заголовки через `http-headers`, cookie через `http-cookies`;
- поддерживает свой Content-Type запроса/ответа (`requestContentType`, `responseContentType`);
- при одном аргументе `io.Reader` отправляет тело запроса потоком; при одном результате `io.ReadCloser` возвращает тело ответа потоком (его нужно закрыть). При нескольких таких аргументах/результатах или при `http-multipart` используется multipart/form-data (имена и типы частей задаются аннотациями).

#### Режимы маппинга `http-headers` / `http-cookies` / `http-args`

Аннотации маппинга поддерживают формат `arg|key` или `arg|key|mode`, где `mode` — один из `explicit`, `implicit`, `body`. Если режим не указан, по умолчанию используется `body`.

- **`explicit`**: аргумент присутствует в сигнатуре метода клиента и передаётся в указанный HTTP-элемент (заголовок, cookie или query-параметр). Такой параметр виден и в сгенерированной OpenAPI‑документации.
- **`implicit`**: аргумент не попадает в публичную сигнатуру методов клиента (значение обычно берётся из контекста или общих опций), но всё так же маппится в заголовок/cookie/query и описывается в OpenAPI.
- **`body`**: аргумент остаётся полем тела запроса/ответа; маппинг может использоваться сервером как дополнительный источник/override, но сам клиент не превращает его в отдельный HTTP-параметр.

### JSON-RPC

- Имя метода в JSON-RPC: `{имяКонтракта}.{имяМетода}` в camelCase (например, `userService.getUser`).
- Параметры (кроме `context.Context`) попадают в `params`, все возвращаемые значения (кроме `error`) — в `result`.
- Для нескольких запросов одним вызовом используется метод `Batch()` у основного клиента.

### Inline-ответ для одного значения

Если у метода одно возвращаемое значение (кроме `error`) и в контракте указано `enableInlineSingle`, результат приходит без обёртки в объект — одним значением (например, строка статуса).

### Примеры по сценариям

**JSON-RPC:**

```go
cli := client.New("https://api.example.com")
userService := cli.UserService()
user, err := userService.GetUser(ctx, "123")
```

**HTTP GET с параметром в пути:**

```go
// В контракте: @tg http-method=GET, @tg http-path=/users/:id, @tg http-args=id|userId
user, err := userService.GetUser(ctx, "123")  // GET /users/123
```

**Upload (тело запроса потоком):**

```go
f, _ := os.Open("local.pdf")
defer f.Close()
id, err := fileService.Upload(ctx, "doc.pdf", f)
```

**Download (тело ответа потоком):**

```go
body, contentType, err := fileService.Download(ctx, "file-id")
if err != nil { return err }
defer body.Close()
// чтение из body...
```

**HTTP POST с телом:**

```go
id, err := userService.CreateUser(ctx, CreateUserRequest{
    Name:  "John",
    Email: "john@example.com",
})
```

**Batch JSON-RPC:**

```go
requests := []client.RequestRPC{
    userService.ReqGetUser(nil, "1"),
    userService.ReqGetUser(func(user User, err error) {
        if err != nil { log.Printf("Error: %v", err); return }
        log.Printf("User: %+v", user)
    }, "2"),
}
cli.Batch(context.Background(), requests...)
```

**Заголовки из контекста:**

Ключи контекста должны совпадать со строками, переданными в `Headers(...)` (например, имена заголовков):

```go
cli := client.New("https://api.example.com", client.Headers("X-Request-ID", "X-User-ID"))
ctx := context.WithValue(context.Background(), "X-Request-ID", "req-123")
ctx = context.WithValue(ctx, "X-User-ID", "user-456")
user, err := userService.GetUser(ctx, "123")
// В запрос подставятся X-Request-ID и X-User-ID из контекста
```

## Обработка ошибок

Клиент обрабатывает ошибки из JSON-RPC и HTTP. Свой формат ошибок можно поддержать опцией `DecodeError`:

```go
customDecoder := func(errData json.RawMessage) error {
    var errMsg struct { Message string `json:"message"`; Code int `json:"code"` }
    if err := json.Unmarshal(errData, &errMsg); err != nil {
        return fmt.Errorf("failed to decode error: %w", err)
    }
    return fmt.Errorf("error %d: %s", errMsg.Code, errMsg.Message)
}
cli := client.New("https://api.example.com", client.DecodeError(customDecoder))
```

## Метрики

Если в контракте указана аннотация `@tg metrics`, в клиенте можно включить сбор метрик опцией `WithMetrics()`. Метрики создаются в отдельном регистре Prometheus (не в глобальном). Собираются:

- **client_versions_count** — версии компонентов (метки: part, version, hostname).
- **client_requests_count** / **client_requests_all_count** — количество запросов (метки: service, method, success, errCode, client_id).
- **client_requests_latency_seconds** — задержка запросов (те же метки).

Экспорт через свой HTTP-эндпоинт:

```go
cli := client.New("https://api.example.com", client.WithMetrics())
reg := cli.GetMetricsRegistry()
if reg != nil {
    http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
}
http.ListenAndServe(":9090", nil)
```

Для нескольких клиентов можно собрать их реестры в один `prometheus.Gatherers` и отдавать один общий `/metrics`.

## Логирование

Опции `LogRequest()` и `LogOnError()` включают логирование через стандартный `log/slog`. В логах выводятся метод запроса и при необходимости команда curl. В production не рекомендуется включать полное логирование запросов без фильтрации чувствительных данных.

## Документация

По умолчанию плагин генерирует в каталог `out` файл документации (по умолчанию `readme.md`) с описанием клиента, списком контрактов и методов, примерами и типами данных. Отключить — опцией `no-doc`, другой файл задать — опцией `doc-file`.

## Зависимости

- Для разбора контрактов в пайплайне используется плагин **astg** (обычно подключается автоматически).
- Сгенерированный код использует стандартную библиотеку Go; при опции `WithMetrics()` — пакет `github.com/prometheus/client_golang/prometheus`. При использовании form/multipart в HTTP добавляется подпакет **schema** в том же модуле.

## Ограничения

1. У каждого метода первым аргументом должен быть `context.Context`, последним возвращаемым значением — `error`.
2. Все остальные параметры и возвращаемые значения должны быть именованными.
3. Учитываются только экспортируемые интерфейсы (с заглавной буквы).
4. Для HTTP-методов в контракте нужно явно указывать `http-method` и `http-path`.
5. При возврате `io.ReadCloser` вызывающий код обязан закрыть поток после чтения.

## Совместимость

Клиент совместим с серверами, сгенерированными плагином `server` по тем же контрактам. Аннотации, описанные в документации плагина `astg`, учитываются при генерации клиента.
