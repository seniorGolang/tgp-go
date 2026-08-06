# Плагин kafka-sub-go — подписчик Kafka (franz-go)

## Назначение

Плагин генерирует типобезопасный **subscriber** из контрактов `@tg kafka`: poll → decode → ваш handler → commit-политика.

Полный синтаксис аннотаций: `tg plugin doc astg`.

## Когда использовать

- Нужен consumer group worker на Go по тем же kafka-контрактам, что и publisher.
- Контракт только Kafka (без HTTP/stream семейства).
- `--out` внутри Go-модуля; не в одном пакете с publisher.

## Команда и параметры

```bash
tg kafka sub go --out internal/subscriber/kafka
tg kafka sub go --out internal/subscriber/kafka --contracts OrderEvents
```

| Параметр | Короткая форма | Обязательный | Описание |
|----------|----------------|--------------|----------|
| `out` | **нет** (только `--out`) | да | Каталог пакета |
| `contracts-dir` | — | нет | Каталог контрактов (default `contracts`) |
| `contracts` | — | нет | Фильтр (`--contracts`, не `-contracts`) |

Нужен `go.mod` для каталога вывода.

## Что генерируется

### Матрица

| На контракте / выбор | Появится |
|----------------------|----------|
| `@tg kafka` | `New`, options, `Run` / `Close`, четыре формы handler на контракт |
| ровно одна форма handler | валидный runtime |
| `kafka-codec` | decode через встроенный/зарегистрированный codec |
| `metrics` / `trace` | Metrics, LagInterval, Trace |
| HTTP/stream | ничего |

### Четыре формы handler (ровно одна на контракт)

| Option | Вход | Когда |
|--------|------|-------|
| `{Contract}` | одно событие `T` | metadata не нужна |
| `{Contract}Meta` | `T` + `Meta` | **рекомендуемый default** (key, headers, topic, partition, offset, timestamp) |
| `{Contract}Slice` | `[]T` | пакет значений без per-record meta |
| `{Contract}Batch` | `Batch[T]` | пакет записей value+Meta |

Ноль, `nil` или две формы сразу — ошибка конфигурации.

## Аннотации и их влияние

Те же контрактные аннотации, что у publisher:

| Аннотация | Уровень | Эффект на subscriber |
|-----------|---------|----------------------|
| `kafka` | iface | генерация consumer adapter |
| `kafka-topic` | method | dispatch топик → метод |
| `kafka-message` | method | тип значения `T` |
| `kafka-key` / `kafka-headers` | method | доступны в `Meta` / ожиданиях модели |
| `kafka-codec` | iface / method | decode (default `json`) |
| `kafka-acks` | iface / method | на subscriber wire не «пишет ack», но контракт общий с pub |
| `metrics` / `trace` / `log` | пакет / iface | observation |

Не смешивайте с `http-server` / `jsonRPC-server` / `ws-server` / `sse-server` / `stream`.

## Использование

```go
subscriber, err := kafka.New(log,
    kafka.Brokers("127.0.0.1:9092"), // обязательно
    kafka.Group("orders-worker"),    // обязательно
    kafka.OrderEventsMeta(handler),  // ровно одна форма
    kafka.ResetOffset(kafka.AtStart), // или AtEnd; только без committed offset
    kafka.CommitAfterBatch(),        // default, at-least-once
    // kafka.CommitAuto(),           // at-most-once; не для обычных handlers
    kafka.ShutdownTimeout(30*time.Second),
    kafka.MaxPollRecords(...),
    kafka.FetchMinBytes(...),
    kafka.FetchMaxWait(...),
    kafka.TLS(...),
    kafka.Auth(user, password),
    kafka.SASL("SCRAM-SHA-256"),
    kafka.Codec(...),
    kafka.LagInterval(...),
    kafka.ClientOpt(...),
)
if err != nil {
    return err
}
defer subscriber.Close()

return subscriber.Run(ctx)
```

### Commit и lifecycle

| Тема | Поведение |
|------|-----------|
| `CommitAfterBatch` | commit после успешного handler (default, at-least-once) |
| `CommitAuto` | auto-commit franz-go во время handler (at-most-once); offset может уйти до успеха |
| Вместе | запрещено |
| Ошибка decode / handler / commit | `Run` завершается |
| Отмена ctx | остановка poll |
| Повторный `Run` | запрещён (один активный цикл) |
| `ShutdownTimeout` | сколько `Close` ждёт `Run` (default **30s**), затем принудительное закрытие клиента |
| `ResetOffset` | позиция при отсутствии committed offset (default AtStart) |

`Auth` и `SASL` задаются **совместно**; `TLS` — отдельно.

## На что обратить внимание

1. Только **`--out`**; фильтр — **`--contracts`**.
2. `Brokers` + `Group` обязательны.
3. Выберите **одну** форму handler; для большинства случаев — `Meta`.
4. Не глотайте ошибки handler’а, рассчитывая на commit — при default политике failed batch не закоммитится успешно.
5. `CommitAuto` не маскирует сбои обработки.
6. Publisher и subscriber — разные `--out`.
7. Идемпотентность/DLQ — в приложении, не в сгенерированном коде.

## Ограничения

1. Только чистые `@tg kafka` контракты.
2. Ровно один handler form на контракт.
3. Кастомный codec — зарегистрировать до `New`.
4. Выход только в модуле с `go.mod`.

## Связь с другими плагинами

| Плагин | Связь |
|--------|-------|
| `kafka-pub-go` | те же topic/message/codec аннотации |
| `astg-json` | проверка resolved Kafka model |
| `server` / clients / `swagger` | Kafka не включают |
| `tgp-contracts` | авторство контрактов |
