# Плагин ASTG

## Назначение

Плагин `astg` анализирует ваш Go-проект: находит контракты (интерфейсы с аннотациями `@tg`), методы, типы и аннотации. Результат анализа используется другими плагинами (сервер, клиенты, Swagger и т.д.) для генерации кода. Вы описываете контракты в комментариях — плагин собирает из них единую модель проекта.

## Опции пайплайна

При запуске пайплайна можно передать плагину опции:

| Опция | Тип | Описание |
|-------|-----|----------|
| `contracts-dir` | строка | Папка с контрактами. По умолчанию `contracts`. Читаются только `.go` файлы **в самой этой папке** (вложенные папки не просматриваются). |
| `contracts` | строка | Список имён контрактов через запятую, чтобы обрабатывать только их (например: `UserService,OrderService`). |
| `no-cache` | bool | Не использовать кэш — каждый раз разбирать проект заново. |

## Кэш

Плагин может брать ранее собранную модель из кэша, чтобы не разбирать проект при каждом запуске. Кэш считается актуальным, пока не изменились учтённые файлы: `go.mod`, `go.sum` и релевантные `.go` файлы проекта. Директории `.tg`, `.git` и `vendor` в расчёт не входят; файлы с заголовком генерации tgp тоже не учитываются. Если нужен принудительный полный разбор — передайте опцию `no-cache`.

## Аннотации `@tg`

Аннотации пишутся в комментариях к Go-коду с префиксом `@tg`:

```go
// @tg <имя>[=<значение>] [<имя2>[=<значение2>] ...]
```

### Примеры

```go
// Одна аннотация без значения
// @tg log

// Со значением
// @tg http-prefix=api/v1

// Несколько в одной строке
// @tg jsonRPC-server log metrics

// Несколько строк
// @tg http-prefix=api/v1
// @tg jsonRPC-server
// @tg log metrics trace
```

### Значения с пробелами или спецсимволами

Используйте обратные кавычки:

```go
// @tg http-path=`/api/v1/users/:id`
```

### Подстановка из файла

В любом значении аннотации можно указать подстановку из файла:

- `file:путь` — подставится содержимое файла (путь относительно корня проекта).
- `file:путь#Заголовок` — подставится только секция markdown с указанным заголовком.

Пример: длинное описание для OpenAPI можно вынести в файл и указать `file:docs/api.md#Description`.

## Уровни аннотаций

1. **Пакет** — действует на все интерфейсы и методы в пакете.
2. **Интерфейс** — на все методы интерфейса.
3. **Метод** — только на этот метод.
4. **Поле/параметр** — на поле структуры или параметр/результат метода.

**Приоритет:** метод переопределяет интерфейс, интерфейс — пакет.

## Список аннотаций

### Уровень пакета

| Аннотация | Описание | Пример |
|-----------|----------|--------|
| `log` | Логирование запросов по пакету | `// @tg log` |
| `trace` | Трассировка запросов по пакету | `// @tg trace` |
| `metrics` | Сбор метрик по пакету | `// @tg metrics` |
| `http-prefix=<префикс>` | Префикс URL для методов пакета | `// @tg http-prefix=api/v1` |
| `packageJSON=<пакет>` | Другой JSON-кодек (по умолчанию `encoding/json`) | `// @tg packageJSON=...` |
| `uuidPackage=<пакет>` | Пакет для UUID (по умолчанию `github.com/google/uuid`) | `// @tg uuidPackage=...` |
| `swaggerTags=<тег1,тег2>` | Теги в OpenAPI | `// @tg swaggerTags=users,api` |
| `security=bearer` | Тип авторизации в OpenAPI | `// @tg security=bearer` |
| `servers=<адрес>;<имя>` | Серверы в OpenAPI | `// @tg servers=https://api.example.com;Production` |
| `version=<версия>` | Версия в документации | `// @tg version=1.0.0` |
| `title=<заголовок>` | Заголовок OpenAPI | `// @tg title=My API` |
| `author=<автор>` | Автор NPM-пакета (JS-клиенты) | `// @tg author=John Doe` |
| `npmRegistry=<адрес>` | Адрес NPM-репозитория | `// @tg npmRegistry=...` |
| `npmName=<имя>` | Имя NPM-пакета | `// @tg npmName=@myorg/my-api-client` |
| `npmPrivate=<true\|false>` | Приватность NPM-пакета | `// @tg npmPrivate=true` |
| `license=<лицензия>` | Лицензия NPM-пакета | `// @tg license=MIT` |

