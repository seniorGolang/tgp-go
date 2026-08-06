# Плагин client-ts — TypeScript SDK (browser / Node)

## Назначение

Плагин генерирует TypeScript-клиент по контрактам `@tg`: JSON-RPC 2.0, REST, WS/SSE, опционально npm-пакет. Методы асинхронные (`Promise`); ошибки — через throw.

Полный синтаксис аннотаций: `tg plugin doc astg`.

## Когда использовать

- Нужен SDK для браузера или Node по tgp-контрактам.
- Пути `--out` / `--package-json` считаются от **корня проекта** (каталог запуска `tg`).
- Не для Kafka и не для правки server `transport/`.

## Команда и параметры

```bash
tg client ts --out web
tg client ts --out web --package-json web/package.json --contracts=UserService
tg client ts --out web --no-doc --no-client-id
```

| Параметр | Короткая форма | Обязательный | Описание |
|----------|----------------|--------------|----------|
| `out` | **нет** (только `--out`) | да | Каталог TS-кода |
| `package-json` | — | нет | Путь к генерируемому `package.json` (нужен `@tg npmName`) |
| `contracts-dir` | — | нет | Каталог контрактов (default `contracts`) |
| `contracts` | — | нет | Фильтр имён (без `!Name`) |
| `doc-file` | — | нет | Документация (default `<out>/readme.md`) |
| `no-doc` | — | нет | Не писать readme |
| `no-client-id` | — | нет | Без `identity.ts`, без `clientName`, без `X-Client-Id` |

### `go generate` и cwd

`go generate` стартует из каталога пакета с директивой. Чтобы `tg` видел корень модуля:

```go
//go:generate sh -c "cd .. && tg client ts --out web --package-json web/package.json"
```

Либо положите директиву в `.go` в корне модуля.

## Что генерируется

### Матрица: контракт → API

| Need / аннотации | API |
|------------------|-----|
| `jsonRPC-server` | `client.userService()` — RPC + stream helpers |
| `http-server` | `client.userServiceHTTP()` — REST |
| оба | два accessor’а на один контракт |
| JSON-RPC batch | `client.batch(...)` + `reqGetUser` и т.п. |
| `ws-server` / `sse-server` | async stream на RPC-accessor; `signal?`, `onReady?` |
| form / multipart | `form.ts` (`parseFormValue` / `parseCSVFormValue`) |
| без `--no-client-id` | `identity.ts`, `X-Client-Id` |
| `--package-json` + npm-аннотации | `package.json`, `tsconfig` с `outDir: dist` |
| `kafka` | ничего |

Имена: контракт `UserService` → `userService()` / `userServiceHTTP()` (camelCase от имени).

### Инвентарь

| Файл | Назначение |
|------|------------|
| `client.ts` | `newClient`, `Client`, accessors, `batch` |
| `options.ts` | `ClientOptions` |
| `headers.ts` | сборка заголовков |
| `identity.ts` | default `X-Client-Id` (нет при `--no-client-id`) |
| `form.ts` | CSV/form parse (browser + Node) |
| `error.ts` | `ErrorDecoder` / `HTTPErrorDecoder` и default’ы |
| `batch.ts`, `jsonrpc/` | при JSON-RPC |
| `{service}.ts`, `{service}-http.ts`, `{service}-exchange.ts` | сервисы и типы |
| `tsconfig.json`, `version.ts`, `readme.md` | конфиг / версия / док |

## Аннотации и их влияние

Полный синтаксис: `tg plugin doc astg`.

| Аннотация | Уровень | Эффект на client-ts |
|-----------|---------|---------------------|
| `jsonRPC-server` | iface | RPC accessor `userService()` |
| `http-server` | iface | REST accessor `userServiceHTTP()` |
| `http-prefix` | пакет / iface | входит в заявленный route; стык с path endpoint сворачивается по совпадающим сегментам |
| `http-method` / `http-path` / `http-args` | method | REST |
| `http-headers` / `http-cookies` | method | headers/cookies; в браузерном WS — ещё query fallback |
| `ws-server` / `sse-server` / `stream` / paths | iface / method | stream API |
| `enableInlineSingle` | method | result без обёртки-объекта |
| `http-multipart` / part-* | method / arg | FormData вместо одного Blob |
| npm: `npmName`, `version`, `license`, `author`, `npmPrivate`, `npmRegistry` | пакет | метаданные при `--package-json` |
| typed enums / `enums` | type / field | TS-константы / union-ограничения в типах |
| `kafka` | iface | игнор |

### Mapping modes

| Mode | HTTP-сигнатура (`*HTTP`) | OpenAPI / wire |
|------|--------------------------|----------------|
| `explicit` | аргумент в сигнатуре | отдельный header/cookie/query |
| `implicit` | может отсутствовать | значение из `headers` в options |
| `body` | в теле | не отдельный HTTP-параметр |

### Endpoint

`newClient(endpoint)`: origin или URL с path. REST/SSE/WS склеивают path endpoint с заявленным route и сворачивают совпадающие сегменты. JSON-RPC — на сам `endpoint`.

