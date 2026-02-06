# Client-TS Plugin - Генератор TypeScript клиента

## Описание

Плагин `client-ts` генерирует TypeScript клиент для взаимодействия с серверами, реализующими контракты с аннотациями `@tg`. Клиент поддерживает оба протокола: JSON-RPC 2.0 и HTTP REST API.

Для получения общей информации о парсере и поддерживаемых аннотациях используйте команду `tg plugin doc astg`.

## Основные возможности

- **JSON-RPC 2.0 клиент** - полная поддержка JSON-RPC протокола с batch запросами
- **HTTP REST клиент** - поддержка HTTP методов (GET, POST, PUT, PATCH, DELETE) с настраиваемыми путями
- **Типобезопасность** - автоматическая генерация типизированных методов на основе Go интерфейсов
- **TypeScript типы** - полная поддержка TypeScript типов и интерфейсов
- **Асинхронность** - все методы возвращают Promise
- **Batch запросы** - поддержка выполнения нескольких запросов в одном batch
- **Кастомизация** - гибкая настройка через опции клиента
- **NPM пакет** - готовый к публикации NPM пакет с поддержкой TypeScript

## Использование

### Установка

```bash
npm install @your-org/your-api-client
# или
yarn add @your-org/your-api-client
```

### Базовый пример

```typescript
import {newClient} from '@your-org/your-api-client';

// Создаем клиент
const client = newClient('https://api.example.com');

// Получаем клиент сервиса
const userService = client.userService();

// Вызываем метод
const user = await userService.getUser('user-id-123');
console.log('User:', user);
```

### Настройка клиента

```typescript
const client = newClient('https://api.example.com', {
    // Статические заголовки
    headers: {
        'Authorization': 'Bearer token',
        'X-Custom-Header': 'value',
    },

    // Динамические заголовки (функция)
    headers: async () => {
        const token = await getAuthToken();
        return {
            'Authorization': `Bearer ${token}`,
        };
    },

    // Настройка URL для генерации ID запросов
    url: 'https://api.example.com',

    // Кастомная функция генерации ID для JSON-RPC
    idGeneratorFn: () => crypto.randomUUID(),

    // Таймаут запросов (в миллисекундах)
    timeout: 30000,
});
```

## Структура генерируемого кода

### Базовый клиент (`client.ts`)

Содержит:

- Класс `Client` с настройками
- Функцию `newClient()` для создания клиента
- Методы для получения клиентов сервисов (например, `userService()`)
- Метод `batch()` для выполнения batch запросов

### Клиенты сервисов (`{service}-client.ts` и `{service}-client-http.ts`)

Для каждого контракта генерируются отдельные файлы:

1. **JSON-RPC клиент** (`{service}-client.ts`) - для контрактов с `@tg jsonRPC-server`
2. **HTTP клиент** (`{service}-client-http.ts`) - для контрактов с `@tg http-server`

Каждый файл содержит:

- Класс `{ServiceName}Client` или `{ServiceName}HTTPClient`
- Типизированные методы для каждого метода контракта
- Полную поддержку TypeScript типов

### JSON-RPC методы

Для методов, помеченных как JSON-RPC, генерируются методы, которые:

- Принимают параметры метода (кроме контекста)
- Возвращают Promise с результатом
- Поддерживают batch запросы через метод `batch()` базового клиента

```typescript
// Синхронный вызов
const user = await userService.getUser('user-id');

// Batch запрос
await client.batch([
    {
        rpcRequest: userService.reqGetUser('user-id-1'),
        retHandler: (result, error) => {
            if (error) {
                console.error('Error:', error);
                return;
            }
            console.log('User 1:', result?.user);
        },
    },
    {
        rpcRequest: userService.reqGetUser('user-id-2'),
    },
]);
```

### HTTP методы

Для методов, помеченных как HTTP, генерируются методы, которые:

- Используют указанный HTTP метод (GET, POST, PUT, PATCH, DELETE)
- Поддерживают параметры пути через `http-path` и `http-args`
- Поддерживают query параметры
- Поддерживают тело запроса для POST/PUT/PATCH
- Поддерживают заголовки через `http-headers`
- Поддерживают cookies через `http-cookies`
- Поддерживают кастомные типы контента
- **Blob и FormData**: аргументы `io.Reader` маппятся на тип `Blob`, возвращаемые `io.ReadCloser` — на `Blob`. При одном таком аргументе тело запроса отправляется как `Blob` (Content-Type из `requestContentType` или `application/octet-stream`). При нескольких `Blob` или аннотации `http-multipart` запрос формируется как `FormData` (multipart/form-data); имена и Content-Type частей задаются аннотациями `http-part-name`, `http-part-content`. Ответ с одним `Blob` разбирается через `response.blob()`, с несколькими или `http-multipart` — через `response.formData()` по именам частей

