# Плагин astg — единая модель контрактов `@tg`

## Назначение

`astg` разбирает Go-проект и собирает **единую resolved-модель**: контракты (экспортируемые интерфейсы с `@tg`), методы, типы, аннотации, ошибки, provenance (module/git). Эта модель — вход для server, client-go, client-ts, swagger, kafka-pub-go, kafka-sub-go и для инспекции через `astg-json`.

Вы пишете контракты и DTO; генераторы читают уже разрешённую модель, а не «угадывают» по комментариям заново.

## Когда использовать

- Любая генерация кода/OpenAPI/Kafka по контрактам (модель строит `astg` в пайплайне).
- Проверка, что аннотации и типы разрешились так, как задумано (`tg astg json`).
- Пересборка модели после правок `contracts/` (или принудительно с `no-cache`).

Отдельно команду `astg` обычно не вызывают: её поднимают command-плагины. Для явного экспорта — `tg astg json` (`astg-json`).

## Опции пайплайна

`contracts-dir` и `no-cache` читаются из **запроса пайплайна**. Их передают command-плагины (`tg server`, `tg swagger`, `tg astg json`, …), а не отдельная CLI у самого `astg`.

| Опция | Тип | Описание |
| ----- | --- | -------- |
| `contracts-dir` | строка | Папка контрактов. По умолчанию `contracts`. Читаются только `.go` **в самой этой папке** (вложенные каталоги для поиска интерфейсов не сканируются). |
| `no-cache` | bool | Не брать кэш — полный разбор заново. |

Опция `--contracts` у генераторов (server, swagger, clients, …) **не сужает** снимок `astg`: в пайплайн всегда кладётся полный проект. Фильтр применяют сами command-плагины на копии. Post-плагин `astg-hook` сохраняет в локальную БД именно полный снимок.

## Кэш

Кэш ускоряет повторные запуски. Он актуален, пока не изменились учтённые файлы: `go.mod`, `go.sum` и релевантные `.go` проекта. Каталоги `.tg`, `.git`, `vendor` не входят в расчёт; файлы с заголовком генерации tgp тоже игнорируются. Принудительный полный разбор — `no-cache`.

## Как писать контракты (layout `contracts/` + `contracts/dto`)

Рекомендуемая (и ожидаемая генераторами) раскладка:

| Путь | Содержимое |
| ---- | ---------- |
| `contracts/*.go` | Только интерфейсы с `@tg` и пакетные аннотации. Плоский каталог: вложенные папки **не** сканируются на контракты. |
| `contracts/dto/*.go` | DTO и shared-типы (`package dto`). В сигнатурах — `dto.User`, `dto.CreateUserRequest`. |

Правила:

- Не кладите payload-структуры в `package contracts` рядом с интерфейсами.
- Не кладите `@tg`-интерфейсы в `contracts/dto` или во вложенные папки — они не будут найдены как контракты.
- DTO резолвятся через импорты/`go/types`: импортируемые пакеты (включая `contracts/dto`) участвуют в графе типов; **искать интерфейсы** `astg` будет только в `contracts-dir`.

Минимальный каркас:

```go
package contracts

import "<module>/contracts/dto"

// @tg http-server
// @tg http-prefix=api/v1
type UserService interface {
	// @tg http-method=GET
	// @tg http-path=/users/:id
	// @tg http-args=id|userId|explicit
	GetUser(ctx context.Context, id string) (user dto.User, err error)
}
```

## Синтаксис `@tg`

Аннотации — в комментариях с префиксом `@tg`:

```go
// @tg <имя>[=<значение>] [<имя2>[=<значение2>] ...]
```

Примеры:

```go
// @tg log
// @tg http-prefix=api/v1
// @tg jsonRPC-server log metrics
// @tg http-prefix=api/v1
// @tg jsonRPC-server
```

### Значения с пробелами и спецсимволами

