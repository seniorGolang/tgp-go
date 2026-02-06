# Init Plugin — генератор базового Go-проекта

## Описание

Плагин `init` создаёт базовый Go-проект с контрактами и каркасом сервиса. Генерируются только исходные файлы; транспорт, swagger и прочие артефакты создаются другими плагинами через `go generate`.

## Команда

```bash
tg init go --module <имя_модуля> [--out <директория>] [--json-rpc <имена>] [--rest <имена>]
```

## Опции

| Опция        | Краткая | Тип    | Обязательная | Описание                                                                 |
|--------------|---------|--------|--------------|---------------------------------------------------------------------------|
| `--module`   | `-m`    | string | да           | Имя Go-модуля (например, `some`)                                         |
| `--out`      | `-o`    | string | нет          | Относительный путь от корня (по умолчанию — корень)                       |
| `--json-rpc` | —       | string | нет*         | Имена интерфейсов JSON-RPC через запятую (например, `some,demo`)         |
| `--rest`     | —       | string | нет*         | Имена интерфейсов REST через запятую (например, `example,siteNova`)      |

\* Нужен хотя бы один из `--json-rpc` или `--rest`.

Имена интерфейсов нормализуются: в публичные имена типов (PascalCase) и в имена файлов по канонам Go (snake_case, например `siteNova` → `site_nova.go`). Для каждого интерфейса создаётся отдельный файл в `contracts/`.

## Требования

- Корень — место запуска команды. Писать можно только внутрь корня. `--out` — относительный путь от корня (пустой или `.` — сам корень).
- Директория вывода должна быть **пустой** или не существовать. Иначе генерация завершится ошибкой.

## Примеры

Создать проект с JSON-RPC-интерфейсами `Some`, `Demo` и REST-интерфейсами `Example`, `SiteNova` в текущей папке:

```bash
tg init go --module demo --json-rpc some,demo --rest example,siteNova
```

В `contracts/` появятся файлы `some.go`, `demo.go`, `example.go`, `site_nova.go` с соответствующими интерфейсами.

Только JSON-RPC-интерфейс `Some` с методом-заглушкой `DoSome(ctx context.Context) (err error)`:

```bash
tg init go -m myapp --json-rpc some
```

Только REST-интерфейс `Example` с CRUD-заглушками для сущности и DTO в `contracts/dto`:

```bash
tg init go -m myapp --rest example
```

Генерация в поддиректорию (относительно корня):

```bash
tg init go -m some -o my-service --json-rpc some --rest example
```

## Структура сгенерированного проекта

- **go.mod** — модуль с зависимостями (fiber, zerolog, envconfig и др.)
- **contracts/** — контракты:
    - **tg.go** — аннотации пакета и `//go:generate` для `tg server` и `tg swagger`
    - **\<имя\>.go** — по одному файлу на интерфейс (имя — нормализованное для Go, например `some.go`, `site_nova.go`) с аннотациями `@tg` (JSON-RPC и/или HTTP)
    - **dto/** — только при `--rest`: типы запросов/ответов для CRUD по каждому REST-интерфейсу
- **internal/config/service.go** — конфиг сервиса из переменных окружения (zerolog, envconfig)
- **internal/services/<module>/** — реализация сервиса и заглушки методов (возвращают `errs.NotImplemented`)
- **cmd/<module>/cmd.go** — точка входа: health, метрики, подключение транспорта
- **pkg/errs/** — типы ошибок и утилиты для JSON-RPC/HTTP

Транспорт (`internal/transport/`), OpenAPI (`api/swagger.json`) и т.п. создаются плагинами `server` и `swagger` при выполнении `go generate ./...` после инициализации.

## Автор

AlexK <seniorGolang@gmail.com>

## Лицензия

MIT