```typescript
// GET запрос
const user = await userServiceHTTP.getUser('user-id');
// Выполнит GET /users/user-id

// POST запрос
const id = await userServiceHTTP.createUser({
    name: 'John',
    email: 'john@example.com',
});
// Выполнит POST /users с телом запроса

// Upload (один Blob — тело запроса как application/octet-stream)
const blob = new Blob([binaryData]);
const id = await fileServiceHTTP.upload({ filename: 'file.txt', body: blob });

// Upload (multipart/form-data — FormData)
const formData = new FormData();
formData.append('file1', blob1);
formData.append('file2', blob2);
await fileServiceHTTP.uploadMultipart({ file1: blob1, file2: blob2 });

// Download (ответ как Blob)
const blob = await fileServiceHTTP.download(id);

// Download (multipart — ответ как части FormData по именам)
const { part1, part2 } = await fileServiceHTTP.downloadMultipart(id);
```

### Типы данных (`types.ts`)

Генерируются все TypeScript типы, используемые в контрактах:

- Интерфейсы для структур
- Типы для массивов, объектов, union типы
- Опциональные поля
- Вложенные типы

```typescript
export interface User {
    id: string;
    name: string;
    email: string;
    role?: string;
}

export interface CreateUserRequest {
    name: string;
    email: string;
    role?: string;
}
```

### Опции клиента (`options.ts`)

Определяет интерфейс `ClientOptions` для настройки клиента:

```typescript
export interface ClientOptions {
    url: string;
    headers?: Record<string, string> | (() => Promise<Record<string, string>>);
    idGeneratorFn?: () => string;
    timeout?: number;
}
```

### JSON-RPC библиотека (`jsonrpc/`)

Содержит реализацию JSON-RPC 2.0 клиента:

- `client.ts` - основной JSON-RPC клиент
- `utils/jsonrpc.ts` - типы и утилиты для JSON-RPC
- `batch.ts` - поддержка batch запросов

### Конфигурация TypeScript (`tsconfig.json`)

Генерируется `tsconfig.json` с рекомендуемыми настройками для проекта.

## Особенности генерации

### JSON-RPC методы

1. **Именование методов**: Имя метода в JSON-RPC формируется как `{contractName}.{methodName}` в camelCase
2. **Параметры**: Все параметры передаются в поле `params` JSON-RPC запроса
3. **Результаты**: Все возвращаемые значения упаковываются в поле `result` JSON-RPC ответа
4. **Batch запросы**: Поддерживается выполнение нескольких запросов в одном batch через метод `batch()` базового клиента. Для каждого метода контракта генерируется метод создания запроса вида `req{MethodName}` (например, для `GetUser` — `reqGetUser`). В `retHandler` первым аргументом приходит результат (или `null` при ошибке), вторым — ошибка (или `null` при успехе): `(result, error) => void`.
5. **Асинхронность**: Все методы возвращают Promise

### HTTP методы

1. **Пути**: Используется аннотация `http-path` для определения пути. Если не указана, используется путь по умолчанию
2. **Параметры пути**: Параметры пути (например, `:id`) маппятся на аргументы метода через аннотацию `http-args`
3. **Query параметры**: Параметры, не являющиеся частью пути, передаются как query параметры
4. **Тело запроса**: Для POST/PUT/PATCH методов тело запроса формируется из всех параметров, кроме параметров пути
5. **Заголовки и cookies**: Маппятся через аннотации `http-headers` и `http-cookies`
6. **Типы контента**: Поддерживаются кастомные типы через `requestContentType` и `responseContentType`

### Inline ответы

Для методов с единственным возвращаемым значением и аннотацией `enableInlineSingle` результат возвращается напрямую, без обертки в объект:

```typescript
// Без enableInlineSingle
const result = await service.getStatus();
// result = {status: "ok"}

// С enableInlineSingle
const status = await service.getStatus();
// status = "ok" (возвращается напрямую)
```

### Динамические заголовки

Клиент поддерживает как статические, так и динамические заголовки:

```typescript
// Статические заголовки
const client = newClient('https://api.example.com', {
    headers: {
        'Authorization': 'Bearer token',
    },
});

// Динамические заголовки (async функция)
const client = newClient('https://api.example.com', {
    headers: async () => {
        const token = await getAuthToken();
        return {
            'Authorization': `Bearer ${token}`,
        };
    },
});
```

## Опции командной строки

- `out` (string, обязательная) - путь к выходной директории
- `contracts-dir` (string, опциональная) - путь к директории с контрактами (по умолчанию: "contracts")
- `contracts` (string, опциональная) - список контрактов через запятую для фильтрации (например: "UserService,OrderService")
- `doc-file` (string, опциональная) - путь к файлу документации (по умолчанию: `<out>/readme.md`)
- `no-doc` (bool, опциональная) - отключить генерацию документации (по умолчанию: false)

