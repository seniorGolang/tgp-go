# Плагин swagger — OpenAPI 3.0 из контрактов

## Назначение

Плагин строит спецификацию **OpenAPI 3.0** (JSON/YAML) по HTTP-семейству контрактов и при необходимости поднимает Swagger UI. Источник правды — контракты `@tg`, не ручная правка файла.

Полный синтаксис аннотаций: `tg plugin doc astg`. Тексты API (`summary`/`desc`/`file:`) — skill `tgp-contracts`.

## Когда использовать

- Нужен OpenAPI для Swagger UI, Postman, Redoc, контрактного review.
- Контракты с `http-server`, `jsonRPC-server`, `ws-server`, `sse-server`.
- **Kafka в OpenAPI не попадает** — не ждите топиков здесь.

## Команда и параметры

```bash
tg swagger --out api-docs/openapi.yaml
tg swagger --serve :8080
tg swagger --out api-docs/openapi.json --serve :8080
tg swagger   # без флагов → UI на :8080
```

| Параметр | Короткая форма | Обязательный | Описание |
|----------|----------------|--------------|----------|
| `out` | **нет** (только `--out`) | нет* | Файл `.json` / `.yaml` / `.yml`; каталог создаётся |
| `serve` | — | нет* | Адрес UI (например `:8080`, `localhost:3000`) |
| `contracts` | — | нет | Имена через запятую; поддержка **исключений `!Name`** |
| `contracts-dir` | — | нет | Каталог контрактов (default `contracts`) |

\* Нужен хотя бы один режим: файл, serve, или оба. Если нет ни `out`, ни `serve` — по умолчанию serve на **`:8080`**.

Примеры фильтра:

```bash
tg swagger --out docs/openapi.yaml --contracts UserService,OrderService
tg swagger --out docs/openapi.yaml --contracts '!OrderService'
```

У server/client фильтр обычно **без** `!Name`; у swagger исключения поддерживаются.

## Что генерируется

### Матрица: транспорт → OpenAPI

| На контракте | В спецификации |
|--------------|----------------|
| `http-server` | REST `paths` + method |
| `jsonRPC-server` | `POST` на путь контракта; body JSON-RPC 2.0 (`params`/`result`) |
| оба | оба набора операций |
| `ws-server` + stream | операция с маркером `x-websocket: true` |
| `sse-server` + stream | `text/event-stream` |
| `kafka` | **опущено** |
| пакетные `title`/`version`/`desc`/`servers`/`security` | `info`, `servers`, `components.securitySchemes` |
| `swaggerTags` / `tagDesc.*` | `tags` и `operation.tags` |

### Артефакт

Один документ OpenAPI 3.0:

- `info`, `servers`, `security` / `components.securitySchemes`
- `paths` — операции
- `components.schemas` — типы
- `tags` — группировка в UI

Формат выбирается расширением файла. Режим `--serve` отдаёт JSON в UI.

## Аннотации и их влияние

Ниже — **эффект на OpenAPI**, не каталог всего astg.

### Метаданные пакета → `info` / `servers` / security

| Аннотация | Уровень | Эффект |
|-----------|---------|--------|
| `title` | пакет | `info.title` |
| `version` | пакет | `info.version` (иначе часто `1.0.0`) |
| `desc` | пакет | `info.description` (md) |
| `servers` | пакет | `servers[]` (`url;desc` через `\|`) |
| `security` | пакет | схемы и глобальный `security` (см. ниже) |

**`@tg security` (app2app):**

| Форма | В OpenAPI |
|-------|-----------|
| `http:bearer[:JWT][|jwks=…]` | `http` bearer + опц. `x-jwks-uri`; в глобальном `security` |
| `apiKey:<in>:<name>` | `apiKey`; в `security` |
| `oauth2:tokenExchange` / `oauth2:clientCredentials[:jwt-bearer]` | только `components.securitySchemes` (`x-grant-type`, `tokenUrl={authorizationServer}`) |
| `openId` / URL в аннотации | **отклоняется** |

Адреса AS/API — runtime опции server/client, не строка `security`.

### Группировка и тексты операций

| Аннотация | Уровень | Эффект |
|-----------|---------|--------|
| `swaggerTags` | iface / method | `operation.tags` (method побеждает iface) |
| `tagDesc.<tag>` | iface | `tags[].description` |
| `desc` | iface | fallback описания тега |
| `summary` | method | `operation.summary` |
| `desc` / `file:…` | method | `operation.description` |
| `deprecated` | method | `operation.deprecated=true` |
| `requestBodyDesc` | method | `requestBody.description` |

Пустые summary/description правятся в контрактах, не в YAML.

### Маршруты и параметры

