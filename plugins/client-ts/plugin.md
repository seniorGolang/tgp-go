# Плагин client-ts

## Назначение

Плагин генерирует **TypeScript-клиент** для вызова вашего API. Клиент поддерживает **JSON-RPC 2.0** и **HTTP REST** — в зависимости от того, как помечены контракты в проекте. По контрактам (интерфейсам с аннотациями `@tg`) строятся типобезопасные методы, типы и обработка ошибок.

Подробнее про контракты и аннотации — в документации плагина astg: `tg plugin doc astg`.

## Возможности

- **JSON-RPC 2.0** — вызовы методов и batch-запросы.
- **HTTP REST** — GET, POST, PUT, PATCH, DELETE с настраиваемыми путями, заголовками и cookie.
- **Типобезопасность** — методы и типы генерируются по сигнатурам интерфейсов.
- **Асинхронность** — все методы возвращают Promise.
- **Заголовки** — статические или динамические (функция, в том числе async).
- **Blob и FormData** — загрузка и скачивание файлов (один Blob — тело запроса/ответа; несколько или multipart — FormData).
- **NPM** — сгенерированный код готов к публикации как пакет (package.json создаётся вручную).

## Как запускать

Команда генерации: `tg client ts` (или через общий пайплайн с плагином astg).

**Обязательный параметр:**

- **`out`** — каталог, куда положить сгенерированный код (например, `./client-ts`).

**Необязательные параметры:**

- **`contracts-dir`** — каталог с контрактами относительно корня проекта (по умолчанию `contracts`). Используется при разборе проекта (плагин astg).
- **`contracts`** — список имён контрактов через запятую; клиент генерируется только по ним (например: `UserService,OrderService`). Если не указан — берутся все контракты.
- **`doc-file`** — путь к файлу с документацией по клиенту. По умолчанию при включённой документации: `<out>/readme.md`.
- **`no-doc`** — не генерировать документацию (по умолчанию документация создаётся).

Перед каждой генерацией старые сгенерированные файлы в `out` удаляются; затем создаются новые.

## Что появляется в каталоге `out`

В указанном каталоге создаются следующие файлы:

- **client.ts** — функция `newClient()`, класс `Client`, методы вида `userService()` и `userServiceHTTP()` для доступа к клиентам контрактов, метод `batch()` для JSON-RPC batch;
- **options.ts** — тип `ClientOptions` (url, headers, idGeneratorFn);
- **version.ts** — константа `VersionASTg` с версией проекта;
- **error.ts** — типы для разбора ошибок JSON-RPC и функция `defaultErrorDecoder`;
- **batch.ts** — типы `BatchRequest` и `RpcCallback` для batch-запросов (только при наличии JSON-RPC-контрактов);
- **jsonrpc/** — реализация JSON-RPC 2.0 клиента;
- **\<имя-контракта>.ts** — JSON-RPC клиент сервиса (например, `user-service.ts`);
- **\<имя-контракта>-http.ts** — HTTP клиент сервиса;
- **\<имя-контракта>-exchange.ts** — TypeScript-типы запросов и ответов по контракту;
- **tsconfig.json** — рекомендуемая конфигурация TypeScript;
- **readme.md** — документация по клиенту (если не отключена через `no-doc`).

## Использование сгенерированного клиента

### Установка

После публикации пакета в NPM:

```bash
npm install @your-org/your-api-client
# или
yarn add @your-org/your-api-client
```

### Базовый пример

```typescript
import { newClient } from '@your-org/your-api-client';

const client = newClient('https://api.example.com');
const userService = client.userService();

const user = await userService.getUser('user-id-123');
console.log('User:', user);
```

Один клиент даёт доступ ко всем контрактам: для JSON-RPC — методы вида `userService()`, для HTTP — `userServiceHTTP()`. Имена методов образуются от имени контракта в kebab-case (например, контракт `UserService` → `userService()` и `userServiceHTTP()`).

### Настройка клиента

```typescript
const client = newClient('https://api.example.com', {
    // Статические заголовки
    headers: {
        'Authorization': 'Bearer token',
        'X-Custom-Header': 'value',
    },

    // Динамические заголовки (функция, можно async)
    headers: async () => {
        const token = await getAuthToken();
        return { 'Authorization': `Bearer ${token}` };
    },

    // URL для запросов (если нужно переопределить)
    url: 'https://api.example.com',

    // Генератор ID для JSON-RPC (по умолчанию crypto.randomUUID())
    idGeneratorFn: () => crypto.randomUUID(),
});
```

### JSON-RPC вызовы

- Имя метода в JSON-RPC: `{контракт}.{метод}` в camelCase (например, `userService.getUser`).
- Параметры метода (кроме контекста) передаются в запросе; результат приходит в ответе. Ошибки выбрасываются как исключения.

```typescript
const client = newClient('https://api.example.com');
const userService = client.userService();

const user = await userService.getUser('123');
```

### JSON-RPC batch

Несколько запросов можно выполнить одним вызовом через `client.batch()`. Для каждого метода контракта есть функция вида `req{MethodName}` (например, `reqGetUser`), которая возвращает запрос для batch. Обработчик результата задаётся в поле `retHandler`.

```typescript
const client = newClient('https://api.example.com');
const userService = client.userService();

await client.batch([
    {
        rpcRequest: userService.reqGetUser('1'),
        retHandler: (error, response) => {
            if (error) {
                console.error('Error:', error);
                return;
            }
            console.log('User 1:', response?.result?.user);
        },
    },
    {
        rpcRequest: userService.reqGetUser('2'),
        retHandler: (error, response) => {
            if (error) {
                console.error('Error:', error);
                return;
            }
            console.log('User 2:', response?.result?.user);
        },
    },
]);
```

### HTTP-методы

Для методов, помеченных в контракте как HTTP (`@tg http-method`, `@tg http-path` и т.д.), клиент:

- отправляет указанный HTTP-метод и путь (параметры пути подставляются по `http-args`);
- передаёт query-параметры, тело запроса для POST/PUT/PATCH;
- поддерживает заголовки и cookie через аннотации;
- при одном аргументе типа Blob отправляет тело как Blob (или application/octet-stream); при нескольких Blob или `http-multipart` — как FormData. Ответ с одним Blob возвращается как Blob, с несколькими или multipart — как части FormData по именам.

#### Режимы маппинга `http-headers` / `http-cookies` / `http-args`

Аннотации маппинга поддерживают формат `arg|key` или `arg|key|mode`, где `mode` — один из `explicit`, `implicit`, `body`. Если режим не указан, используется `body`.

- **`explicit`**: аргумент присутствует в сигнатуре HTTP‑методов (`userServiceHTTP()` и т.п.) и отправляется в указанный заголовок/cookie/query‑параметр. Такой параметр также отражается в сгенерированной OpenAPI‑спецификации.
- **`implicit`**: параметр описан в контракте и спецификации, но может не попадать в явную сигнатуру методов клиента; значение типично берётся из общих настроек (headers в `ClientOptions`) или внешнего кода.
- **`body`**: значение остаётся частью тела запроса/ответа; маппинг может использоваться сервером как дополнительный источник/override, но TypeScript‑клиент не делает из него отдельный HTTP‑параметр.

```typescript
const client = newClient('https://api.example.com');
const userServiceHTTP = client.userServiceHTTP();

// GET
const user = await userServiceHTTP.getUser('123');

// POST с телом
const id = await userServiceHTTP.createUser({
    name: 'John',
    email: 'john@example.com',
});

// Загрузка файла (один Blob — тело запроса)
const blob = new Blob([binaryData]);
const id = await fileServiceHTTP.upload({ filename: 'file.txt', body: blob });

// Загрузка нескольких файлов (multipart/form-data)
await fileServiceHTTP.uploadMultipart({ file1: blob1, file2: blob2 });

// Скачивание (ответ как Blob)
const blob = await fileServiceHTTP.download(id);

// Скачивание multipart (ответ как части FormData по именам)
const { part1, part2 } = await fileServiceHTTP.downloadMultipart(id);
```

### Inline-ответ для одного значения

Если у метода одно возвращаемое значение (кроме error) и в контракте указана аннотация `@tg enableInlineSingle`, результат приходит без обёртки в объект — одним значением.

```typescript
// Без enableInlineSingle: result = { status: "ok" }
// С enableInlineSingle: result = "ok"
const status = await service.getStatus();
```

### Обработка ошибок

Ошибки вызовов выбрасываются как исключения. Их можно ловить через try/catch.

```typescript
try {
    const user = await userService.getUser('123');
    console.log('User:', user);
} catch (error) {
    if (error instanceof Error) {
        console.error('Error:', error.message);
    } else {
        console.error('Unknown error:', error);
    }
}
```

### Пример с React

```typescript
import { useState, useEffect } from 'react';
import { newClient } from '@your-org/your-api-client';

function UserComponent({ userId }: { userId: string }) {
    const [user, setUser] = useState<User | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    useEffect(() => {
        const client = newClient('https://api.example.com');
        const userService = client.userService();

        userService.getUser(userId)
            .then(setUser)
            .catch(setError)
            .finally(() => setLoading(false));
    }, [userId]);

    if (loading) return <div>Loading...</div>;
    if (error) return <div>Error: {error.message}</div>;
    if (!user) return <div>User not found</div>;

    return <div>{user.name}</div>;
}
```

## Документация по клиенту

По умолчанию плагин генерирует в каталоге `out` файл `readme.md` с описанием контрактов, методов и типов. Документацию можно отключить опцией `--no-doc` или указать другой файл через `--doc-file`.

## Публикация как NPM-пакет

Плагин генерирует только исходный TypeScript и `tsconfig.json`; `package.json` не создаётся. Чтобы опубликовать клиент в NPM:

1. Создайте в каталоге `out` файл `package.json` вручную (при необходимости можно опираться на аннотации в контрактах, например `@tg npmName=@your-org/your-api-client`).
2. Перейдите в каталог сгенерированного клиента и выполните публикацию:

```bash
cd generated-client
npm publish
```

## Зависимости

Сгенерированный код не требует внешних зависимостей для базовой работы. Используются только стандартные API:

- `fetch` — для HTTP-запросов;
- `crypto.randomUUID()` — для генерации ID (при необходимости можно подключить полифилл);
- Promise — для асинхронности.

## Ограничения

- В контрактах методы должны принимать `context.Context` первым аргументом (в TypeScript контекст не передаётся).
- Методы должны возвращать `error` последним значением (в TypeScript ошибки приходят как исключения).
- Параметры и возвращаемые значения (кроме error) должны быть именованными.
- Поддерживаются только публичные интерфейсы (с заглавной буквы).
- Для HTTP-методов в контракте должны быть указаны аннотации `http-method` и `http-path`.
- Рекомендуется TypeScript 4.0 или выше.

## Совместимость

Клиент совместим с контрактами, поддерживаемыми плагином astg, и с серверами, сгенерированными плагином server. Все аннотации astg учитываются при генерации клиента.
