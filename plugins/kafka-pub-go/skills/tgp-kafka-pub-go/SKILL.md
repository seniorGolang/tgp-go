---
name: tgp-kafka-pub-go
description: Генерация Go-издателя Kafka из @tg kafka контрактов.
---

# Kafka publisher Go

Используйте `tg kafka pub go -o internal/publisher/kafka` после изменения
контрактов. Контракт помечается `@tg kafka`; каждый метод требует
`@tg kafka-topic` и принимает первым аргументом `context.Context`, возвращая
только `error`.

Сообщение задаётся `@tg kafka-message`, а ключ и заголовки — `@tg kafka-key` и
`@tg kafka-headers`. Значение `kafka-acks` выбирается из `noAck`, `leaderAck`,
`allISRAcks`. Вызов `kafka.New` требует logger и `kafka.Brokers(...)`.
