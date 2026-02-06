# Swagger Plugin - Генератор OpenAPI документации

## Описание

Плагин `swagger` генерирует OpenAPI 3.0 документацию для контрактов с аннотациями `@tg`. Документация полностью соответствует спецификации OpenAPI 3.0 и может использоваться с любыми инструментами, поддерживающими эту спецификацию (Swagger UI, Postman, Insomnia и др.).

Для получения общей информации о парсере и поддерживаемых аннотациях используйте команду `tg plugin doc astg`.

## Основные возможности

- **OpenAPI 3.0** - полная поддержка спецификации OpenAPI 3.0
- **HTTP endpoints** - автоматическая генерация документации для всех HTTP методов
- **JSON-RPC endpoints** - документация для JSON-RPC методов в формате OpenAPI
- **Схемы данных** - автоматическая генерация схем для всех типов данных
- **Параметры** - документация параметров пути, query, заголовков и cookies
- **Примеры** - генерация примеров запросов и ответов
- **Безопасность** - поддержка схем авторизации (Bearer token и др.)
- **Теги** - группировка endpoints по тегам
- **Множественные форматы** - поддержка JSON и YAML форматов

## Использование

### Базовый пример

```bash
tg plugin swagger --out api-docs/openapi.yaml
```

### С фильтрацией контрактов

```bash
tg plugin swagger --out api-docs/openapi.yaml --contracts UserService,OrderService
```

### Генерация в JSON формате

```bash
tg plugin swagger --out api-docs/openapi.json
```

## Структура генерируемой документации

### Информация о API (`info`)

Содержит:

- `title` - название API (из аннотации `@tg title` или имя модуля)
- `version` - версия API (из аннотации `@tg version` или "1.0.0")
- `description` - описание API (из аннотации `@tg desc`)

### Серверы (`servers`)

Список серверов для документации (из аннотации `@tg servers`):

```go
// @tg servers=https://api.example.com;Production|https://staging.example.com;Staging
```

### Безопасность (`security`)

Схемы авторизации (из аннотации `@tg security`):

```go
// @tg security=bearer
```

Поддерживается Bearer token авторизация.

### Пути (`paths`)

Для каждого метода контракта генерируется путь с:

- HTTP методом (GET, POST, PUT, PATCH, DELETE)
- Параметрами пути
- Query параметрами
- Телом запроса (для POST/PUT/PATCH)
- Заголовками и cookies
- Ответами с кодами статуса
- Схемами ошибок

### Схемы (`components.schemas`)

Автоматически генерируются схемы для всех типов данных:

- Структуры запросов и ответов
- Вложенные типы
- Массивы, слайсы, мапы
- Базовые типы (string, int, bool и др.)
- Опциональные поля
- Enum значения (из аннотации `@tg enums`)

## Особенности генерации

### HTTP endpoints

Для методов с аннотацией `@tg http-server` и `@tg http-method`:

1. **Путь**: Используется аннотация `http-path` или путь по умолчанию
2. **Параметры пути**: Автоматически извлекаются из пути (например, `:id`)
3. **Query параметры**: Параметры, не являющиеся частью пути, заголовков или cookies
4. **Тело запроса**: Для POST/PUT/PATCH методов формируется из параметров метода; при аргументах `io.Reader` — как `application/octet-stream` (один поток) или `multipart/form-data` (несколько частей или аннотация `http-multipart`); имена и Content-Type частей — аннотации `http-part-name`, `http-part-content`
5. **Заголовки**: Маппятся через аннотацию `http-headers`
6. **Cookies**: Маппятся через аннотацию `http-cookies`
7. **Типы контента**: Поддерживаются через `requestContentType` и `responseContentType`
8. **Ответ с потоком**: При возвращаемых `io.ReadCloser` — ответ описывается как `application/octet-stream` или `multipart/form-data` (при нескольких частях или `http-multipart`)

### JSON-RPC endpoints

Для методов с аннотацией `@tg jsonRPC-server`:

1. **Путь**: Используется аннотация `http-path` или путь по умолчанию
2. **Метод**: POST (JSON-RPC всегда использует POST)
3. **Тело запроса**: JSON-RPC 2.0 формат с полями `jsonrpc`, `method`, `params`, `id`
4. **Тело ответа**: JSON-RPC 2.0 формат с полями `jsonrpc`, `result` или `error`, `id`
5. **Имя метода**: Формируется как `{contractName}.{methodName}` в camelCase

### Схемы данных

1. **Автоматическое определение типов**: Go типы автоматически преобразуются в OpenAPI типы
2. **Вложенные структуры**: Рекурсивно обрабатываются все вложенные типы
3. **Массивы и слайсы**: Преобразуются в `array` с соответствующим `items`
4. **Мапы**: Преобразуются в `object` с `additionalProperties`
5. **Указатели**: Обрабатываются как nullable типы
6. **Enum**: Поддерживаются через аннотацию `@tg enums=val1,val2`

### Параметры

1. **Параметры пути**: Автоматически определяются из `http-path` (например, `/users/:id`)
2. **Query параметры**: Все параметры, не являющиеся частью пути, заголовков или cookies
3. **Заголовки**: Маппятся через аннотацию `http-headers`
4. **Cookies**: Маппятся через аннотацию `http-cookies`
5. **Обязательные параметры**: Определяются через аннотацию `@tg required`

### Ответы