| Аннотация | Эффект в OpenAPI |
|-----------|------------------|
| `http-prefix` + `http-path` | ключ `paths` (сегменты `:name` → `{name}`) |
| `http-method` | verb; default POST при `http-server` без явного method |
| `http-args` / `http-headers` / `http-cookies` | `parameters` при mode `explicit`/`implicit` |
| mode `body` | остаётся в schema тела, **не** отдельный parameter |
| `http-success` | код успешного response (default 200) |
| `requestContentType` / `responseContentType` | media types |
| `http-multipart` / part-* | multipart / octet-stream схемы |
| `enableInlineSingle` | schema успеха без лишней обёртки |
| `http-response` | абстрактное описание кастомного ответа |
| `*.tags` (`json:inline`, `form:…`) | inline / имена form-полей |

### Поля и `required`

| Аннотация | Эффект |
|-----------|--------|
| `required` / `<var>.required` | явная обязательность |
| `desc` / `format` / `example` / `type` | schema/parameter metadata |
| `enums` на голом скаляре | `schema.enum` |
| именованный тип + typed const (≥2) | `$ref` + `enum` (без обязательного `@tg enums`) |

**Вывод `required`, если явно не задан:**

| Контекст | Обязательно, если… |
|----------|-------------------|
| Поле тела **запроса** | не указатель и нет `omitempty` |
| Поле тела **ответа** | нет `omitempty` (указатель без omitempty всё ещё required) |
| **Query** | не указатель; явный `required` делает обязательным и указатель |
| **Header** / **cookie** | только явный `required` |

Ошибки в `responses` — только те коды/типы, что есть в модели метода. Стандартные 400/401/404/500 **не** синтезируются.

## Использование

1. Опишите контракты и документационные аннотации.
2. Сгенерируйте канонический файл в репозиторий.
3. Diff `paths` / schemas / security перед merge.
4. При необходимости `--serve` для визуальной проверки (не замена semantic review).

```bash
tg swagger --out docs/openapi.yaml
tg swagger --out docs/openapi.json --serve :8080
jq -e '.openapi == "3.0.0" and (.paths | length > 0)' docs/openapi.json
jq '.paths, .components.schemas, .components.securitySchemes' docs/openapi.json
```

Импорт: Postman/Insomnia ← JSON/YAML; Redoc ← тот же файл.

### Что проверить в diff перед merge

| Зона | Считать breaking, если… |
|------|-------------------------|
| `paths` / methods | удалили path или verb |
| parameters | убрали параметр или сделали required строже |
| request/response schemas | несовместимая смена типа / enum |
| security | убрали схему, на которую опираются клиенты |
| `operationId` / tags | переименование ломает сгенерированные клиенты внешних tooling |

Совместимые: новые optional поля, новые paths, уточнение `description`/`example`.

### Mapping modes → parameters (кратко)

| Mode | В `parameters`? | В body schema? |
|------|-----------------|----------------|
| `explicit` | да (`header`/`cookie`/`query`) | нет дубля |
| `implicit` | да (документируется) | нет дубля |
| `body` | нет | да |

Path placeholders (`:id`) всегда требуют соответствующего аргумента метода.

### Enums в схемах

| Источник | OpenAPI |
|----------|---------|
| `@tg enums=a,b` на голом скаляре | inline `enum` |
| именованный тип + typed const (≥2) | `$ref` на schema с `enum` |
| override `@tg enums` на поле | локальный перечень |

### Пакетный блок — пример

```go
// @tg title=`User Service API`
// @tg version=1.2.3
// @tg desc=`API пользователей`
// @tg servers=`https://api.example.com;Production|https://staging.example.com;Staging`
// @tg security=`http:bearer:JWT,oauth2:tokenExchange`
package contracts
```

`http:bearer` попадёт в глобальный `security`; `oauth2:tokenExchange` — в `components.securitySchemes` (способ получения токена), без URL в аннотации.

## На что обратить внимание

1. Только **`--out`**, не `-o`.
2. Kafka отсутствует намеренно.
3. Mode `body` → параметр «пропал» из `parameters`, остался в body.
4. `openId` в `security` отвергается; oauth2 не всегда в глобальном `security`.
5. Неправильный `required` чаще из inference, чем из бага генератора.
6. Типы с кастомным MarshalJSON часто дают opaque object.
7. Фильтр `!Name` — фича swagger; не копируйте слепо на server/clients.
8. Не редактируйте OpenAPI вручную как source of truth.

## Ограничения

1. Только OpenAPI **3.0** (не Swagger 2.0).
2. JSON-RPC описан как HTTP POST + JSON-RPC envelope.
3. Не все Go-типы (интерфейсы, custom marshalers) дают полную schema.
4. Без prose в контракте UI будет «пустым» — это не баг swagger.

## Связь с другими плагинами

| Плагин | Связь |
|--------|-------|
| `tgp-contracts` | тексты и аннотации документации |
| `astg-json` | проверка resolved model до генерации |
| `server` / clients | тот же HTTP-семейный surface |
| `init-go` | часто кладёт стартовый `api/swagger.json` |
| Kafka-плагины | вне OpenAPI |