### Уровень интерфейса

| Аннотация | Описание | Пример |
|-----------|----------|--------|
| `jsonRPC-server` | Включить JSON-RPC 2.0 сервер для интерфейса | `// @tg jsonRPC-server` |
| `http-server` | Включить HTTP-сервер для интерфейса | `// @tg http-server` |
| `log` | Логирование запросов по интерфейсу | `// @tg log` |
| `trace` | Трассировка по интерфейсу | `// @tg trace` |
| `metrics` | Метрики по интерфейсу | `// @tg metrics` |
| `http-prefix=<префикс>` | Префикс URL для методов интерфейса | `// @tg http-prefix=api/v1` |
| `swaggerTags=<тег1,тег2>` | Теги в OpenAPI | `// @tg swaggerTags=users,api` |
| `tagOmitemptyAll` | Добавить `omitempty` ко всем полям запроса/ответа | `// @tg tagOmitemptyAll` |
| `desc=<описание>` | Краткое описание интерфейса | `// @tg desc=User management service` |

### Уровень метода

| Аннотация | Описание | Пример |
|-----------|----------|--------|
| `http-method=<метод>` | HTTP-метод (GET, POST, PUT, PATCH, DELETE, OPTIONS) | `// @tg http-method=POST` |
| `http-path=<путь>` | URL-путь метода, поддерживает параметры (`:id`) | `// @tg http-path=/users/:id` |
| `http-success=<код>` | HTTP-код успеха (по умолчанию 200) | `// @tg http-success=201` |
| `http-args=<переменная>\|<ключ>` | Связь параметра URL с аргументом метода | `// @tg http-args=id\|userId` |
| `http-headers=<переменная>\|<заголовок>` | Связь заголовка с аргументом/результатом | `// @tg http-headers=token\|Authorization` |
| `http-cookies=<переменная>\|<cookie>` | Связь cookie с аргументом/результатом | `// @tg http-cookies=session\|sessionId` |
| `http-response=<модуль>:<метод>` | Свой обработчик ответа (go-fiber) | `// @tg http-response=handlers:CustomHandler` |
| `handler=<модуль>:<метод>` | Полностью свой обработчик | `// @tg handler=handlers:CustomHandler` |
| `requestContentType=<mime>` | MIME запроса (по умолчанию `application/json`) | `// @tg requestContentType=application/xml` |
| `responseContentType=<mime>` | MIME ответа | `// @tg responseContentType=application/xml` |
| `enableInlineSingle` | Inline для метода с одним возвращаемым значением | `// @tg enableInlineSingle` |
| `http-multipart` | Режим multipart для запроса/ответа | `// @tg http-multipart` |
| `http-part-name=<аргумент>\|<часть>` | Имя части в multipart | `// @tg http-part-name=body\|file1` |
| `http-part-content=<аргумент>\|<mime>` | Content-Type части в multipart | `// @tg http-part-content=body\|image/png` |
| `log-skip=<переменная>` | Не логировать указанные переменные | `// @tg log-skip=password` |
| `deprecated` | Пометка метода как устаревшего в OpenAPI | `// @tg deprecated` |
| `summary=<описание>` | Описание метода для OpenAPI | `// @tg summary=Creates a new user` |
| `desc=<описание>` | Краткое описание метода | `// @tg desc=Create user endpoint` |
| `swaggerTags=<тег1,тег2>` | Теги в OpenAPI (переопределяют интерфейс) | `// @tg swaggerTags=users` |

### Уровень поля/параметра

Комментарии перед полем структуры или параметром/результатом метода:

