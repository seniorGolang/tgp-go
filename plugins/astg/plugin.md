# ASTG Plugin - Парсер AST для генерации кода

## Описание

Плагин `astg` (AST Generator) является основным парсером проекта, который анализирует Go код и извлекает структурированную информацию о контрактах, типах, методах и аннотациях. Этот плагин служит основой для всех остальных плагинов генерации кода, предоставляя им единую модель проекта.

## Основные возможности

- **Парсинг интерфейсов** - автоматическое обнаружение интерфейсов с аннотациями `@tg`
- **Анализ типов** - полное извлечение информации о типах данных (структуры, массивы, мапы, указатели)
- **Обработка аннотаций** - парсинг и структурирование аннотаций на всех уровнях (пакет, интерфейс, метод, поле)
- **Разрешение зависимостей** - автоматическое разрешение типов из импортированных пакетов
- **Генерация модели проекта** - создание структурированной модели для использования в других плагинах

## Опции (при использовании в пайплайне)

Плагин читает из запроса следующие опции:

- `contracts-dir` (string, опциональная) - путь к директории с контрактами (по умолчанию: "contracts")
- `contracts` (string, опциональная) - список имён контрактов через запятую для фильтрации (например: "UserService,OrderService")
- `no-cache` (bool, опциональная) - отключить использование кэша, выполнить парсинг заново

## Формат аннотаций

Аннотации записываются в комментариях Go кода с префиксом `@tg`:

```go
// @tg <имя>[=<значение>] [<имя2>[=<значение2>] ...]
```

### Примеры

```go
// Одна аннотация без значения
// @tg log

// Одна аннотация со значением
// @tg http-prefix=api/v1

// Несколько аннотаций в одной строке
// @tg jsonRPC-server log metrics

// Несколько строк с аннотациями
// @tg http-prefix=api/v1
// @tg jsonRPC-server
// @tg log metrics trace
```

### Значения в обратных кавычках

Для значений, содержащих пробелы или специальные символы, можно использовать обратные кавычки:

```go
// @tg http-path=`/api/v1/users/:id`
```

## Уровни применения аннотаций

Аннотации могут применяться на следующих уровнях:

1. **Пакет** - действует на все интерфейсы и методы в пакете
2. **Интерфейс** - действует на все методы интерфейса
3. **Метод** - действует только на конкретный метод
4. **Тип/Поле** - применяется к полям структур или параметрам методов

**Приоритет**: Метод > Интерфейс > Пакет

Аннотации с более высоким приоритетом переопределяют аннотации с более низким приоритетом.

## Полный список аннотаций

### Аннотации уровня пакета

| Аннотация                  | Описание                                                                | Пример                                              |
|----------------------------|-------------------------------------------------------------------------|-----------------------------------------------------|
| `log`                      | Включает логирование запросов для всех интерфейсов пакета               | `// @tg log`                                        |
| `trace`                    | Включает трассировку запросов для всех интерфейсов пакета               | `// @tg trace`                                      |
| `metrics`                  | Включает сбор метрик для всех интерфейсов пакета                        | `// @tg metrics`                                    |
| `http-prefix=<префикс>`    | Префикс пути URL для всех методов пакета                                | `// @tg http-prefix=api/v1`                         |
| `packageJSON=<пакет>`      | Переопределение JSON-кодека (по умолчанию `encoding/json`)              | `// @tg packageJSON=github.com/seniorGolang/json`   |
| `uuidPackage=<пакет>`      | Переопределение пакета для UUID (по умолчанию `github.com/google/uuid`) | `// @tg uuidPackage=github.com/gofrs/uuid`          |
| `swaggerTags=<тег1,тег2>`  | Теги для группировки в OpenAPI документации                             | `// @tg swaggerTags=users,api`                      |
| `security=bearer`          | Указание типа авторизации для OpenAPI                                   | `// @tg security=bearer`                            |
| `servers=<адрес>;<имя>`    | Список серверов для документации OpenAPI                                | `// @tg servers=https://api.example.com;Production` |
| `version=<версия>`         | Версия сервиса для документации                                         | `// @tg version=1.0.0`                              |
| `title=<заголовок>`        | Заголовок документации OpenAPI                                          | `// @tg title=My API`                               |
| `author=<автор>`           | Автор NPM-пакета (для JS клиентов)                                      | `// @tg author=John Doe`                            |
| `npmRegistry=<адрес>`      | Адрес NPM-репозитория                                                   | `// @tg npmRegistry=https://registry.npmjs.org`     |
| `npmName=<имя>`            | Имя NPM-пакета                                                          | `// @tg npmName=@myorg/my-api-client`               |
| `npmPrivate=<true\|false>` | Приватность NPM-пакета                                                  | `// @tg npmPrivate=true`                            |
| `license=<лицензия>`       | Лицензия NPM-пакета                                                     | `// @tg license=MIT`                                |