Значение без кавычек обрывается на первом пробеле (остаток парсер воспринимает как следующие токены). Чтобы сохранить пробелы, пути, `|` и прочее целиком — оборачивайте значение в **обратные кавычки**:

```go
// @tg http-path=`/api/v1/users/:id`
// @tg summary=`Custom fiber handler`
// @tg desc=`Bearer token from Authorization header`
// @tg security=`http:bearer:JWT`
```

Без кавычек `summary=Custom fiber handler` даст `summary=Custom`, а `fiber` и `handler` станут отдельными (обычно бесполезными) ключами.

### Подстановка из файла

В любом значении допустима подстановка:

- `file:путь` — содержимое файла (путь от корня проекта);
- `file:путь#Заголовок` — секция markdown с указанным заголовком.

Удобно для длинных OpenAPI-описаний: `file:docs/api.md#Description`.

## Уровни и приоритет

Уровни:

1. **Пакет** — на все интерфейсы и методы пакета.
2. **Интерфейс** — на все методы интерфейса.
3. **Метод** — на метод (включая sub-аннотации аргументов/результатов).
4. **Поле / параметр** — комментарий перед полем структуры или перед параметром/результатом метода.

**Приоритет** (выше перекрывает ниже):

1. поле / параметр;
2. sub-аннотация на методе (`<имя>.<ключ>`);
3. метод;
4. интерфейс;
5. пакет.

Inherited-аннотации **не копируются** в каждый узел JSON-модели: эффективное значение вычисляют генераторы по этой цепочке. При инспекции через `tg astg json` смотрите аннотации на исходном уровне и мысленно применяйте cascade.

## Sub-аннотации

На методе можно задать свойства аргумента или результата через префикс имени из сигнатуры:

```go
// @tg token.required
// @tg token.desc=`Bearer token from Authorization header`
// @tg result.tags=`json:inline`
GetProfile(ctx context.Context, token string) (result dto.Profile, err error)
```

Форма: `// @tg <имяПеременной>.<ключ>[=значение]`.

Поддерживаемые ключи: `required`, `desc`, `format`, `example`, `enums`, `type`, `tags`, `log-skip`, `http-part-name`, `http-part-content`.

Специальный ключ **`tags`** — теги сериализации через `|`:

| Значение в `tags` | Назначение |
| ----------------- | ---------- |
| `json:inline` | Развернуть поле/результат в родительский JSON-объект |
| `json:<имя>,omitempty` | Переопределить `json`-тег для exchange-структур |
| `form:<имя>` | Имя поля при `application/x-www-form-urlencoded` |
| `dumper:hide` | Не выводить значение в логах (omit; варианты маскирования — через доп. флаги в значении) |

Прямой комментарий на поле/параметре **перекрывает** одноименную sub-аннотацию на методе.

## Каталог аннотаций

### Уровень пакета

| Аннотация | Описание | Пример |
| --------- | -------- | ------ |
| `log` | Логирование запросов по пакету | `// @tg log` |
| `trace` | Трассировка по пакету | `// @tg trace` |
| `metrics` | Метрики по пакету | `// @tg metrics` |
| `http-prefix=<префикс>` | Префикс URL для методов пакета | `// @tg http-prefix=api/v1` |
| `packageJSON=<пакет>` | JSON-кодек (по умолчанию `encoding/json`) | `// @tg packageJSON=...` |
| `uuidPackage=<пакет>` | Пакет UUID (по умолчанию `github.com/google/uuid`) | `// @tg uuidPackage=...` |
| `swaggerTags=<тег1,тег2>` | Теги OpenAPI | `// @tg swaggerTags=users,api` |
| `security=<схемы>` | App2app-схемы **без URL**: `http:bearer[:JWT][\|jwks=/path/:sub/…]`, `apiKey:<in>:<name>`, `oauth2:tokenExchange`, `oauth2:clientCredentials[:jwt-bearer]`. Форма `openId` **отклоняется**. Эндпоинты AS/API — runtime options генераторов, не значения аннотации | `// @tg security=\`http:bearer:JWT\`` |
| `servers=<адрес>;<имя>` | Серверы в OpenAPI | `// @tg servers=https://api.example.com;Production` |
| `version=<версия>` | Версия в документации | `// @tg version=1.0.0` |
| `title=<заголовок>` | Заголовок OpenAPI / пакета | `// @tg title=My API` |
| `desc=<описание>` | Описание API (`info.description` в OpenAPI) | `// @tg desc=\`User management API\`` |
| `author=<автор>` | Автор NPM-пакета (JS-клиенты) | `// @tg author=John Doe` |
| `npmRegistry=<адрес>` | Адрес NPM-репозитория | `// @tg npmRegistry=...` |
| `npmName=<имя>` | Имя NPM-пакета | `// @tg npmName=@myorg/my-api-client` |
| `npmPrivate=<true\|false>` | Приватность NPM-пакета | `// @tg npmPrivate=true` |
| `license=<лицензия>` | Лицензия NPM-пакета | `// @tg license=MIT` |

