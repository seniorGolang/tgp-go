# Server Plugin - Генератор серверного кода

## Описание

Плагин `server` генерирует полный серверный код для контрактов с аннотациями `@tg`. Сервер построен на основе фреймворка [Fiber](https://github.com/gofiber/fiber) и поддерживает оба протокола: JSON-RPC 2.0 и HTTP REST API.

Для получения общей информации о парсере и поддерживаемых аннотациях используйте команду `tg plugin doc astg`.

## Основные возможности

- **HTTP обработчики** - автоматическая генерация HTTP endpoints на основе аннотаций
- **JSON-RPC 2.0 обработчики** - полная поддержка JSON-RPC протокола с batch запросами
- **REST API** - поддержка всех HTTP методов (GET, POST, PUT, PATCH, DELETE)
- **Middleware** - встроенная поддержка логирования, метрик и трассировки
- **Типобезопасность** - автоматическая генерация типизированных обработчиков
- **Обработка ошибок** - единообразная обработка и форматирование ошибок
- **Метрики** - автоматический сбор метрик запросов (Prometheus)
- **Трассировка** - поддержка OpenTelemetry трассировки
- **Логирование** - структурированное логирование запросов и ответов

## Использование

### Базовый пример

```go
package main

import (
    "context"
    "log"
    
    "your-module/contracts"
    "your-module/transport"
)

// Реализация контракта
type userService struct{}

func (s *userService) GetUser(ctx context.Context, id string) (user contracts.User, err error) {
    // Ваша бизнес-логика
    return contracts.User{ID: id, Name: "John"}, nil
}

func main() {
    // Создаем сервис
    svc := &userService{}
    
    // Создаем транспортный сервер
    srv := transport.NewServer(
        transport.UserService(contracts.NewUserService(svc)),
    )
    
    // Настраиваем middleware
    srv.WithLog()
    srv.WithMetrics(transport.NewMetrics())
    srv.WithTrace()
    
    // Запускаем сервер
    if err := srv.Listen(":8080"); err != nil {
        log.Fatal(err)
    }
}
```

## Структура генерируемого кода

### Транспортный сервер (`server.go`)

Содержит:

- Структуру `Server` с настройками
- Функцию `NewServer()` для создания сервера
- Методы для настройки middleware
- Методы для запуска и остановки сервера
- Health check endpoints

### HTTP обработчики (`{service}-http.go`)

Для каждого контракта с аннотацией `@tg http-server` генерируется файл с:

- Структурой `http{ServiceName}` (например, `httpUserService`)
- Функцией `new{ServiceName}()` для создания обработчика
- Методом `SetRoutes()` для регистрации маршрутов
- Обработчиками для каждого HTTP метода

### JSON-RPC обработчики (`{service}-jsonrpc.go`)

Для каждого контракта с аннотацией `@tg jsonRPC-server` генерируется файл с:

- Обработчиками для каждого JSON-RPC метода
- Поддержкой batch запросов
- Обработкой ошибок JSON-RPC

### Серверная обертка (`{service}-server.go`)

Для каждого контракта генерируется файл с:

- Структурой `server{ServiceName}` - обертка над реализацией контракта
- Методами для применения middleware
- Типами для middleware

### Middleware (`{service}-middleware.go`, `{service}-logger.go`, `{service}-metrics.go`, `{service}-trace.go`)

Генерируются файлы с middleware:

- **Logger** - логирование запросов и ответов (если указана аннотация `@tg log`)
- **Metrics** - сбор метрик Prometheus (если указана аннотация `@tg metrics`)
- **Trace** - трассировка OpenTelemetry (если указана аннотация `@tg trace`)

### Обмен данными (`{service}-exchange.go`)

Содержит функции для сериализации/десериализации данных и обработки ошибок.

## Особенности генерации

### HTTP обработчики

1. **Маршрутизация**: Маршруты регистрируются автоматически на основе аннотаций `http-method` и `http-path`
2. **Параметры пути**: Параметры пути (например, `:id`) автоматически извлекаются и маппятся на аргументы метода
3. **Query параметры**: Query параметры автоматически парсятся и передаются в метод
4. **Тело запроса**: Для POST/PUT/PATCH методов тело запроса автоматически десериализуется в JSON; при наличии аргумента типа `io.Reader` тело передаётся потоком (upload)
5. **Заголовки и cookies**: Маппятся через аннотации `http-headers` и `http-cookies`
6. **Типы контента**: Поддерживаются кастомные типы через `requestContentType` и `responseContentType`
7. **Upload/Download**: Аргументы `io.Reader` и возвращаемые `io.ReadCloser` допускаются только в контрактах с аннотацией `http-server`. Один такой аргумент/результат — тело запроса/ответа передаётся одним потоком. Несколько таких аргументов/результатов или аннотация `http-multipart` — контент обрабатывается как `multipart/form-data` (имя и Content-Type части задаются аннотациями `http-part-name` и `http-part-content`)

### JSON-RPC обработчики

1. **Именование методов**: Имя метода в JSON-RPC формируется как `{contractName}.{methodName}` в camelCase
2. **Параметры**: Все параметры (кроме `context.Context`) автоматически извлекаются из поля `params` JSON-RPC запроса
3. **Результаты**: Все возвращаемые значения (кроме `error`) автоматически упаковываются в поле `result` JSON-RPC ответа
4. **Batch запросы**: Поддерживается обработка нескольких запросов в одном batch
5. **Ошибки**: Ошибки автоматически форматируются в соответствии со спецификацией JSON-RPC 2.0

### Middleware

Middleware применяются в следующем порядке:

1. **Trace** (если включен) - создает span для запроса
2. **Metrics** (если включен) - собирает метрики
3. **Logger** (если включен) - логирует запрос и ответ
4. **Бизнес-логика** - выполнение метода контракта

### Метрики

Если включена аннотация `@tg metrics`, генерируются следующие метрики Prometheus:

| Метрика | Тип | Лейблы | Описание |
|---------|-----|--------|----------|
| `service_entry_requests_total` | Counter | `protocol` (json-rpc, rest), `result`, `client_id` | Входящие HTTP-запросы на все эндпоинты (JSON-RPC и REST) по исходу на точке входа. Для JSON-RPC возможные значения `result`: `ok`, `method_not_allowed`, `parse_error`, `empty_body`, `batch_size_exceeded`, `invalid_request`. Для REST: `ok`, `method_not_allowed` и др. |
| `service_panics_total` | Counter | — | Количество перехваченных паник по всем эндпоинтам |
| `service_error_responses_total` | Counter | `protocol` (json-rpc, rest), `code`, `client_id` | Ответы с ошибкой по всем эндпоинтам. Каждый ответ клиенту с ошибкой (HTTP 4xx/5xx или тело с полем `error`) инкрементирует эту метрику. `code` — числовой код ошибки в ответе |
| `service_requests_in_flight` | Gauge | — | Текущее число обрабатываемых HTTP-запросов (все эндпоинты: JSON-RPC, REST) |
| `service_request_duration_seconds` | Histogram | `client_id` | Полное время HTTP-запроса по всем эндпоинтам (JSON-RPC, REST) от входа в обработку до отправки ответа. Бакеты: `[0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]` секунд |
| `service_batch_size` | Histogram | — | Распределение размера батча (число JSON-RPC запросов в одном HTTP-запросе). Только для эндпоинтов JSON-RPC |
| `service_requests_count` | Counter | `service`, `method`, `success`, `errCode`, `client_id` | Обработанные вызовы методов по сервису, методу, успеху, коду ошибки и клиенту. Единица учёта — один вызов метода: в batch-запросе каждый JSON-RPC вызов в батче даёт отдельный инкремент |
| `service_requests_latency_seconds` | Histogram | `service`, `method`, `success`, `errCode`, `client_id` | Время выполнения одного вызова метода. Бакеты: `[0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]` секунд. В batch-запросе каждому JSON-RPC вызову в батче соответствует отдельное наблюдение |
| `service_versions_count` | Gauge | `part`, `version`, `hostname` | Версия компонента и хост для идентификации инстанса |

**Лейбл `client_id`**: Значение извлекается из заголовка `X-Client-Id`. Если заголовок отсутствует или пуст после trim, используется значение `unknown`. Лейбл применяется к метрикам: `service_entry_requests_total`, `service_error_responses_total`, `service_requests_count`, `service_requests_latency_seconds`, `service_request_duration_seconds`.

### Трассировка

Если включена аннотация `@tg trace`, генерируется поддержка OpenTelemetry:

- Автоматическое создание spans для каждого запроса
- Интеграция с Fiber middleware
- Поддержка контекста трассировки

### Логирование

Если включена аннотация `@tg log`, генерируется структурированное логирование:

- Логирование всех запросов с параметрами
- Логирование ответов с результатами
- Исключение чувствительных данных через аннотацию `log-skip`
- Использование `log/slog` для логирования

## Опции командной строки

- `out`, `-o` (string, обязательная) - путь к выходной директории
- `contracts-dir` (string, опциональная) - путь к директории с контрактами (по умолчанию: "contracts")
- `contracts` (string, опциональная) - список имён контрактов через запятую для фильтрации (например: "UserService,OrderService")

## Примеры использования

### HTTP сервер

```go
// Контракт
// @tg http-server
// @tg http-prefix=api/v1
// type UserService interface {
//     // @tg http-method=GET
//     // @tg http-path=/users/:id
//     // @tg http-args=id|userId
//     GetUser(ctx context.Context, userId string) (user User, err error)
// }

// Использование
srv := transport.NewServer(
    transport.UserService(contracts.NewUserService(&userService{})),
)
srv.WithLog()
srv.Listen(":8080")
// Endpoint будет доступен по адресу: GET /api/v1/users/:id
```

### JSON-RPC сервер

```go
// Контракт
// @tg jsonRPC-server
// type UserService interface {
//     GetUser(ctx context.Context, id string) (user User, err error)
// }

// Использование
srv := transport.NewServer(
    transport.UserService(contracts.NewUserService(&userService{})),
)
srv.WithLog()
srv.Listen(":8080")
// JSON-RPC endpoint будет доступен по адресу: POST /userService
// Метод: userService.getUser
```

### Смешанный сервер (HTTP + JSON-RPC)

```go
// Контракт
// @tg jsonRPC-server
// @tg http-server
// type UserService interface {
//     // JSON-RPC метод (по умолчанию)
//     GetUser(ctx context.Context, id string) (user User, err error)
//     
//     // HTTP метод (явно указан)
//     // @tg http-method=POST
//     // @tg http-path=/users
//     CreateUser(ctx context.Context, req CreateUserRequest) (id string, err error)
// }

// Использование
srv := transport.NewServer(
    transport.UserService(contracts.NewUserService(&userService{})),
)
srv.WithLog()
srv.WithMetrics(transport.NewMetrics())
srv.Listen(":8080")
// Доступны оба протокола:
// - JSON-RPC: POST /userService (метод: userService.getUser)
// - HTTP: POST /users (метод: CreateUser)
```

### Кастомный обработчик ошибок

```go
srv := transport.NewServer(
    transport.UserService(contracts.NewUserService(&userService{})),
    transport.WithErrorHandler(func(ctx *fiber.Ctx, err error) error {
        // Кастомная обработка ошибок
        code := fiber.StatusInternalServerError
        if e, ok := err.(*fiber.Error); ok {
            code = e.Code
        }
        return ctx.Status(code).JSON(fiber.Map{
            "error": err.Error(),
        })
    }),
)
```

### Upload и Download (потоковое тело)

Для методов с одним аргументом `io.Reader` тело запроса передаётся потоком; для одного возвращаемого `io.ReadCloser` тело ответа отдаётся потоком. Content-Type задаётся аннотациями `requestContentType` и `responseContentType` или по умолчанию `application/octet-stream`.

При нескольких аргументах `io.Reader` или нескольких возвращаемых `io.ReadCloser` контент обрабатывается как **multipart/form-data**. То же самое при одном таком аргументе/результате, если на методе указана аннотация **`http-multipart`**.

- **`http-multipart`** — включить режим multipart для запроса и ответа (один `io.Reader`/`io.ReadCloser` при этой аннотации тоже обрабатывается как одна часть multipart).
- **`http-part-name`** — имя части в multipart (по умолчанию — имя аргумента/результата). Можно задать на параметре/результате или на методе в формате `имя|часть,имя2|часть2`.
- **`http-part-content`** — Content-Type части (по умолчанию `application/octet-stream`). Если задан, в запросе проверяется совпадение с Content-Type части; при несовпадении возвращается 400 с указанием части и причины, в аргумент передаётся пустой reader.

Часть с нужным именем не найдена в запросе — в соответствующий аргумент передаётся пустой reader.

```go
// Контракт
// @tg http-server
// @tg http-prefix=api/v1
// type FileService interface {
//     // @tg http-method=POST
//     // @tg http-path=/files
//     Upload(ctx context.Context, filename string, body io.Reader) (id string, err error)
//
//     // @tg http-method=GET
//     // @tg http-path=/files/:id
//     // @tg http-args=id|fileId
//     Download(ctx context.Context, fileId string) (body io.ReadCloser, contentType string, err error)
// }
```

### Health check

Сервер автоматически генерирует health check endpoint:

```go
srv := transport.NewServer(...)
srv.Listen(":8080")
// Health check доступен по адресу: GET /health
```

### Graceful shutdown

```go
srv := transport.NewServer(...)

// Запуск в горутине
go func() {
    if err := srv.Listen(":8080"); err != nil {
        log.Fatal(err)
    }
}()

// Обработка сигналов для graceful shutdown
quit := make(chan os.Signal, 1)
signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
<-quit

// Graceful shutdown
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
if err := srv.Shutdown(ctx); err != nil {
    log.Fatal(err)
}
```

## Зависимости

Плагин зависит от:

- `astg` - для парсинга контрактов и аннотаций

Генерируемый код использует следующие зависимости:

- `github.com/gofiber/fiber/v2` - HTTP фреймворк
- `github.com/prometheus/client_golang` - метрики Prometheus
- `go.opentelemetry.io/otel` - трассировка OpenTelemetry (опционально)
- `log/slog` - структурированное логирование

## Ограничения

1. Все методы должны принимать `context.Context` первым аргументом
2. Все методы должны возвращать `error` последним значением
3. Все параметры и возвращаемые значения (кроме `error`) должны быть именованными
4. Поддерживаются только публичные интерфейсы (начинающиеся с заглавной буквы)
5. Для HTTP методов требуется явное указание `http-method` и `http-path` аннотаций
6. JSON-RPC методы не могут использовать параметры пути (только тело запроса)
7. Типы `io.Reader` и `io.ReadCloser` разрешены только в контрактах с аннотацией `http-server`

## Совместимость

Плагин полностью совместим с клиентами, сгенерированными плагинами `client-go` и `client-ts`. Все аннотации из плагина `astg` поддерживаются и учитываются при генерации сервера.
