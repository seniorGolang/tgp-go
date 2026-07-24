# Kafka publisher для Go

Команда генерирует типобезопасный издатель Kafka из контрактов с аннотацией
`@tg kafka`.

```bash
tg kafka pub go -o internal/publisher/kafka
```

Обязательный параметр `out` задаёт каталог и имя генерируемого пакета. Можно
ограничить запуск: `-contracts OrderEvents,AuditEvents`. По умолчанию контракты
ищутся в `contracts`.

```go
// @tg kafka
// @tg kafka-acks=allISRAcks
// @tg log metrics trace
type OrderEvents interface {
	// @tg kafka-topic=orders.created
	// @tg kafka-key=orderID
	// @tg kafka-message=event
	OrderCreated(ctx context.Context, orderID string, event OrderCreated) (err error)
}
```

```go
publisher, err := kafka.New(log,
	kafka.Brokers("127.0.0.1:9092"),
	kafka.TLS(&tls.Config{MinVersion: tls.VersionTLS12}),
	kafka.Auth(user, password),
	kafka.SASL("SCRAM-SHA-256"),
	kafka.Metrics(reg),
	kafka.Trace(provider),
)
if err != nil {
	return err
}
defer publisher.Close()

return publisher.OrderEvents().OrderCreated(ctx, orderID, event)
```

Поддерживаются встроенные кодеки `json`, `bytes`, `msgpack`, `cbor`, `yaml` и
`xml`; пользовательский кодек передаётся через `kafka.Codec`. Для методов с
`[]T` или `...T` кодируются все сообщения до первой отправки, а пустой пакет
не отправляется.