### Уровень интерфейса

| Аннотация | Описание | Пример |
| --------- | -------- | ------ |
| `jsonRPC-server` | JSON-RPC 2.0 (+ batch) | `// @tg jsonRPC-server` |
| `http-server` | HTTP REST | `// @tg http-server` |
| `ws-server` | WebSocket stream transport | `// @tg ws-server` |
| `sse-server` | SSE server-stream | `// @tg sse-server` |
| `kafka` | Контракт событий Kafka (kafka-pub-go / kafka-sub-go) | `// @tg kafka` |
| `log` / `trace` / `metrics` | Наблюдаемость по интерфейсу | `// @tg log` |
| `http-prefix=<префикс>` | Префикс URL | `// @tg http-prefix=api/v1` |
| `swaggerTags=<тег1,тег2>` | Теги OpenAPI | `// @tg swaggerTags=users,api` |
| `tagDesc.<тег>=<описание>` | Описание тега OpenAPI | `// @tg tagDesc.users=Операции с пользователями` |
| `tagOmitemptyAll` | `omitempty` на все поля request/response | `// @tg tagOmitemptyAll` |
| `desc=<описание>` | Описание интерфейса | `// @tg desc=User management service` |

REST и JSON-RPC могут жить на одном интерфейсе. Kafka — **отдельный** контракт: не смешивайте с http/jsonrpc/ws/sse/stream.

### Уровень метода

