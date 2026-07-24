# Kafka subscriber Go

Используйте `tg kafka sub go -o <dir>` для генерации подписчика из `@tg kafka` контрактов.

Для каждого контракта передавайте в `kafka.New` одну из опций обработчика:
`Contract`, `ContractMeta`, `ContractSlice` или `ContractBatch`.
Рекомендуется `ContractMeta`, так как она передаёт ключ, заголовки и offset в `Meta`.
