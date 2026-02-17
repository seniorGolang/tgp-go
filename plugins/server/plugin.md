# Плагин server — генератор серверного кода по контрактам

## Назначение

Плагин генерирует готовый HTTP/JSON-RPC сервер по вашим контрактам: вы описываете интерфейсы с аннотациями `@tg`, запускаете генерацию — получаете код на [Fiber](https://github.com/gofiber/fiber), в котором остаётся только передать реализацию сервисов и вызвать запуск.

Список аннотаций и их смысл: `tg plugin doc astg`.

## Запуск

```bash
tgp server -o <каталог>
```

| Параметр | Обязательный | Описание |
|----------|--------------|----------|
| **out**, **-o** | да | Каталог, в который записывается сгенерированный код (например, `transport`). |
| **contracts-dir** | нет | Папка с контрактами относительно корня проекта. По умолчанию: `contracts`. |
| **contracts** | нет | Список имён контрактов через запятую (например, `UserService,OrderService`). Генерируется код только для них. |

## Что вы получаете

В указанном каталоге появляется Go-пакет. Вы импортируете его и:

1. Создаёте сервер: `transport.New(log, options...)`.
2. В опциях передаёте реализации контрактов (например, `transport.UserService(impl)`).
3. При необходимости включаете логирование, метрики или трассировку: `srv.WithLog()`, `srv.WithMetrics()`, `srv.WithTrace(...)`.
4. Запускаете: `srv.Fiber().Listen(":8080")`.

Детали API — в разделах ниже.

## Аннотации, важные для сервера

Имеют смысл только аннотации, уже описанные в плагине `astg`. Для сервера важны в частности:

- **`@tg http-server`** — по контракту генерируются HTTP (REST) обработчики.
- **`@tg jsonRPC-server`** — по контракту генерируются JSON-RPC 2.0 обработчики (в том числе batch).
- **`@tg log`** — доступно логирование запросов/ответов через `srv.WithLog()`.
- **`@tg metrics`** — доступны метрики Prometheus через `srv.WithMetrics()`.
- **`@tg trace`** — доступна трассировка OpenTelemetry через `srv.WithTrace(...)`.

Остальные аннотации (`http-method`, `http-path`, `http-prefix`, `http-headers`, `http-cookies`, `log-skip` и т.д.) задают маршруты, заголовки и поведение. Их описание см. в документации плагина `astg`.

## Создание и запуск сервера

### Базовый пример

```go
package main

import (
    "context"
    "log"
    "log/slog"

    "your-module/contracts"
    "your-module/transport"
)

type userService struct{}

func (s *userService) GetUser(ctx context.Context, id string) (user contracts.User, err error) {
    return contracts.User{ID: id, Name: "John"}, nil
}

func main() {
    svc := &userService{}
    srv := transport.New(slog.Default(), transport.UserService(contracts.NewUserService(svc)))

    srv.WithLog()
    srv.WithMetrics() // при первом вызове метрики создаются внутри
    srv.WithTrace(context.Background(), "myapp", "http://localhost:4318" /* при необходимости: attributes */)

    if err := srv.Fiber().Listen(":8080"); err != nil {
        log.Fatal(err)
    }
}
```

- **`New(log *slog.Logger, options ...Option) *Server`** — создаёт сервер. В `options` передаёте реализации контрактов и при необходимости конфигурацию.
- **`srv.Fiber()`** — возвращает `*fiber.App`; запуск через `srv.Fiber().Listen(addr)`.
- **`srv.WithLog()`** — включает логирование запросов и ответов (нужна аннотация `@tg log` у контракта).
- **`srv.WithMetrics()`** — включает метрики Prometheus (нужна аннотация `@tg metrics`). При первом вызове объект метрик создаётся внутри.
- **`srv.WithTrace(ctx, appName, endpoint, attributes...)`** — включает трассировку OpenTelemetry (нужна аннотация `@tg trace`).

## Опции при создании сервера (Option)

В `New(log, options...)` можно передать:

| Опция | Назначение |
|-------|------------|
| **`{ContractName}(impl)`** | Регистрация реализации контракта и маршрутов (например, `UserService(impl)`). Имя совпадает с именем интерфейса. |
| **`Service(svc ServiceRoute)`** | Регистрация произвольного маршрутизатора (реализация задаёт маршруты через `SetRoutes(route)`). |
| **`SetFiberCfg(cfg fiber.Config)`** | Полная конфигурация Fiber. |
| **`SetReadBufferSize(size int)`**, **`SetWriteBufferSize(size int)`** | Размеры буферов чтения/записи. |
| **`MaxBodySize(size int)`** | Лимит размера тела запроса. |
| **`ReadTimeout(timeout time.Duration)`**, **`WriteTimeout(timeout time.Duration)`** | Таймауты чтения/записи. |
| **`MaxBatchSize(size int)`**, **`MaxBatchWorkers(size int)`** | Только при наличии контракта с `@tg jsonRPC-server`: макс. размер batch и число воркеров. По умолчанию 100 и 10. |
| **`WithRequestID(headerName string)`** | Обработка заголовка Request ID: если значение пустое, подставляется UUID. |
| **`WithHeader(headerName string, handler HeaderHandler)`** | Свой обработчик заголовка. |
| **`Use(args ...any)`** | Добавление произвольных Fiber middleware. |

## Остановка и health-check

- **`srv.Shutdown() error`** — корректная остановка сервера (таймаут по умолчанию 30 секунд). Останавливается основной HTTP-сервер и при наличии — сервер метрик.

- **`transport.ServeHealth(log, path, address, response)`** — отдельный HTTP-сервер для health-check (не встроен в основной `Server`). Возвращает `*HealthServer` с методом **`Stop()`** для остановки.  
  Пример: поднять health на другом порту, в ответ отдать JSON.

```go
hs := transport.ServeHealth(slog.Default(), "/health", ":8081", map[string]string{"status": "ok"})
defer hs.Stop()
// GET /health на :8081 вернёт указанный response; при response == nil в ответ уходит "ok"
```

Основной API-сервер по-прежнему запускается через `srv.Fiber().Listen(":8080")`.

## Кастомная обработка ошибок

Для контрактов с **`@tg http-server`** у сервера есть метод по имени контракта, возвращающий HTTP-обработчик этого контракта. Через него можно задать свой обработчик ошибок:

- **`srv.UserService().WithErrorHandler(handler)`** — пример для контракта `UserService`.  
  `handler` имеет тип **`func(err error) error`**: вы можете обернуть или подменить ошибку перед отправкой клиенту.

Вызов делается после создания сервера, до запуска (до `Listen`). Для контрактов только с JSON-RPC (без `http-server`) такого доступа к обработчику нет.

## Маршруты HTTP и JSON-RPC

- **HTTP**: маршруты строятся по аннотациям `http-method`, `http-path`, `http-prefix`. Параметры пути (`:id` и т.п.), query и тело запроса маппятся на аргументы методов. Для POST/PUT/PATCH тело по умолчанию парсится как JSON.
- **JSON-RPC**: имя метода в запросе — `{contractName}.{methodName}` в camelCase (например, `userService.getUser`). Параметры передаются в поле `params`, результат — в `result`. Поддерживаются batch-запросы.

Если у контракта указаны и `http-server`, и `jsonRPC-server`, один и тот же контракт обслуживает и HTTP, и JSON-RPC методы.

## Загрузка и выгрузка файлов (потоковое тело)

- Один аргумент **`io.Reader`** — тело запроса передаётся потоком в этот аргумент.
- Один результат **`io.ReadCloser`** — тело ответа отдаётся потоком из этого результата.
- Несколько таких аргументов/результатов или аннотация **`http-multipart`** — обмен идёт в формате **multipart/form-data**. Имена и Content-Type частей задаются аннотациями **`http-part-name`** и **`http-part-content`**.

Типы `io.Reader` и `io.ReadCloser` в контракте допускаются только при аннотации **`@tg http-server`**.

## Метрики (Prometheus)

При аннотации **`@tg metrics`** и вызове **`srv.WithMetrics()`** в метриках появляются счётчики и гистограммы в пространстве имён `service_*`. Чтобы отдать метрики по HTTP (например, для сбора Prometheus), вызовите **`srv.ServeMetrics(log, path, address, regs...)`** — будет поднят отдельный HTTP-сервер на указанном адресе (как и для health-check). При **`Shutdown()`** этот сервер тоже останавливается.

- входящие запросы, паники, ответы с ошибкой;
- число запросов в обработке, длительность запроса и длительность вызова метода;
- для JSON-RPC — размер batch;
- версия компонента и хост.

Во все метрики, где это предусмотрено, добавляется лейбл **`client_id`**: он берётся из заголовка **`X-Client-Id`**; если заголовка нет или он пустой — используется значение `unknown`.

Точный список имён метрик и лейблов задаётся сгенерированным кодом (файл `metrics.go` в сгенерированном пакете).

## Трассировка и логирование

- **Трассировка** (**`@tg trace`**): вызов **`srv.WithTrace(ctx, appName, endpoint, attributes...)`** инициализирует провайдер OpenTelemetry и подключает трассировку к обработчикам.
- **Логирование** (**`@tg log`**): вызов **`srv.WithLog()`** включает структурированное логирование запросов и ответов через `log/slog`. Чувствительные поля можно исключить аннотацией **`log-skip`** (см. astg).

## Примеры

### Только HTTP

```go
// Контракт с @tg http-server, @tg http-prefix=api/v1 и методами с http-method, http-path

srv := transport.New(slog.Default(), transport.UserService(contracts.NewUserService(&userService{})))
srv.WithLog()
_ = srv.Fiber().Listen(":8080")
// Например: GET /api/v1/users/:id
```

### Только JSON-RPC

```go
// Контракт с @tg jsonRPC-server

srv := transport.New(slog.Default(), transport.UserService(contracts.NewUserService(&userService{})))
srv.WithLog()
_ = srv.Fiber().Listen(":8080")
// POST /userService, метод в запросе: userService.getUser
```

### HTTP и JSON-RPC вместе

```go
// Контракт с @tg jsonRPC-server и @tg http-server, у части методов — http-method и http-path

srv := transport.New(slog.Default(), transport.UserService(contracts.NewUserService(&userService{})))
srv.WithLog()
srv.WithMetrics()
_ = srv.Fiber().Listen(":8080")
```

### Graceful shutdown

```go
srv := transport.New(slog.Default(), transport.UserService(contracts.NewUserService(&userService{})))
srv.WithLog()

go func() {
    if err := srv.Fiber().Listen(":8080"); err != nil {
        log.Fatal(err)
    }
}()

quit := make(chan os.Signal, 1)
signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
<-quit

if err := srv.Shutdown(); err != nil {
    log.Fatal(err)
}
```

## Ограничения

1. Методы контракта должны принимать **`context.Context`** первым аргументом и возвращать **`error`** последним значением.
2. Все параметры и возвращаемые значения (кроме `error`) должны быть именованными.
3. Поддерживаются только публичные интерфейсы (имя с заглавной буквы).
4. Для HTTP-методов нужны аннотации **`http-method`** и **`http-path`**; при использовании **`http-prefix`** в **`http-path`** указывайте путь с ведущим `/`.
5. В JSON-RPC параметры передаются только в теле запроса (параметры пути не используются).
6. Типы **`io.Reader`** и **`io.ReadCloser`** в контракте допустимы только при аннотации **`@tg http-server`**.

## Зависимости и совместимость

- Плагин зависит от **astg** (парсинг контрактов и аннотаций).
- Сгенерированный код использует: **github.com/gofiber/fiber/v2**, **log/slog**, при включённых метриках — **github.com/prometheus/client_golang**, при трассировке — **go.opentelemetry.io/otel**.

Сервер совместим с клиентами, сгенерированными плагинами **client-go** и **client-ts**: аннотации из astg согласованы между сервером и клиентами.