## Примеры использования

### JSON-RPC вызов

```typescript
// Контракт
// @tg jsonRPC-server
// type UserService interface {
//     GetUser(ctx context.Context, id string) (user User, err error)
// }

// Использование
const client = newClient('https://api.example.com');
const userService = client.userService();
const user = await userService.getUser('123');
```

### HTTP GET запрос

```typescript
// Контракт
// @tg http-server
// type UserService interface {
//     // @tg http-method=GET
//     // @tg http-path=/users/:id
//     // @tg http-args=id|userId
//     GetUser(ctx context.Context, userId string) (user User, err error)
// }

// Использование
const client = newClient('https://api.example.com');
const userServiceHTTP = client.userServiceHTTP();
const user = await userServiceHTTP.getUser('123');
// Выполнит GET /users/123
```

### HTTP POST запрос с телом

```typescript
// Контракт
// @tg http-server
// type UserService interface {
//     // @tg http-method=POST
//     // @tg http-path=/users
//     CreateUser(ctx context.Context, req CreateUserRequest) (id string, err error)
// }

// Использование
const client = newClient('https://api.example.com');
const userServiceHTTP = client.userServiceHTTP();
const id = await userServiceHTTP.createUser({
    name: 'John',
    email: 'john@example.com',
});
// Выполнит POST /users с телом запроса
```

### Batch JSON-RPC запросы

```typescript
const client = newClient('https://api.example.com');
const userService = client.userService();

await client.batch([
    {
        rpcRequest: userService.reqGetUser('1'),
        retHandler: (result, error) => {
            if (error) {
                console.error('Error:', error);
                return;
            }
            console.log('User 1:', result?.user);
        },
    },
    {
        rpcRequest: userService.reqGetUser('2'),
        retHandler: (result, error) => {
            if (error) {
                console.error('Error:', error);
                return;
            }
            console.log('User 2:', result?.user);
        },
    },
]);
```

### Обработка ошибок

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

### Использование с React

```typescript
import {useState, useEffect} from 'react';
import {newClient} from '@your-org/your-api-client';

function UserComponent({userId}: { userId: string }) {
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

    if (loading) return <div>Loading
...
    </div>;
    if (error) return <div>Error
:
    {
        error.message
    }
    </div>;
    if (!user) return <div>User
    not
    found < /div>;

    return <div>{user.name} < /div>;
}
```

## NPM пакет

Плагин генерирует готовый к публикации NPM пакет. Для настройки пакета используйте аннотации на уровне пакета:

```go
// @tg npmName=@your-org/your-api-client
// @tg npmRegistry=https://registry.npmjs.org
// @tg npmPrivate=false
// @tg author=Your Name
// @tg license=MIT
// @tg version=1.0.0
```

После генерации клиента:

1. Перейдите в директорию сгенерированного клиента
2. Установите зависимости (если необходимо)
3. Опубликуйте пакет:

```bash
cd generated-client
npm publish
```

## Документация

По умолчанию плагин генерирует файл `readme.md` в выходной директории с полной документацией по всем методам и типам. Документация включает:

- Описание клиента
- Инструкции по установке
- Список всех контрактов и методов
- Примеры использования для каждого метода
- Описание всех TypeScript типов
- Примеры запросов и ответов
- Интеграция с популярными фреймворками (React, Vue, Angular)

Документацию можно отключить опцией `--no-doc` или указать другой файл через `--doc-file`.

## Зависимости

Генерируемый код не требует внешних зависимостей для базовой функциональности. Все используемые API являются стандартными для браузеров и Node.js:

- `fetch` API - для HTTP запросов
- `crypto.randomUUID()` - для генерации ID (или полифилл)
- Promise API - для асинхронности

Для старых окружений может потребоваться полифилл для `fetch` и `crypto.randomUUID()`.

## Ограничения

1. Все методы должны принимать `context.Context` первым аргументом (в TypeScript контекст не передается явно)
2. Все методы должны возвращать `error` последним значением (в TypeScript ошибки выбрасываются как исключения)
3. Все параметры и возвращаемые значения (кроме `error`) должны быть именованными
4. Поддерживаются только публичные интерфейсы (начинающиеся с заглавной буквы)
5. Для HTTP методов требуется явное указание `http-method` и `http-path` аннотаций
6. TypeScript версия должна быть 4.0 или выше

## Совместимость

Плагин полностью совместим с контрактами, сгенерированными плагином `server`. Все аннотации из плагина `astg` поддерживаются и учитываются при генерации клиента.

## TypeScript версии

Рекомендуется использовать TypeScript 4.0 или выше для полной поддержки всех возможностей генерируемого кода.
