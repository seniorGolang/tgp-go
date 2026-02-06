# Client-Go Plugin - Генератор Go клиента

## Описание

Плагин `client-go` генерирует Go клиент для взаимодействия с серверами, реализующими контракты с аннотациями `@tg`. Клиент поддерживает оба протокола: JSON-RPC 2.0 и HTTP REST API.

Для получения общей информации о парсере и поддерживаемых аннотациях используйте команду `tg plugin doc astg`.

## Основные возможности

- **JSON-RPC 2.0 клиент** - полная поддержка JSON-RPC протокола с batch запросами
- **HTTP REST клиент** - поддержка HTTP методов (GET, POST, PUT, PATCH, DELETE) с настраиваемыми путями
- **Типобезопасность** - автоматическая генерация типизированных методов на основе Go интерфейсов
- **Обработка ошибок** - встроенная обработка и декодирование ошибок
- **Метрики** - опциональная поддержка метрик запросов
- **Логирование** - опциональное логирование запросов и ответов
- **Кастомизация** - гибкая настройка через опции клиента

## Использование

### Базовый пример

```go
package main

import (
    "context"
    "fmt"

    "your-module/client"
)

func main() {
    // Создаем клиент
    cli := client.New("https://api.example.com")

    // Получаем клиент сервиса по имени контракта (один клиент содержит и HTTP, и JSON-RPC методы контракта)
    userService := cli.UserService()

    // Вызываем метод
    user, err := userService.GetUser(context.Background(), "user-id-123")
    if err != nil {
        panic(err)
    }

    fmt.Printf("User: %+v\n", user)
}
```

### Настройка клиента

```go
cli := client.New("https://api.example.com",
    client.LogRequest(),                    // Логировать все запросы
    client.LogOnError(),                   // Логировать только ошибки
    client.Headers("X-Request-ID"),        // Динамические заголовки из контекста
    client.BeforeRequest(func(ctx context.Context, req *http.Request) context.Context {
        // Кастомная логика перед запросом
        return ctx
    }),
    client.AfterRequest(func(ctx context.Context, resp *http.Response) error {
        // Кастомная логика после запроса
        return nil
    }),
    client.DecodeError(customErrorDecoder), // Кастомный декодер ошибок
    client.WithMetrics(),                  // Включить сбор метрик
)
```

### HTTP методы

Для методов, помеченных как HTTP (с аннотацией `@tg http-method=GET` и т.д.), генерируются методы, которые:

- Используют указанный HTTP метод (GET, POST, PUT, PATCH, DELETE)
- Поддерживают параметры пути через `http-path` и `http-args`
- Поддерживают заголовки через `http-headers`
- Поддерживают cookies через `http-cookies`
- Поддерживают кастомные типы контента через `requestContentType` и `responseContentType`
- Поддерживают upload (аргумент `io.Reader` — тело запроса потоком) и download (возврат `io.ReadCloser` — тело ответа потоком); при нескольких таких аргументах/результатах или аннотации `http-multipart` используется `multipart/form-data` (имя и Content-Type части — аннотации `http-part-name`, `http-part-content`)

## Особенности генерации

### JSON-RPC методы

1. **Именование методов**: Имя метода в JSON-RPC формируется как `{contractName}.{methodName}` в camelCase
2. **Параметры**: Все параметры (кроме `context.Context`) передаются в поле `params` JSON-RPC запроса
3. **Результаты**: Все возвращаемые значения (кроме `error`) упаковываются в поле `result` JSON-RPC ответа
4. **Batch запросы**: Поддерживается выполнение нескольких запросов в одном batch через метод `Batch()` базового клиента

### HTTP методы

1. **Пути**: Используется аннотация `http-path` для определения пути. Если не указана, используется путь по умолчанию: `/{prefix}/{contractName}/{methodName}`
2. **Параметры пути**: Параметры пути (например, `:id`) маппятся на аргументы метода через аннотацию `http-args`
3. **Query параметры**: Параметры, не являющиеся частью пути, передаются как query параметры
4. **Тело запроса**: Для POST/PUT/PATCH методов тело запроса формируется из параметров (JSON); при аргументе типа `io.Reader` тело передаётся потоком; при нескольких `io.Reader` или аннотации `http-multipart` — как `multipart/form-data`
5. **Заголовки и cookies**: Маппятся через аннотации `http-headers` и `http-cookies`
6. **Upload/Download**: При одном аргументе `io.Reader` тело запроса отправляется потоком (Content-Type из `requestContentType` или `application/octet-stream`). При нескольких `io.Reader` или аннотации `http-multipart` запрос формируется как `multipart/form-data` (имя и Content-Type части — `http-part-name`, `http-part-content`). При возвращаемом `io.ReadCloser` ответ не декодируется в JSON — возвращается тело ответа; при нескольких `io.ReadCloser` или `http-multipart` ответ разбирается как multipart. Вызывающий обязан закрыть все возвращённые `ReadCloser`. Второй возврат типа `string` при download заполняется значением заголовка `Content-Type`