### Аннотации уровня интерфейса

| Аннотация                 | Описание                                                  | Пример                                |
|---------------------------|-----------------------------------------------------------|---------------------------------------|
| `jsonRPC-server`          | Включение JSON-RPC 2.0 сервера для интерфейса             | `// @tg jsonRPC-server`               |
| `http-server`             | Включение HTTP-сервера для интерфейса                     | `// @tg http-server`                  |
| `log`                     | Включает логирование запросов для всех методов интерфейса | `// @tg log`                          |
| `trace`                   | Включает трассировку запросов для всех методов интерфейса | `// @tg trace`                        |
| `metrics`                 | Включает сбор метрик для всех методов интерфейса          | `// @tg metrics`                      |
| `http-prefix=<префикс>`   | Префикс пути URL для всех методов интерфейса              | `// @tg http-prefix=api/v1`           |
| `swaggerTags=<тег1,тег2>` | Теги для группировки в OpenAPI                            | `// @tg swaggerTags=users,api`        |
| `tagOmitemptyAll`         | Добавление тега `omitempty` ко всем полям Exchange (запрос/ответ) | `// @tg tagOmitemptyAll`              |
| `desc=<описание>`         | Краткое описание интерфейса для документации              | `// @tg desc=User management service` |

### Аннотации уровня метода

| Аннотация                                | Описание                                                                 | Пример                                        |
|------------------------------------------|--------------------------------------------------------------------------|-----------------------------------------------|
| `http-method=<метод>`                    | HTTP-метод для доступа к методу (GET, POST, PUT, PATCH, DELETE, OPTIONS) | `// @tg http-method=POST`                     |
| `http-path=<путь>`                       | Путь URL для метода. Поддерживает параметры пути (`:id`)                 | `// @tg http-path=/users/:id`                 |
| `http-success=<код>`                     | HTTP-код успешного ответа (по умолчанию 200)                             | `// @tg http-success=201`                     |
| `http-args=<переменная>\|<ключ>`         | Маппинг параметров URL на аргументы метода                               | `// @tg http-args=id\|userId`                 |
| `http-headers=<переменная>\|<заголовок>` | Маппинг заголовков на аргументы/результаты                               | `// @tg http-headers=token\|Authorization`    |
| `http-cookies=<переменная>\|<cookie>`    | Маппинг cookies на аргументы/результаты                                  | `// @tg http-cookies=session\|sessionId`      |
| `http-response=<модуль>:<метод>`         | Указание кастомного обработчика с контекстом go-fiber                    | `// @tg http-response=handlers:CustomHandler` |
| `handler=<модуль>:<метод>`               | Указание полностью кастомного обработчика                                | `// @tg handler=handlers:CustomHandler`       |
| `requestContentType=<mime>`              | MIME-тип запроса (по умолчанию `application/json`)                       | `// @tg requestContentType=application/xml`   |
| `responseContentType=<mime>`             | MIME-тип ответа (по умолчанию `application/json`)                        | `// @tg responseContentType=application/xml`  |
| `enableInlineSingle`                     | Включение inline для методов с единственным возвращаемым значением       | `// @tg enableInlineSingle`                   |
| `http-multipart`                         | Включение режима multipart для запроса и ответа (один `io.Reader`/`io.ReadCloser` при этой аннотации тоже обрабатывается как одна часть) | `// @tg http-multipart`                       |
| `http-part-name=<аргумент>\|<часть>`     | Имя части в multipart (на методе: маппинг имён аргументов/результатов на имена частей) | `// @tg http-part-name=body\|file1`           |
| `http-part-content=<аргумент>\|<mime>`   | Content-Type части в multipart (на методе: маппинг имён на MIME-типы)     | `// @tg http-part-content=body\|image/png`    |
| `log-skip=<переменная>`                  | Исключение переменных из логов                                           | `// @tg log-skip=password`                    |
| `deprecated`                             | Пометка метода как устаревшего в OpenAPI                                 | `// @tg deprecated`                           |
| `summary=<описание>`                     | Детальное описание метода для OpenAPI. Поддерживает форматирование       | `// @tg summary=Creates a new user`           |
| `desc=<описание>`                        | Краткое описание метода для документации                                 | `// @tg desc=Create user endpoint`            |
| `swaggerTags=<тег1,тег2>`                | Теги для группировки в OpenAPI (переопределяет теги интерфейса)          | `// @tg swaggerTags=users`                    |

