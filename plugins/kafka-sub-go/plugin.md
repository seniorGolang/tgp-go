# kafka-sub-go

Генерирует Go-подписчик Kafka из контрактов с аннотацией `@tg kafka`.

```sh
tg kafka sub go -o internal/subscriber/kafka
```

`out` обязателен и должен находиться внутри Go-модуля. Фильтр `--contracts`
ограничивает список Kafka-контрактов.

## Обработчики

Для каждого контракта необходимо зарегистрировать ровно одну форму:

- `Contract` — одно декодированное событие;
- `ContractMeta` — событие и `Meta` с key, headers, topic, partition, offset, timestamp;
- `ContractSlice` — пакет декодированных событий;
- `ContractBatch` — пакет записей со значением и metadata.

Для большинства обработчиков рекомендуется `Meta`: она сохраняет контекст записи
без пакетной модели.

## Запуск

```go
subscriber, err := kafka.New(log,
	kafka.Brokers("127.0.0.1:9092"),
	kafka.Group("orders-worker"),
	kafka.OrderEventsMeta(handler),
)
if err != nil {
	return err
}
defer subscriber.Close()

return subscriber.Run(ctx)
```

`Brokers` и `Group` обязательны. Одновременный повторный `Run` запрещён.

## Offset и commit

- `ResetOffset(kafka.AtStart|kafka.AtEnd)` задаёт позицию без committed offset;
- по умолчанию offset commit выполняется после успешной обработки batch;
- `CommitAuto()` переключает на auto-commit franz-go;
- `CommitAfterBatch()` и `CommitAuto()` нельзя задавать вместе.

Ошибка decode, handler или commit завершает `Run`. Отмена контекста останавливает
цикл чтения.

## Настройки

Поддерживаются `MaxPollRecords`, `FetchMinBytes`, `FetchMaxWait`, пользовательские
кодеки, TLS, SASL (`PLAIN`, `SCRAM-SHA-256`, `SCRAM-SHA-512`), а также
Prometheus-метрики, lag и OpenTelemetry trace при соответствующих аннотациях.
`Auth` и `SASL` задаются совместно.