### Inline ответы

Для методов с единственным возвращаемым значением (кроме `error`) и аннотацией `enableInlineSingle` результат возвращается напрямую, без обертки в объект:

```go
// Без enableInlineSingle
result := struct {
    Status string `json:"status"`
}{}
// result = {status: "ok"}

// С enableInlineSingle
status := "ok" // Возвращается напрямую
```

## Опции командной строки

- `out` (string, обязательная) - путь к выходной директории
- `contracts-dir` (string, опциональная) - путь к директории с контрактами (по умолчанию: "contracts")
- `contracts` (string, опциональная) - список контрактов через запятую для фильтрации (например: "UserService,OrderService")
- `doc-file` (string, опциональная) - путь к файлу документации (по умолчанию: `<out>/readme.md`)
- `no-doc` (bool, опциональная) - отключить генерацию документации (по умолчанию: false)

## Примеры использования

### JSON-RPC вызов

```go
// Контракт
// @tg jsonRPC-server
type UserService interface {
    GetUser(ctx context.Context, id string) (user User, err error)
}

// Использование
cli := client.New("https://api.example.com")
userService := cli.UserService()
user, err := userService.GetUser(ctx, "123")
```

### HTTP GET запрос

```go
// Контракт
// @tg http-server
type UserService interface {
    // @tg http-method=GET
    // @tg http-path=/users/:id
    // @tg http-args=id|userId
    GetUser(ctx context.Context, userId string) (user User, err error)
}

// Использование
cli := client.New("https://api.example.com")
userService := cli.UserService()
user, err := userService.GetUser(ctx, "123")
// Выполнит GET /users/123
```

### Upload (тело запроса потоком)

```go
// Контракт с аргументом io.Reader — тело запроса передаётся потоком
// @tg http-server
type FileService interface {
    // @tg http-method=POST
    // @tg http-path=/files
    Upload(ctx context.Context, filename string, body io.Reader) (id string, err error)
}

// Использование
cli := client.New("https://api.example.com")
fileService := cli.FileService()
f, _ := os.Open("local.pdf")
defer f.Close()
id, err := fileService.Upload(ctx, "doc.pdf", f)
```

### Download (тело ответа потоком)

```go
// Контракт с возвращаемым io.ReadCloser — тело ответа возвращается потоком; вызывающий обязан закрыть ReadCloser
// @tg http-server
type FileService interface {
    // @tg http-method=GET
    // @tg http-path=/files/:id
    Download(ctx context.Context, id string) (body io.ReadCloser, contentType string, err error)
}

// Использование
body, contentType, err := fileService.Download(ctx, "file-id")
if err != nil {
    return err
}
defer body.Close()
// чтение из body...
```

### HTTP POST запрос с телом

```go
// Контракт
// @tg http-server
type UserService interface {
    // @tg http-method=POST
    // @tg http-path=/users
    CreateUser(ctx context.Context, req CreateUserRequest) (id string, err error)
}

// Использование
cli := client.New("https://api.example.com")
userService := cli.UserService()
id, err := userService.CreateUser(ctx, CreateUserRequest{
    Name:  "John",
    Email: "john@example.com",
})
// Выполнит POST /users с телом запроса
```

### Batch JSON-RPC запросы

```go
cli := client.New("https://api.example.com")
userService := cli.UserService()

// Создаем batch запросы
// Callback функция должна соответствовать сигнатуре метода (result, err)
requests := []client.RequestRPC{
    userService.ReqGetUser(nil, "1"), // Без callback
    userService.ReqGetUser(func(user User, err error) {
        // Callback будет вызван автоматически при получении ответа
        if err != nil {
            log.Printf("Error: %v", err)
            return
        }
        log.Printf("User: %+v", user)
    }, "2"),
}

// Выполняем batch запрос (не возвращает ошибку)
cli.Batch(context.Background(), requests...)

// Callback функции будут вызваны автоматически при получении ответов
```

### Заголовки из контекста

```go
// Настройка клиента
cli := client.New("https://api.example.com",
    client.Headers("X-Request-ID", "X-User-ID"),
)

// Использование
ctx := context.WithValue(context.Background(), "X-Request-ID", "req-123")
ctx = context.WithValue(ctx, "X-User-ID", "user-456")

userService := cli.UserService()
user, err := userService.GetUser(ctx, "123")
// Заголовки X-Request-ID и X-User-ID будут автоматически добавлены из контекста
```