### Аннотации уровня типа/поля

Аннотации для полей структур и параметров методов применяются в комментариях перед полем или параметром:

| Аннотация                            | Описание                                                                    | Пример                                                |
|--------------------------------------|-----------------------------------------------------------------------------|-------------------------------------------------------|
| `desc=<описание>`                    | Описание поля для документации                                              | `// @tg desc=User identifier`                         |
| `type=<тип>`                         | Указание типа поля в документации OpenAPI                                   | `// @tg type=string`                                  |
| `enums=val1,val2`                    | Список возможных значений поля                                              | `// @tg enums=active,inactive,pending`                |
| `format=<формат>`                    | Формат поля в документации OpenAPI (например, `uuid`, `email`, `date-time`) | `// @tg format=uuid`                                  |
| `required`                           | Указывает, что поле обязательно                                             | `// @tg required`                                     |
| `example=<значение>`                 | Пример значения поля в документации                                         | `// @tg example=550e8400-e29b-41d4-a716-446655440000` |
| `<переменная>.tags=<тег>:<значение>` | Настройка тегов для полей (например, `dumper:hide` для скрытия в логах)     | `// @tg user.tags=dumper:hide`                        |
| `http-part-name=<имя части>`         | Имя части в multipart (для аргумента `io.Reader` или результата `io.ReadCloser`) | `// @tg http-part-name=file1`                        |
| `http-part-content=<mime>`           | Content-Type части в multipart                                            | `// @tg http-part-content=image/png`                 |

## Примеры использования

### Базовый пример интерфейса

```go
// @tg jsonRPC-server log metrics trace
// @tg http-prefix=api/v1
// @tg swaggerTags=users,api
type UserService interface {
// @tg summary=Creates a new user
// @tg desc=Creates a new user in the system
CreateUser(ctx context.Context, user CreateUserRequest) (id string, err error)

// @tg summary=Gets user by ID
GetUser(ctx context.Context, id string) (user User, err error)
}
```

### Пример с HTTP заголовками и cookies

```go
// @tg http-server
type AuthService interface {
// @tg http-method=POST
// @tg http-path=/login
// @tg http-headers=token|Authorization
// @tg http-cookies=session|sessionId
Login(ctx context.Context, username string, password string) (token string, session string, err error)
}
```

### Пример с параметрами URL

```go
// @tg http-server
type UserService interface {
// @tg http-method=GET
// @tg http-path=/users/:id/posts
// @tg http-args=id|userId
GetUserPosts(ctx context.Context, id string) (posts []Post, err error)
}
```

### Пример с кастомным обработчиком

```go
// @tg http-server
type FileService interface {
// @tg http-method=POST
// @tg http-path=/upload
// @tg handler=handlers:FileUploadHandler
UploadFile(ctx context.Context, file []byte) (url string, err error)
}
```

### Пример с аннотациями полей

```go
type CreateUserRequest struct {
// @tg required
// @tg desc=User email address
// @tg format=email
Email string `json:"email"`

// @tg required
// @tg desc=User password
// @tg log-skip
Password string `json:"password"`

// @tg desc=User role
// @tg enums=admin,user,guest
Role string `json:"role"`
}
```

### Пример с разными типами контента

```go
// @tg http-server
type DocumentService interface {
// @tg http-method=POST
// @tg http-path=/documents
// @tg requestContentType=application/xml
// @tg responseContentType=application/xml
CreateDocument(ctx context.Context, xml string) (id string, err error)
}
```

### Пример с inline ответом

```go
// @tg http-server
type StatusService interface {
// @tg http-method=GET
// @tg http-path=/status
// @tg enableInlineSingle
GetStatus(ctx context.Context) (status string, err error)
}
```

## Приоритеты аннотаций

Аннотации наследуются от более высокого уровня к более низкому, но могут быть переопределены:

1. **Пакет** → применяется ко всем интерфейсам пакета
2. **Интерфейс** → переопределяет аннотации пакета для конкретного интерфейса
3. **Метод** → переопределяет аннотации интерфейса для конкретного метода
4. **Поле** → применяется только к конкретному полю

