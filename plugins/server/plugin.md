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
        transport.WithUserService(contracts.NewUserService(svc)),
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
4. **Тело запроса**: Для POST/PUT/PATCH методов тело запроса автоматически десериализуется
5. **Заголовки и cookies**: Маппятся через аннотации `http-headers` и `http-cookies`
6. **Типы контента**: Поддерживаются кастомные типы через `requestContentType` и `responseContentType`

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

- `request_count` - счетчик запросов с метками: `method`, `service`, `success`
- `request_count_all` - счетчик всех запросов
- `request_latency` - гистограмма задержек запросов

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

- `output`, `-o` (string, обязательная) - путь к выходной директории
- `contracts`, `-c` (string, опциональная) - путь к директории с контрактами (по умолчанию: "contracts")
- `ifaces` (string, опциональная) - список интерфейсов через запятую для фильтрации (например: "UserService,OrderService")

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
    transport.WithUserService(contracts.NewUserService(&userService{})),
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
    transport.WithUserService(contracts.NewUserService(&userService{})),
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
    transport.WithUserService(contracts.NewUserService(&userService{})),
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
    transport.WithUserService(contracts.NewUserService(&userService{})),
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

## Совместимость

Плагин полностью совместим с клиентами, сгенерированными плагинами `client-go` и `client-ts`. Все аннотации из плагина `astg` поддерживаются и учитываются при генерации сервера.