## Обработка ошибок

Клиент автоматически обрабатывает ошибки из JSON-RPC и HTTP ответов. Для кастомизации обработки ошибок можно использовать опцию `DecodeError`:

```go
import "encoding/json"

type ErrorDecoder func(errData json.RawMessage) error

customDecoder := func(errData json.RawMessage) error {
    // Кастомная логика обработки ошибок из JSON-RPC или HTTP
    var errMsg struct {
        Message string `json:"message"`
        Code    int    `json:"code"`
    }
    if err := json.Unmarshal(errData, &errMsg); err != nil {
        return fmt.Errorf("failed to decode error: %w", err)
    }
    return fmt.Errorf("error %d: %s", errMsg.Code, errMsg.Message)
}

cli := client.New("https://api.example.com",
    client.DecodeError(customDecoder),
)
```

## Метрики

Если в контракте указана аннотация `@tg metrics`, клиент может собирать метрики запросов. Для этого используйте опцию `WithMetrics()`:

```go
cli := client.New("https://api.example.com",
    client.WithMetrics(), // Автоматически создаст метрики
)
```

Метрики создаются автоматически при использовании опции `WithMetrics()` и собираются в формате Prometheus.

### Набор собираемых метрик

Клиент собирает следующие метрики:

#### `client_versions_count` (Gauge)

Версии компонентов клиента.

- **Метки:**
    - `part` - название компонента (например, "tg")
    - `version` - версия компонента
    - `hostname` - имя хоста, на котором работает клиент

#### `client_requests_count` (Counter)

Количество отправленных запросов (успешных и неуспешных).

- **Метки:**
    - `service` - имя сервиса в формате `client_{serviceName}` (например, "client_userService")
    - `method` - имя метода в camelCase (например, "getUser")
    - `success` - результат запроса: "true" для успешных, "false" для ошибок
    - `errCode` - код ошибки (число в виде строки, "0" для успешных запросов)

#### `client_requests_all_count` (Counter)

Общее количество всех отправленных запросов (дубликат `client_requests_count` для совместимости).

- **Метки:** те же, что и у `client_requests_count`

#### `client_requests_latency_seconds` (Histogram)

Задержка выполнения запросов в секундах.

- **Метки:**
    - `service` - имя сервиса
    - `method` - имя метода
    - `success` - результат запроса
    - `errCode` - код ошибки

### Пример использования метрик

```go
import (
    "net/http"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

cli := client.New("https://api.example.com",
    client.WithMetrics(),
)

// Экспорт метрик через HTTP endpoint
http.Handle("/metrics", promhttp.Handler())
http.ListenAndServe(":9090", nil)
```

Метрики доступны по адресу `http://localhost:9090/metrics` в формате Prometheus.

## Логирование

Клиент поддерживает логирование запросов и ответов:

```go
cli := client.New("https://api.example.com",
    client.LogRequest(),  // Логировать все запросы
    client.LogOnError(), // Логировать только ошибки
)
```

Логирование использует стандартный `log/slog` пакет Go. При включении `LogRequest()` все запросы логируются в формате curl команды для удобства отладки.

## Документация

По умолчанию плагин генерирует документацию с полной информацией по всем методам и типам. Документация включает:

- Описание клиента
- Список всех контрактов и методов
- Примеры использования для каждого метода
- Описание всех типов данных
- Примеры запросов и ответов

Документацию можно отключить опцией `--no-doc` или указать другой файл через `--doc-file`.

## Зависимости

Плагин зависит от:

- `astg` - для парсинга контрактов и аннотаций

Генерируемый код использует стандартные библиотеки Go:

- `context` - для контекстов
- `net/http` - для HTTP клиента
- `encoding/json` - для JSON сериализации
- `log/slog` - для логирования (опционально)

## Ограничения

1. Все методы должны принимать `context.Context` первым аргументом
2. Все методы должны возвращать `error` последним значением
3. Все параметры и возвращаемые значения (кроме `error`) должны быть именованными
4. Поддерживаются только публичные интерфейсы (начинающиеся с заглавной буквы)
5. Для HTTP методов требуется явное указание `http-method` и `http-path` аннотаций
6. При download (возврат `io.ReadCloser`) вызывающий обязан закрыть возвращённый `ReadCloser`

## Совместимость

Плагин полностью совместим с контрактами, сгенерированными плагином `server`. Все аннотации из плагина `astg` поддерживаются и учитываются при генерации клиента.