| Аннотация | Описание | Пример |
| --------- | -------- | ------ |
| `http-method=<метод>` | GET, POST, PUT, PATCH, DELETE, OPTIONS | `// @tg http-method=POST` |
| `stream=server\|client\|bidi` | Направление stream-метода | `// @tg stream=server` |
| `ws-path=<путь>` / `sse-path=<путь>` | Переопределить путь WS/SSE | `// @tg ws-path=/ws/live` |
| `kafka-topic=<топик>` | Топик Kafka (`@tg kafka`) | `// @tg kafka-topic=orders.created` |
| `kafka-key=<аргумент>` | Аргумент → Kafka key | `// @tg kafka-key=orderID` |
| `kafka-headers=<arg>\|<header>,…` | Аргументы → headers | `// @tg kafka-headers=traceID\|x-trace-id` |
| `kafka-message=<аргумент>` | Аргумент = тело записи | `// @tg kafka-message=event` |
| `kafka-codec=<имя>` | Кодек тела | `// @tg kafka-codec=json` |
| `kafka-acks=<режим>` | noAck / leaderAck / allISRAcks | `// @tg kafka-acks=allISRAcks` |
| `http-path=<путь>` | URL-путь (`:id` и т.п.) | `// @tg http-path=/users/:id` |
| `http-success=<код>` | HTTP-код успеха (по умолчанию 200) | `// @tg http-success=201` |
| `http-args=<arg>\|<ключ>[\|<mode>]` | Query/path; `mode`: `explicit`\|`implicit`\|`body` (default **`body`**) | `// @tg http-args=id\|userId\|explicit` |
| `http-headers=<arg>\|<заголовок>[\|<mode>]` | Header; тот же `mode` (default **`body`**) | `// @tg http-headers=token\|Authorization\|explicit` |
| `http-cookies=<arg>\|<cookie>[\|<mode>]` | Cookie; тот же `mode` (default **`body`**) | `// @tg http-cookies=session\|sessionId` |
| `<код>=<pkg>:<Type>` | HTTP-ошибка метода (коды **400–599**) для OpenAPI/клиентов | `// @tg 404=errs:NotFound` |
| `http-response=<модуль>:<метод>` | Свой обработчик ответа (Fiber) | `// @tg http-response=handlers:CustomHandler` |
| `handler=<модуль>:<метод>` | Полностью свой обработчик | `// @tg handler=handlers:CustomHandler` |
| `requestContentType=<mime>` | MIME запроса (default `application/json`) | `// @tg requestContentType=application/xml` |
| `responseContentType=<mime>` | MIME ответа | `// @tg responseContentType=application/xml` |
| `enableInlineSingle` | Inline для одного возвращаемого значения | `// @tg enableInlineSingle` |
| `http-multipart` | Multipart request/response | `// @tg http-multipart` |
| `http-part-name=<аргумент>\|<часть>` | Имя части multipart | `// @tg http-part-name=body\|file1` |
| `http-part-content=<аргумент>\|<mime>` | Content-Type части | `// @tg http-part-content=body\|image/png` |
| `log-skip=<переменная>` | Не логировать; `request`/`response` — весь атрибут | `// @tg log-skip=password` |
| `deprecated` | Устаревший метод в OpenAPI | `// @tg deprecated` |
| `summary=<описание>` | Краткое описание для OpenAPI | `// @tg summary=Creates a new user` |
| `desc=<описание>` | Описание метода | `// @tg desc=Create user endpoint` |
| `requestBodyDesc=<описание>` | Описание тела запроса в OpenAPI | `// @tg requestBodyDesc=Payload for user creation` |
| `swaggerTags=<тег1,тег2>` | Теги OpenAPI (перекрывают интерфейс) | `// @tg swaggerTags=users` |
| `<переменная>.<аннотация>` | Sub-аннотация аргумента/результата | `// @tg token.required` |

### Уровень поля / параметра

Те же ключи можно задать как sub-аннотации на методе; прямой комментарий на поле/параметре сильнее.

| Аннотация | Описание | Пример |
| --------- | -------- | ------ |
| `desc=<описание>` | Описание поля | `// @tg desc=User identifier` |
| `type=<тип>` | Тип в OpenAPI | `// @tg type=string` |
| `enums=val1,val2` | Допустимые значения (для голых скаляров; для именованных типов предпочтительны typed const) | `// @tg enums=active,inactive` |
| `format=<формат>` | uuid, email, date-time, … | `// @tg format=uuid` |
| `required` | Обязательное поле | `// @tg required` |
| `example=<значение>` | Пример для документации | `// @tg example=550e8400-...` |
| `http-part-name=<имя>` | Имя multipart-части (`io.Reader` / `io.ReadCloser`) | `// @tg http-part-name=file1` |
| `http-part-content=<mime>` | Content-Type части | `// @tg http-part-content=image/png` |
| `log-skip` | Не логировать поле | `// @tg log-skip` |

## Как аннотации влияют на генераторы

Обзор по семействам (детали — в `tg plugin doc <плагин>`):

