# Плагин init-go — bootstrap Go-сервиса

## Назначение

Одноразовая инициализация нового Go-модуля: контракты, заглушки сервисов, конфиг, `cmd`, ошибки, затем автоматические `go generate ./...` и `go mod tidy` (transport + OpenAPI).

Это **не** ежедневный генератор и **не** способ «добавить один контракт» в уже живой сервис.

## Когда использовать

- Новый репозиторий / пустой каталог под сервис.
- Нужен стартовый JSON-RPC и/или REST каркас с Fiber transport.
- Не используйте поверх существующего сервиса с `.go` файлами.

## Команда и параметры

```bash
tg init go --module <имя_модуля> [--out <dir>] [--json-rpc a,b] [--rest c,d]
```

| Параметр | Короткая форма | Обязательный | Описание |
|----------|----------------|--------------|----------|
| `module` | `-m` | да | Путь/имя Go-модуля (например `myapp`) |
| `out` | `-o` | нет | Каталог относительно cwd (default — текущий) |
| `json-rpc` | — | нет | Имена JSON-RPC интерфейсов через запятую |
| `rest` | — | нет | Имена REST интерфейсов через запятую |

Если не указать ни `--json-rpc`, ни `--rest` — один JSON-RPC контракт `Some`.

Имена: PascalCase тип + snake_case файл (`siteNova` → `SiteNova`, `site_nova.go`).

## Что генерируется

### Матрица флагов

| Флаги | Результат |
|-------|-----------|
| только `--json-rpc users,orders` | контракты с заглушкой `Do` на каждый |
| только `--rest users` | CRUD-контракты + `contracts/dto/` + stubs |
| оба | оба семейства |
| ни одного | один JSON-RPC `Some` |
| после записи шаблонов | `go generate ./...` → `internal/transport/`, OpenAPI; `go mod tidy` |

### Структура после generate

| Путь | Владение |
|------|----------|
| `go.mod`, `.golangci.yml`, `.gitignore` | проект / tooling |
| `contracts/`, `contracts/tg.go` | **редактируете** (API) |
| `contracts/dto/` | только при REST scaffold; редактируете |
| `internal/services/` | **редактируете** (вместо NotImplemented) |
| `internal/transport/` | **только регенерация** |
| `api/swagger.json` | **только регенерация** |
| `internal/config/` | runtime config (`SHUTDOWN_TIMEOUT` и др.) |
| `cmd/<module>/main.go` | точка входа приложения |
| `pkg/errs/` | ошибки JSON-RPC/HTTP |

## Аннотации и их влияние

Плагин сам кладёт стартовые `@tg` на scaffold-контракты (JSON-RPC и/или REST). Дальнейшие аннотации вы пишете сами; init-go их не «интерпретирует» повторно.

| Что в scaffold | Зачем |
|----------------|-------|
| `@tg jsonRPC-server` / `http-server` | чтобы `server` и `swagger` что-то сгенерировали |
| `contracts/tg.go` | метаданные + `go:generate` цепочка |
| CRUD / `Do` stubs | временные методы, не финальный домен |

Полный синтаксис: `tg plugin doc astg`. Правка API — skill `tgp-contracts`, затем снова generate.

## Использование

```bash
tg init go --module demo --json-rpc some,demo --rest example,siteNova
# или в подкаталог:
tg init go -m some -o my-service --json-rpc some --rest example
```

После успеха:

1. Реализуйте методы в `internal/services/`.
2. Меняйте контракты → `go generate ./...` (transport/OpenAPI).
3. При необходимости добавьте `client-go` / `client-ts`.

### Что уже в `cmd`

- `signal.NotifyContext` для SIGINT/SIGTERM;
- `transport.ServeHealth` (тело по умолчанию `"ok"`; readiness с зависимостями — `WithCheck` после работы с transport API);
- `srv.ShutdownWithTimeout(cfg.ShutdownTimeout)` (`SHUTDOWN_TIMEOUT`, default **10s**).

Для фиксированного budget есть и `srv.Shutdown()` (default 30s).

## На что обратить внимание

1. Каталог должен **не существовать** или **не содержать `.go` файлов**. Текст ошибки может быть `directory is not empty` даже когда лежат только non-Go файлы — ориентируйтесь на правило про `.go`.
2. Init — bootstrap once; не запускайте, чтобы «добавить контракт» в зрелый сервис.
3. **Ownership**: contracts/services правите руками; `internal/transport/` и swagger — только regenerate.
4. Выберите финальный `--module` сразу: смена пути модуля дорогая по импортам.
5. Заглушки NotImplemented — ожидаемы, пока не реализовали сервис.
6. Дальше используйте `tg server` / `tg swagger` / clients по мере роста API.

## Ограничения

1. Пишет только внутрь выбранного `--out` относительно cwd.
2. Не мигратор существующего кода.
3. Стартовые CRUD/`Do` — шаблон, не доменная модель.
4. Требует доступных tgp-плагинов для вложенного `go generate`.

## Связь с другими плагинами

| Плагин | Связь |
|--------|-------|
| `server` | генерирует `internal/transport/` |
| `swagger` | стартовый OpenAPI |
| `tgp-contracts` | развитие API после scaffold |
| `client-go` / `client-ts` | по мере появления потребителей |
| Kafka-плагины | добавляются отдельно, init их не создаёт |