| Аннотация | Описание | Пример |
|-----------|----------|--------|
| `desc=<описание>` | Описание поля | `// @tg desc=User identifier` |
| `type=<тип>` | Тип в OpenAPI | `// @tg type=string` |
| `enums=val1,val2` | Допустимые значения | `// @tg enums=active,inactive,pending` |
| `format=<формат>` | Формат в OpenAPI (uuid, email, date-time и т.д.) | `// @tg format=uuid` |
| `required` | Поле обязательно | `// @tg required` |
| `example=<значение>` | Пример для документации | `// @tg example=550e8400-...` |
| `<переменная>.tags=<тег>:<значение>` | Теги (напр. `dumper:hide` для логов) | `// @tg user.tags=dumper:hide` |
| `http-part-name=<имя>` | Имя части в multipart (для `io.Reader`/`io.ReadCloser`) | `// @tg http-part-name=file1` |
| `http-part-content=<mime>` | Content-Type части в multipart | `// @tg http-part-content=image/png` |
| `log-skip` | Не логировать это поле | `// @tg log-skip` |

## Примеры контрактов

### Базовый интерфейс

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

### HTTP: заголовки и cookies

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

### Параметры в URL

```go
// @tg http-server
type UserService interface {
// @tg http-method=GET
// @tg http-path=/users/:id/posts
// @tg http-args=id|userId
GetUserPosts(ctx context.Context, id string) (posts []Post, err error)
}
```

### Свой обработчик

```go
// @tg http-server
type FileService interface {
// @tg http-method=POST
// @tg http-path=/upload
// @tg handler=handlers:FileUploadHandler
UploadFile(ctx context.Context, file []byte) (url string, err error)
}
```

### Аннотации полей

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

### Разный тип контента (XML)

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

### Inline-ответ для одного значения

```go
// @tg http-server
type StatusService interface {
// @tg http-method=GET
// @tg http-path=/status
// @tg enableInlineSingle
GetStatus(ctx context.Context) (status string, err error)
}
```

### Наследование аннотаций

```go
// Пакет
// @tg log metrics
// @tg http-prefix=api/v1

// Интерфейс переопределяет префикс
// @tg http-prefix=api/v2
// @tg trace
type UserService interface {
// Метод переопределяет для себя
// @tg http-method=GET
// @tg http-path=/users
// @tg log-skip=password
GetUsers(ctx context.Context) (users []User, err error)
}
```

Для `GetUsers`: используются `log`, `metrics`, `trace` (от пакета и интерфейса), префикс URL — `api/v2`, метод и путь заданы на уровне метода.

## Правила оформления контрактов

### Именованные аргументы и результаты

Все параметры методов и все возвращаемые значения (кроме `error`) должны быть с именами — они используются в транспорте (например, в JSON-RPC):

```go
// Правильно
Method(ctx context.Context, userID string, count int) (result string, total int, err error)

// Неправильно
Method(ctx context.Context, string, int) (string, int, error)
```

### Контекст и ошибка

- Первый аргумент метода — `context.Context`.
- Последний возвращаемый тип — `error`.

## Формат JSON-RPC 2.0

Для метода `UserService.GetUser` запрос и ответ выглядят так:

**Запрос:**

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

**Ответ:**

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

При запуске с уровнем лога `debug` (например, флаг `--log-level debug` на хосте) плагин записывает в проект файл `.tg/project.json` — полную модель в JSON (контракты, методы, типы, аннотации). Его можно открыть для проверки того, как плагин увидел ваш проект.

## Ограничения

1. Интерфейсы **без** аннотаций `@tg` не учитываются.
2. Методы без своих аннотаций наследуют настройки от интерфейса и пакета.
3. Учитываются только **экспортируемые** интерфейсы (с заглавной буквы).
4. Встроенные интерфейсы (embedded) в контрактах не обрабатываются.
5. Контракты ищутся **только** в указанной директории контрактов; только `.go` файлы в ней самой, без поддиректорий.
6. Файлы с заголовком генерации tgp не участвуют в поиске сервисов и реализаций и не входят в набор файлов для кэша.