| Семейство аннотаций | Кто в первую очередь использует | Что меняется у потребителя |
| ------------------- | ------------------------------- | -------------------------- |
| `http-server`, `http-method`/`path`/`prefix`, mapping | server, client-go, client-ts, swagger | REST-маршруты, сигнатуры, OpenAPI paths/parameters |
| `jsonRPC-server` | server, clients, swagger | RPC-методы `{contract}.{method}`, batch, schemas |
| `ws-server` / `sse-server`, `stream`, `ws-path`/`sse-path` | server, clients, swagger | Stream endpoints, каналы, события |
| `kafka`, `kafka-*` | kafka-pub-go, kafka-sub-go | Publisher/subscriber adapters (HTTP-семейство не трогает) |
| `log` / `metrics` / `trace` | server, clients, kafka | Опции наблюдения (`WithLog` / metrics / OTel) |
| `security` | swagger, server, clients | Security schemes, bearer/JWKS, token helpers |
| `summary` / `desc` / `title` / `version` / `servers` / `swaggerTags` / `tagDesc` / `requestBodyDesc` | swagger (и NPM meta у client-ts) | Тексты и группировка OpenAPI / package.json |
| `404=pkg:Type` и др. HTTP-ошибки | swagger, clients | Документированные/типизированные ошибки |
| `log-skip`, `tags` (`dumper:hide`, `json:inline`, `form:`) | server/clients (логи и wire), swagger | Payload логов, форма JSON/form, схемы |
| `handler` / `http-response` | server | Кастомные Fiber-обработчики вместо сгенерированных |
| NPM-мета (`npmName`, …) | client-ts | Метаданные публикуемого пакета |

Режимы маппинга `explicit` / `implicit` / `body` (default) меняют и серверный разбор, и клиентские сигнатуры, и OpenAPI parameters — задавайте режим осознанно, а не «по умолчанию body случайно».

## Форма wire

- **Typed enums.** Именованный тип + ≥2 typed `const` того же типа → `Type.Enums` (без обязательного `@tg enums`). Override `@tg enums=` — для голых скаляров или явного переопределения.
- **`json:inline` / anonymous embed.** Поле разворачивается в родительский JSON / OpenAPI / клиентские exchange-типы.
- **Opaque-типы** с собственным `MarshalJSON` / `UnmarshalJSON`: внутренние поля не тянутся в wire-схему (в OpenAPI часто «объект без детальной структуры»).
- **HTTP-ошибки метода:** `// @tg 404=errs:NotFound` (коды 400–599); плюс ошибки, найденные в реализациях/handler.
- **Mapping modes** на `http-headers` / `http-cookies` / `http-args`:
  - `explicit` — значение только через указанный HTTP-элемент, без дубля в body;
  - `implicit` — как explicit на HTTP, часто из middleware/контекста; в клиентах аргумент может быть скрыт;
  - `body` (default) — тело первично; HTTP-оверлей может переопределить (в т.ч. JSON-RPC context).
- **Form-urlencoded:** полям в body нужны явные `form:<name>` в `tags`.
- **JSON-RPC (кратко):** имя метода `{contract}.{method}` в camelCase; параметры в `params`, результат в `result`; batch — на стороне server/clients.

## Примеры контрактов

### REST + DTO

```go
package contracts

import "<module>/contracts/dto"

// @tg jsonRPC-server http-server
// @tg http-prefix=api/v1
// @tg swaggerTags=users
// @tg log metrics
type UserService interface {
	// @tg http-method=POST
	// @tg http-path=/users
	// @tg http-success=201
	// @tg summary=Creates a new user
	// @tg 409=errs:Conflict
	CreateUser(ctx context.Context, user dto.CreateUserRequest) (id string, err error)

	// @tg http-method=GET
	// @tg http-path=/users/:id
	// @tg http-args=id|userId|explicit
	// @tg 404=errs:NotFound
	GetUser(ctx context.Context, id string) (user dto.User, err error)

	// @tg http-method=GET
	// @tg http-path=/profile
	// @tg http-headers=token|Authorization|explicit
	// @tg token.required
	// @tg token.desc=`Bearer token`
	GetProfile(ctx context.Context, token string) (profile dto.Profile, err error)
}
```

### Поля DTO и typed enum