## Использование

### Базовый пример

```typescript
import { newClient } from './client';

const client = newClient('https://api.example.com', {
  clientName: 'web-checkout',
  headers: async () => ({ Authorization: `Bearer ${await getToken()}` }),
});

const user = await client.userService().getUser('123');
const created = await client.userServiceHTTP().createUser({ name: 'John', email: 'a@b.c' });
```

### Опции клиента

| Поле | Назначение |
|------|------------|
| `url` | переопределить base URL |
| `headers` | объект или sync/async функция |
| `clientName` | `X-Client-Id` (если не `--no-client-id`) |
| `idGeneratorFn` | ID для JSON-RPC (default `crypto.randomUUID`) |
| `errorDecoder` / `httpErrorDecoder` | кастом разбора RPC / REST ошибок |

### Batch

```typescript
await client.batch([
  {
    rpcRequest: userService.reqGetUser('1'),
    retHandler: (error, response) => { /* … */ },
  },
]);
```

### Streams

Последние аргументы stream-методов: `signal?: AbortSignal`, `onReady?: () => void`.

- SSE: `signal` → `fetch` + `reader.cancel`; `onReady` после успешного response.
- WS: abort → `socket.close`; `onReady` после `open`.
- В браузере custom upgrade headers недоступны: сгенерированный клиент дублирует нужные значения в **query**; сервер читает query как fallback.

### Blob / FormData / CSV

| Контракт | TS |
|----------|----|
| один binary body/result | `Blob` |
| несколько / multipart | `FormData` |
| слайсы в header/query/form | CSV через `form.ts` |

### NPM

```go
// @tg npmName=@your-org/api-client
// @tg version=v1.0.0
// @tg license=MIT
```

```bash
tg client ts --out web --package-json web/package.json
cd web && npm install && npm run build && npm publish
```

Без `npmName` флаг `--package-json` завершится ошибкой. Для source-only клиента флаг не указывайте.

React/Vue и т.п. не входят в зону плагина: достаточно `newClient` + `AbortSignal` на размонтировании.

### Ошибки

Вызовы reject/throw. Различайте RPC и REST через кастомные decoder’ы в options (`errorDecoder`, `httpErrorDecoder`), если default `ErrorJsonRPC` / `ResponseError` недостаточно.

```typescript
try {
  await client.userService().getUser('123');
} catch (e) {
  // сузьте к сгенерированным/своим типам на границе приложения
}
```

### Inline single

`@tg enableInlineSingle` на методе с одним значимым result: вместо `{ status: "ok" }` приходит `"ok"`. Меняет типы exchange — перегенерируйте потребителей.

### Identity по умолчанию

Без `--no-client-id` и без явного `clientName`:

| Runtime | Default identity |
|---------|------------------|
| Node | hostname-based |
| Browser | agent token + hash User-Agent |

Итог уходит в `X-Client-Id` и должен совпадать с ожиданиями server metrics (`MaxClientIDs`, normalize).

### REST vs RPC — как не перепутать

```typescript
const rpc = client.userService();       // JSON-RPC + streams
const rest = client.userServiceHTTP();  // только REST

await rpc.getUser('1');                 // method userService.getUser
await rest.getUser('1');                // GET по http-path
```

Если в контракте только `http-server`, RPC-accessor может отсутствовать (и наоборот) — смотрите матрицу генерации и `astg-json`.

### Типичный цикл разработки

1. Правите контракты (`tgp-contracts`).
2. Из корня модуля: `tg client ts --out web …`.
3. `npx tsc -p web/tsconfig.json` (или `npm run build` пакета).
4. Smoke: один RPC, один REST, при наличии — stream cancel через `AbortSignal`.
5. При npm: проверить `package.json` exports после `--package-json`.

## На что обратить внимание

1. Только **`--out`**, не `-o`.
2. Неверный cwd (`go generate` из `contracts/`) → файлы не там.
3. Выбирайте accessor по транспорту: RPC → `userService()`, REST → `userServiceHTTP()`.
4. Browser WS: headers через query fallback — проверяйте серверный upgrade.
5. `--no-client-id` меняет метки на server metrics/logs.
6. `implicit` args «исчезают» из сигнатуры.
7. Не патчите сгенерированный TS — правьте контракты.
8. Kafka не генерируется.
9. Полифиллы `fetch` / `crypto.randomUUID` / `Blob` зависят от runtime.

## Ограничения

1. В Go-контракте `context.Context` и `error` обязательны; в TS контекст не передаётся, ошибки — исключения.
2. Именованные параметры/результаты; только публичные интерфейсы.
3. REST требует `http-method` + `http-path`.
4. Рекомендуется TypeScript ≥ 4.
5. Внешних runtime-зависимостей для базового клиента нет.

## Связь с другими плагинами

| Плагин | Связь |
|--------|-------|
| `astg` / `astg-json` | модель и диагностика |
| `server` | совместимый wire (в т.ч. query fallback на WS) |
| `client-go` | тот же контракт на Go |
| `swagger` | документация API |
| `tgp-contracts` | npm- и HTTP-аннотации |
