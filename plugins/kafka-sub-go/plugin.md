# kafka-sub-go

Генерирует Go-подписчик Kafka из контрактов с аннотацией `@tg kafka`.

```sh
tg kafka sub go -o internal/subscriber/kafka
```

Для каждого контракта необходимо зарегистрировать ровно одну форму обработчика:
обычную, `Meta`, `Slice` или `Batch`. Рекомендуется форма `Meta`.