### Пример наследования

```go
// Пакетный уровень
// @tg log metrics
// @tg http-prefix=api/v1

// Интерфейс переопределяет префикс
// @tg http-prefix=api/v2
// @tg trace
type UserService interface {
// Метод переопределяет все для конкретного метода
// @tg http-method=GET
// @tg http-path=/users
// @tg log-skip=password
GetUsers(ctx context.Context) (users []User, err error)
}
```

В этом примере:

- `GetUsers` будет иметь `log`, `metrics`, `trace` (наследуется от пакета и интерфейса)
- `http-prefix` будет `api/v2` (переопределен на уровне интерфейса)
- `http-method` и `http-path` установлены только для этого метода

## Работа с аннотациями в плагинах

Плагины генерации получают структурированную модель проекта через тип `astg.Project`, который содержит все аннотации в виде `DocTags` (map[string]string).

### Проверка наличия аннотации

```go
if contract.Annotations.IsSet("jsonRPC-server") {
// Генерируем JSON-RPC код
}
```

### Получение значения аннотации

```go
// Строковое значение с дефолтом
prefix := contract.Annotations.Value("http-prefix", "api/v1")

// Числовое значение с дефолтом
successCode := method.Annotations.ValueInt("http-success", 200)

// Булево значение с дефолтом
enabled := method.Annotations.ValueBool("enableInlineSingle", false)
```

### Работа с подтегами

```go
// Получить все теги с префиксом "http."
httpTags := method.Annotations.Sub("http")
method := httpTags.Value("method") // Получит значение из "http-method"
```

## Особенности парсинга

### Именованные аргументы и возвращаемые значения

Все параметры методов и возвращаемые значения (кроме `error`) должны быть именованными. Эти имена используются в транспортном слое (например, в JSON-RPC):

```go
// ✅ Правильно
Method(ctx context.Context, userID string, count int) (result string, total int, err error)

// ❌ Неправильно
Method(ctx context.Context, string, int) (string, int, error)
```

### Контекст и ошибка

- Первый аргумент метода должен быть `context.Context`
- Последний возвращаемый тип должен быть `error`

### Формат JSON-RPC запроса

Для метода `UserService.GetUser` запрос в формате JSON-RPC 2.0 будет выглядеть так:

```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "userService.getUser",
  "params": {
    "id": "123"
  }
}
```

Ответ:

```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": {
    "user": {
      "id": "123",
      "name": "John Doe"
    }
  }
}
```

## Отладка

Для упрощения отладки и анализа структуры проекта плагин сохраняет полную структуру распарсенного контракта в файл `.tg/project.json` в корне проекта **только когда в request передан уровень лога `debug`** (хост кладёт в request значение флага `--log-level`, ключ `log-level`). Этот файл содержит полную модель проекта в формате JSON, включая все контракты, методы, типы, аннотации и их связи.

Файл создаётся при запуске плагина с `log-level: debug` в request и может быть использован для:

- Анализа структуры проекта
- Отладки проблем парсинга
- Проверки корректности извлечённых данных
- Интеграции с внешними инструментами анализа

**Примечание**: Файл `.tg/project.json` создаётся только при `log-level: debug` в request (например, при вызове с `--log-level debug` на хосте) и не влияет на производительность в production-режиме.

## Совместимость со старой системой

Плагин `astg` полностью совместим с аннотациями из старой системы `tg`. Все поддерживаемые аннотации из [документации старой системы](https://github.com/seniorGolang/tg/raw/refs/heads/main/README.md) работают в новой системе без изменений.

### Основные отличия

1. **Модульная архитектура** - плагин `astg` является отдельным модулем, который может использоваться независимо
2. **Расширяемость** - новые плагины могут легко добавлять поддержку новых аннотаций
3. **Типобезопасность** - использование структурированных типов вместо строковых констант

## Ограничения

1. Интерфейсы без аннотаций `@tg` игнорируются парсером
2. Методы без аннотаций наследуют настройки от интерфейса и пакета
3. Аннотации применяются только к публичным интерфейсам (начинающимся с заглавной буквы)
4. Парсер не обрабатывает методы, которые являются встроенными интерфейсами (embedded interfaces)

## Заключение

Плагин `astg` предоставляет мощный и гибкий механизм для парсинга Go кода и извлечения структурированной информации о контрактах, типах и аннотациях. Используя аннотации `@tg`, разработчики могут легко настраивать генерацию кода без изменения бизнес-логики.
