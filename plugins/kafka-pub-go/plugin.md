# Плагин kafka-pub-go — издатель Kafka (franz-go)

## Назначение

Плагин генерирует типобезопасный **publisher** из контрактов с `@tg kafka`: методы публикуют сообщения в топики с key/headers/codec/acks.

Полный синтаксис аннотаций: `tg plugin doc astg`.

## Когда использовать

- Нужна отправка доменных событий в Kafka из Go.
- Контракт **только** Kafka (без `http-server` / `jsonRPC-server` / `ws-server` / `sse-server` / `stream`).
- `--out` внутри Go-модуля; не смешивайте пакет с subscriber.

## Команда и параметры

```bash
tg kafka pub go --out internal/publisher/kafka
tg kafka pub go --out internal/publisher/kafka --contracts OrderEvents,AuditEvents
```

| Параметр | Короткая форма | Обязательный | Описание |
|----------|----------------|--------------|----------|
| `out` | **нет** (только `--out`) | да | Каталог пакета (basename = имя пакета) |
| `contracts-dir` | — | нет | Каталог контрактов (default `contracts`) |
| `contracts` | — | нет | Фильтр имён (`--contracts`, не `-contracts`) |

Без `go.mod` для каталога `out` генерация падает: `go.mod not found`.

## Что генерируется

### Матрица

| На контракте | Появится |
|--------------|----------|
| `@tg kafka` + методы с `kafka-topic` | `New`, options, adapter `OrderEvents().OrderCreated(...)` |
| `kafka-codec` | встроенный кодек / требование `Codec(...)` |
| `kafka-acks` | клиент(ы) с выбранной политикой ack |
| message `[]T` / `...T` | batch publish |
| `log` / `metrics` / `trace` | соответствующие runtime-опции (если сгенерированы) |
| HTTP/stream семейство | ничего (контракт должен быть чистым kafka) |

Типичный surface: конструктор `kafka.New`, опции (`Brokers`, `TLS`, `Auth`+`SASL`, …), методы контрактов, `Close`.

## Аннотации и их влияние

| Аннотация | Уровень | Эффект на publisher |
|-----------|---------|---------------------|
| `kafka` | iface | включает контракт в генерацию pub |
| `kafka-topic` | method | топик (уникален в проекте) |
| `kafka-message` | method | какой аргумент — тело (иначе должен быть однозначен) |
| `kafka-key` | method | аргумент ключа (string/bytes-совместимый) |
| `kafka-headers` | method | аргументы заголовков |
| `kafka-codec` | iface / method | `json` (default), `bytes`, `msgpack`, `cbor`, `yaml`, `xml` |
| `kafka-acks` | iface / method | `noAck` / `leaderAck` / `allISRAcks` (default cascade → allISRAcks) |
| `log` / `metrics` / `trace` | пакет / iface | observation hooks при генерации |

Правила метода: первый аргумент `context.Context`, единственный результат `error`, message не pointer. Key/header/message — разные аргументы; лишние unused args не публикуются.

**Пример контракта:**

```go
// @tg kafka
// @tg kafka-acks=allISRAcks
// @tg metrics
type OrderEvents interface {
	// @tg kafka-topic=orders.created
	// @tg kafka-key=orderID
	// @tg kafka-message=event
	// @tg kafka-codec=json
	OrderCreated(ctx context.Context, orderID string, event OrderCreated) (err error)
}
```

**Batch:** message `[]T` или `...T` — все записи кодируются до первой отправки; пустой batch — no-op; `[]byte` — одно байтовое сообщение, не batch.

Cascade: codec и acks резолвятся method → iface → default. Runtime поднимает только те producer-клиенты, которые нужны выбранным ack-политикам.

## Использование

```go
publisher, err := kafka.New(log,
    kafka.Brokers("127.0.0.1:9092"), // обязательно
    kafka.TLS(&tls.Config{MinVersion: tls.VersionTLS12}),
    kafka.Auth(user, password),
    kafka.SASL("SCRAM-SHA-256"), // вместе с Auth
    kafka.Compression(...),
    kafka.BatchMaxLinger(...),
    kafka.BatchMaxBytes(...),
    kafka.MaxBufferedRecords(...),
    kafka.Codec("custom", myCodec),
    kafka.ShutdownTimeout(30*time.Second), // default 30s
    kafka.ClientOpt(...), // escape hatch franz-go, кроме уже явного security
)
if err != nil {
    return err
}
defer publisher.Close()

return publisher.OrderEvents().OrderCreated(ctx, orderID, event)
```

| Option | Правило |
|--------|---------|
| `Brokers` | обязателен |
| `Auth` + `SASL` | только **вместе** (`PLAIN`, `SCRAM-SHA-256`, `SCRAM-SHA-512`) |
| `TLS` | независимо |
| `ShutdownTimeout` | лимит ожидания незавершённых publish при `Close` (default **30s**) |
| `Compression` / `BatchMax*` / `MaxBufferedRecords` | тюнинг продюсера |
| `ClientOpt` | доп. опции franz-go без дублирования security |
| `Metrics` / `Trace` | при соответствующих аннотациях |

## На что обратить внимание

1. Только **`--out`**, не `-o`; фильтр — **`--contracts`**.
2. `Brokers` обязателен; Auth без SASL (и наоборот) — ошибка конфигурации.
3. Топики глобально уникальны.
4. Не смешивайте Kafka и HTTP-семейство на одном iface.
5. Publisher и subscriber — **разные** пакеты `--out`.
6. Пустой batch молча не шлёт — это норма.
7. После смены codec/acks перегенерируйте и проверьте wire.

## Ограничения

1. Только `@tg kafka` (legacy `kafka-consumer` / `kafka-publisher` отвергаются).
2. Нет HTTP/RPC методов в этом пакете.
3. Кастомный codec нужно зарегистрировать до `New`.
4. Выход только внутри модуля с `go.mod`.

## Связь с другими плагинами

| Плагин | Связь |
|--------|-------|
| `kafka-sub-go` | тот же контракт на чтении |
| `astg` / `astg-json` | проверка topic/message/codec |
| `server` / clients / `swagger` | Kafka не трогают |
| `tgp-contracts` | авторство kafka-аннотаций |