```go
package dto

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
	RoleGuest Role = "guest"
)

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
	Role Role `json:"role"`
}
```

Для голого `string` без именованного типа допустим override: `// @tg enums=admin,user,guest`.

### Свой обработчик и inline-ответ

```go
// @tg http-server
type FileService interface {
	// @tg http-method=POST
	// @tg http-path=/upload
	// @tg handler=handlers:FileUploadHandler
	UploadFile(ctx context.Context, file []byte) (url string, err error)
}

// @tg http-server
type StatusService interface {
	// @tg http-method=GET
	// @tg http-path=/status
	// @tg enableInlineSingle
	GetStatus(ctx context.Context) (status string, err error)
}
```

### Наследование уровней

Пакет `// @tg log metrics` + `http-prefix=api/v1`, интерфейс `http-prefix=api/v2` + `trace`, метод задаёт `http-method`/`http-path`/`log-skip` — у метода эффективны log/metrics с пакета, trace с интерфейса, префикс `api/v2`.

## Правила оформления

### Именованные аргументы и результаты

Все параметры и все возвраты (кроме `error`) — **с именами**: они входят в JSON-RPC params/result и в generated exchange.

```go
// правильно
Method(ctx context.Context, userID string, count int) (result string, total int, err error)

// неправильно
Method(ctx context.Context, string, int) (string, int, error)
```

### Контекст и ошибка

- Первый аргумент — `context.Context`.
- Последний возврат — `error`.

Для Kafka-методов обычно только `error` в возврате; для stream — каналы в направлении `stream=…`.

## Отладка модели

**Основной способ:** `tg astg json -o .tg/project.json` (плагин `astg-json`) — явный экспорт resolved-модели. Пишите в файл и смотрите точечно (`jq`), не вставляйте весь JSON в чат.

**Вторично:** при `--log-level debug` хост может записать `.tg/project.json` как побочный артефакт разбора — удобно, но не как основной канал инспекции.

С `--from-db` экспорт идёт из локальной БД (`astg-db`); локальные правки исходников в этот снимок не попадут, пока модель не сохранена заново через `astg-hook`.

## На что обратить внимание

- Значения с пробелами/`|`/путями — **только в обратных кавычках**.
- Default mode маппинга — **`body`**: случайный `http-headers=token|Authorization` без `|explicit` меняет wire и клиентские сигнатуры не так, как часто ожидают.
- `security` — app2app-формы без URL; `openId` отвергается; JWKS — относительный path-шаблон, не `https://…`.
- `--contracts` у генератора не уменьшает модель `astg` и не влияет на снимок `astg-hook`.
- Интерфейсы без `@tg`, неэкспортируемые и embedded — не контракты.
- Пакетный `desc` / `title` / `version` питают OpenAPI `info.*`; method `summary` — обязателен для читаемой публичной поверхности.
- Kafka и HTTP-семейство не смешивать на одном интерфейсе.

## Ограничения

1. Интерфейсы без `@tg` не учитываются.
2. Методы без своих аннотаций наследуют настройки от интерфейса и пакета (cascade), но в JSON-модели ключи физически не дублируются на каждый метод.
3. Учитываются только **экспортируемые** интерфейсы.
4. Встроенные (embedded) интерфейсы в контрактах не обрабатываются.
5. Контракты ищутся только в `contracts-dir`; только `.go` в самой папке, без поддиректорий.
6. Файлы с заголовком генерации tgp не участвуют в поиске сервисов/реализаций и не входят в набор для кэша.

## Связь с другими плагинами

| Плагин | Роль относительно `astg` |
| ------ | ------------------------ |
| `astg-db` | Pre: подставить модель из локальной БД вместо/до разбора |
| `astg-hook` | Post: сохранить полный снимок после локального разбора |
| `astg-json` | Экспорт модели в JSON (`tg astg json`) |
| server / client-* / swagger / kafka-* | Генерация по модели |

Документация: `tg plugin doc <имя>`.