1. **Успешные ответы**: Код статуса из аннотации `http-success` (по умолчанию 200)
2. **Ошибки**: Автоматически добавляются стандартные коды ошибок (400, 401, 403, 404, 500)
3. **Схемы ответов**: Автоматически генерируются из возвращаемых значений методов
4. **Заголовки ответов**: Маппятся через аннотацию `http-headers` в результатах

### Примеры

Примеры генерируются автоматически на основе:

- Типов данных
- Аннотации `@tg example`
- Значений по умолчанию

## Опции командной строки

- `out` (string, опциональная) - путь к выходному файлу (поддерживаются .json и .yaml/.yml)
- `serve` (string, опциональная) - запустить HTTP-сервер с Swagger UI по указанному адресу (например: `:8080` или `localhost:3000`)
- `contracts` (string, опциональная) - список контрактов через запятую для фильтрации (например: "UserService,OrderService")

## Примеры использования

### Генерация документации для всех контрактов

```bash
tg plugin swagger --out docs/openapi.yaml
```

### Генерация документации для конкретных контрактов

```bash
tg plugin swagger --out docs/openapi.yaml --contracts UserService,OrderService
```

### Генерация в JSON формате

```bash
tg plugin swagger --out docs/openapi.json
```

### Использование с Swagger UI

```bash
# Генерация документации
tg plugin swagger --out docs/openapi.yaml

# Запуск Swagger UI с помощью Docker
docker run -p 8080:8080 -e SWAGGER_JSON=/docs/openapi.yaml -v $(pwd)/docs:/docs swaggerapi/swagger-ui
```

### Использование с Postman

1. Сгенерируйте документацию:

```bash
tg plugin swagger --out docs/openapi.json
```

2. Импортируйте в Postman:
    - File → Import
    - Выберите файл `openapi.json`

## Аннотации для настройки документации

### Уровень пакета

```go
// @tg title=My API
// @tg version=1.0.0
// @tg desc=API для управления пользователями
// @tg servers=https://api.example.com;Production|https://staging.example.com;Staging
// @tg security=bearer
```

### Уровень интерфейса

```go
// @tg swaggerTags=users,api
// @tg desc=Сервис для управления пользователями
type UserService interface {
    // ...
}
```

### Уровень метода

```go
// @tg http-method=GET
// @tg http-path=/users/:id
// @tg summary=Получить пользователя по ID
// @tg desc=Возвращает информацию о пользователе с указанным ID
// @tg deprecated
// @tg swaggerTags=users
GetUser(ctx context.Context, id string) (user User, err error)
```

### Уровень поля

```go
type User struct {
    // @tg required
    // @tg desc=Уникальный идентификатор пользователя
    // @tg format=uuid
    // @tg example=550e8400-e29b-41d4-a716-446655440000
    ID string `json:"id"`
    
    // @tg desc=Имя пользователя
    // @tg example=John Doe
    Name string `json:"name"`
    
    // @tg desc=Роль пользователя
    // @tg enums=admin,user,guest
    Role string `json:"role"`
}
```

## Пример сгенерированной документации

```yaml
openapi: 3.0.0
info:
  title: My API
  version: 1.0.0
  description: API для управления пользователями
servers:
  - url: https://api.example.com
    description: Production
security:
  - BearerAuth: []
paths:
  /api/v1/users/{id}:
    get:
      summary: Получить пользователя по ID
      description: Возвращает информацию о пользователе с указанным ID
      tags:
        - users
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '200':
          description: Успешный ответ
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
        '400':
          description: Неверный запрос
        '404':
          description: Пользователь не найден
components:
  schemas:
    User:
      type: object
      required:
        - id
        - name
      properties:
        id:
          type: string
          format: uuid
          description: Уникальный идентификатор пользователя
          example: 550e8400-e29b-41d4-a716-446655440000
        name:
          type: string
          description: Имя пользователя
          example: John Doe
        role:
          type: string
          description: Роль пользователя
          enum:
            - admin
            - user
            - guest
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
```

## Интеграция с инструментами

### Swagger UI

```bash
# Установка Swagger UI
npm install -g swagger-ui-serve

# Запуск с документацией
swagger-ui-serve docs/openapi.yaml
```

### Redoc

```bash
# Установка Redoc
npm install -g redoc-cli

# Генерация статической документации
redoc-cli bundle docs/openapi.yaml -o docs/index.html
```

### Postman

1. Сгенерируйте документацию в JSON формате
2. Импортируйте в Postman через File → Import
3. Все endpoints будут автоматически добавлены в коллекцию

### Insomnia

1. Сгенерируйте документацию
2. Импортируйте через Application → Preferences → Data → Import Data → From File
3. Выберите формат OpenAPI

## Зависимости

Плагин зависит от:

- `astg` - для парсинга контрактов и аннотаций

Генерируемая документация не требует зависимостей - это обычный YAML/JSON файл, соответствующий спецификации OpenAPI 3.0.

## Ограничения

1. Поддерживается только OpenAPI 3.0 (не 2.0/Swagger 2.0)
2. Для JSON-RPC методов путь всегда POST
3. Не все Go типы могут быть точно представлены в OpenAPI (например, интерфейсы)
4. Кастомные маршаллеры могут требовать ручной настройки схем

## Совместимость

Плагин полностью совместим с контрактами, сгенерированными плагином `server`. Все аннотации из плагина `astg` поддерживаются и учитываются при генерации документации.

## Дополнительные ресурсы

- [OpenAPI Specification](https://swagger.io/specification/)
- [Swagger UI](https://swagger.io/tools/swagger-ui/)
- [Redoc](https://github.com/Redocly/redoc)
