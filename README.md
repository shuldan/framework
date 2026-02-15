# `framework` — Модульный фреймворк для DDD-приложений на Go

[![Go CI](https://github.com/shuldan/framework/workflows/Go%20CI/badge.svg)](https://github.com/shuldan/framework/actions)
[![codecov](https://codecov.io/gh/shuldan/framework/branch/main/graph/badge.svg)](https://codecov.io/gh/shuldan/framework)
[![Go Report Card](https://goreportcard.com/badge/github.com/shuldan/framework)](https://goreportcard.com/report/github.com/shuldan/framework)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

Фреймворк для создания Go-приложений по принципам Domain-Driven Design. Тонкий Kernel, manual wiring, ленивая инициализация. Один бинарь — HTTP-сервер, воркеры очередей, миграции и утилиты через CLI-команды.

Построен на экосистеме пакетов [`shuldan`](https://github.com/shuldan): `app`, `cli`, `config`, `errors`, `events`, `migrator`, `queue`, `repository`.

---

## 🚀 Основные возможности

- **Тонкий Kernel** — знает только о конфигурации и логгере. Не знает о базе данных, очередях, событиях
- **Manual wiring** — зависимости передаются через конструкторы. Нет магии, нет DI-контейнера
- **Ленивая инициализация** — `Lazy[T]` создаёт компоненты по требованию. `config:dump` не трогает Redis
- **Один бинарь** — `serve`, `migrate:up`, `queue:work`, `health` через CLI
- **Расширяемость без ломки** — новый компонент (cache, search, ...) = 0 изменений в framework
- **Компоненты опциональны** — API Gateway без БД, Worker без HTTP
- **HTTP-сервер** — обёртка Go 1.22+ `ServeMux` с middleware, группами, error-хелперами
- **Множество БД** — именованные подключения с отдельными пулами и миграциями
- **Event → Queue relay** — выборочная пересылка доменных событий в очередь
- **Structured logging** — единый `slog`-логгер для всех компонентов
- **Domain errors → HTTP** — автоматический маппинг `errors.Kind` в HTTP-статус и JSON

---

## 📦 Установка

```sh
go get github.com/shuldan/framework
```

Требуется Go 1.24+.

---

## 🏗️ Архитектура

### Принцип

```
Kernel  ──→  знает только о cfg + log + CLI
  │
  │  main.go / bootstrap: manual wiring
  │
  ├── database.Manager     ← опционально
  ├── eventbus.Module      ← опционально
  ├── httpserver.Module    ← опционально
  ├── queueworker.Module  ← опционально
  ├── migration.Runner     ← опционально
  └── DDD Modules          ← получают зависимости через конструкторы
```

Kernel **никогда не меняется** при добавлении новых инфраструктурных компонентов. Новый пакет (cache, search, scheduler) подключается в `main.go` без изменений framework.

### Ленивая инициализация

```
config:dump   → ничего не создаётся
health        → только БД (Lazy.Get())
migrate:up    → только БД
queue:work    → БД + Events + Queue
serve         → всё
```

`Lazy[T]` гарантирует: каждый компонент создаётся **ровно один раз** (`sync.Once`), даже при запросе из нескольких модулей.

---

## 📖 Содержание

- [Быстрый старт](#-быстрый-старт)
- [Kernel](#kernel)
- [Lazy[T]](#lazyt)
- [Logger](#logger)
- [HTTP Server](#http-server)
    - [Router](#router)
    - [Request / Response](#request--response)
    - [Middleware](#middleware)
    - [Domain Errors → HTTP](#domain-errors--http)
- [Database Manager](#database-manager)
- [EventBus](#eventbus)
- [Event → Queue Relay](#event--queue-relay)
- [Queue Worker](#queue-worker)
- [Migration Runner](#migration-runner)
- [CLI-команды](#cli-команды)
- [Сценарии использования](#-сценарии-использования)
- [Структура DDD-модуля](#структура-ddd-модуля)
- [Структура пакетов](#структура-пакетов)
- [Разработка](#-разработка)

---

## 🚀 Быстрый старт

### Минимальный HTTP-сервер

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "time"

    "github.com/shuldan/cli"
    "github.com/shuldan/config"

    "github.com/shuldan/framework"
    "github.com/shuldan/framework/command"
    "github.com/shuldan/framework/httpserver"
    "github.com/shuldan/framework/httpserver/middleware"
)

func main() {
    k, err := framework.NewKernel(
        framework.WithConfigFile("config.yaml"),
        framework.WithEnvPrefix("APP_"),
    )
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }

    log := k.Logger()

    // Router
    router := httpserver.NewRouter()
    router.Use(
        middleware.Recovery(log.Error),
        middleware.RequestID(),
        middleware.Logging(log.Info),
    )
    router.GET("/ping", func(w http.ResponseWriter, _ *http.Request) {
        httpserver.OK(w, map[string]string{"status": "ok"})
    })

    // HTTP Server module
    server := httpserver.NewModule(router, httpserver.Config{
        Host: k.Config().GetString("server.host", "0.0.0.0"),
        Port: k.Config().GetInt("server.port", 8080),
    })

    // CLI
    k.Command(
        command.Serve("myapp", log, 15*time.Second, server),
        command.Health(),
        command.ConfigDump(k.Config()),
    )

    if err := k.Run(context.Background(), os.Args[1:]); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(cli.GetExitCode(err))
    }
}
```

```sh
myapp serve          # Запуск HTTP-сервера
myapp health         # Проверка здоровья
myapp config:dump    # Вывод конфигурации
myapp help           # Справка
```

---

## Kernel

Точка входа фреймворка. Знает только о конфигурации, логгере и CLI.

```go
k, err := framework.NewKernel(
    framework.WithConfigFile("config.yaml"),    // YAML-файл (default)
    framework.WithEnvPrefix("APP_"),            // APP_SERVER__PORT=9090
    framework.WithProfileEnv("APP_ENV"),        // config.production.yaml
)
```

### Методы

| Метод | Описание |
|-------|----------|
| `Config() *config.Config` | Загруженная конфигурация |
| `Logger() *logger.Logger` | Structured-логгер |
| `Command(cmds ...cli.Command)` | Регистрация CLI-команд |
| `OnShutdown(fn func())` | Callback при завершении (LIFO) |
| `Run(ctx, args) error` | Парсинг args → выполнение команды |
| `RunWith(ctx, in, out, args) error` | То же, с кастомным I/O (для тестов) |

### Опции

| Опция | Описание |
|-------|----------|
| `WithConfigFile(paths...)` | Пути к конфиг-файлам. Пустой список = без файлов |
| `WithEnvPrefix(prefix)` | Загрузка из ENV с авто-парсингом типов |
| `WithProfileEnv(envVar)` | Профильные конфиги: `config.production.yaml` |
| `WithLogger(log)` | Предсобранный логгер (bypass конфига) |
| `WithConfig(cfg)` | Предсобранная конфигурация (для тестов) |

### OnShutdown для Lazy-ресурсов

```go
dbm := framework.NewLazy(func() (*database.Manager, error) {
    return database.NewManager(configs, log)
})

k.OnShutdown(func() {
    dbm.IfCreated(func(m *database.Manager) {
        _ = m.Stop(context.Background())
    })
})
```

---

## Lazy[T]

Потокобезопасная ленивая инициализация. Фабрика вызывается ровно один раз.

```go
dbm := framework.NewLazy(func() (*database.Manager, error) {
    return database.NewManager(configs, log)
})

// Первый вызов — создаёт. Последующие — возвращают кэш.
manager, err := dbm.Get()

// Паника при ошибке
manager := dbm.MustGet()

// Был ли создан успешно?
if dbm.IsCreated() { ... }

// Callback только если создан (для shutdown)
dbm.IfCreated(func(m *database.Manager) {
    _ = m.Stop(ctx)
})
```

| Метод | Описание |
|-------|----------|
| `Get() (T, error)` | Создаёт при первом вызове, кэширует результат |
| `MustGet() T` | Паника при ошибке |
| `IsCreated() bool` | `true` если фабрика вернула без ошибки |
| `IfCreated(func(T))` | Callback только для успешно созданных значений |

---

## Logger

Structured-логгер на базе `slog`. Совместим с `app.Logger`, `config.Logger` и `migrator.Logger` через structural typing.

```go
// Из конфигурации
log := logger.New(logger.Config{
    Level:  "info",     // debug, info, warn, error
    Format: "json",     // json, text
    Output: "stdout",   // stdout, stderr
})

// Методы
log.Info("server started", "port", 8080)
log.Error("connection failed", "error", err)
log.Debug("query executed", "sql", query, "duration", d)
log.Warn("deprecated endpoint", "path", "/old")

// Дочерний логгер с контекстом
moduleLog := log.With("module", "orders")
moduleLog.Info("order created", "id", orderID)

// Для тестов — запись в буфер
var buf bytes.Buffer
testLog := logger.NewWithWriter(&buf, logger.Config{Level: "debug"})
```

Конфигурация из YAML:

```yaml
log:
  level: info
  format: json
  output: stdout
```

---

## HTTP Server

### Router

Обёртка Go 1.22+ `http.ServeMux` с поддержкой middleware и групп.

```go
router := httpserver.NewRouter()

// Глобальный middleware
router.Use(
    middleware.Recovery(log.Error),
    middleware.RequestID(),
    middleware.Logging(log.Info),
)

// Маршруты
router.GET("/health", healthHandler)
router.POST("/users", createUserHandler)

// Группы с префиксом и middleware
api := router.Group("/api/v1", authMiddleware)
api.GET("/orders", listOrdersHandler)
api.GET("/orders/{id}", getOrderHandler)
api.POST("/orders", createOrderHandler)

admin := router.Group("/admin", adminAuthMiddleware)
admin.DELETE("/users/{id}", deleteUserHandler)
```

Методы: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `Handle(method, pattern, handler)`.

Параметры пути — нативный синтаксис Go 1.22: `/users/{id}`, `/files/{path...}`.

### Request / Response

**Запрос:**

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Параметр пути: /users/{id}
    id := httpserver.PathParam(r, "id")

    // Query-параметр: /search?q=hello
    query := httpserver.QueryParam(r, "q")

    // JSON body → struct (лимит 1 MB по умолчанию)
    var input CreateOrderInput
    if err := httpserver.Bind(r, &input); err != nil {
        httpserver.Error(w, err)
        return
    }

    // Кастомный лимит
    if err := httpserver.BindWithLimit(r, &input, 10<<20); err != nil { // 10 MB
        httpserver.Error(w, err)
        return
    }
}
```

**Ответ:**

```go
// 200 OK с JSON
httpserver.OK(w, map[string]any{"users": users})

// 201 Created с JSON
httpserver.Created(w, map[string]string{"id": newID})

// 204 No Content
httpserver.NoContent(w)

// Произвольный статус
httpserver.JSON(w, http.StatusAccepted, data)

// Ошибка → автоматический HTTP-статус из errors.Kind
httpserver.Error(w, domainErr)
```

**Wrap — handler, возвращающий error:**

```go
router.GET("/users/{id}", httpserver.Wrap(func(w http.ResponseWriter, r *http.Request) error {
    id := httpserver.PathParam(r, "id")

    user, err := userService.Find(r.Context(), id)
    if err != nil {
        return err // Error() вызовется автоматически
    }

    httpserver.OK(w, user)
    return nil
}))
```

### Middleware

Все middleware принимают `func(msg string, args ...any)` вместо конкретного логгера — нет импортных зависимостей.

**Recovery** — перехват паник:

```go
middleware.Recovery(log.Error)
```

Паника → 500 JSON `{"code":"internal","message":"internal error"}`, стектрейс в лог.

**RequestID** — генерация/проброс X-Request-Id:

```go
middleware.RequestID()

// Извлечение из контекста
id := middleware.IDFromContext(r.Context())
```

Если заголовок `X-Request-Id` присутствует — используется. Иначе — генерируется.

**Logging** — логирование запросов:

```go
middleware.Logging(log.Info)
```

Логирует: method, path, status, duration, request_id.

**CORS** — Cross-Origin Resource Sharing:

```go
middleware.CORS(middleware.CORSConfig{
    AllowedOrigins: []string{"https://example.com", "https://app.example.com"},
    AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders: []string{"Content-Type", "Authorization"},
    MaxAge:         86400,
})

// Wildcard
middleware.CORS(middleware.CORSConfig{
    AllowedOrigins: []string{"*"},
})
```

Обрабатывает preflight (`OPTIONS`) запросы автоматически.

### Domain Errors → HTTP

`httpserver.Error(w, err)` использует `shuldan/errors` для маппинга:

| `errors.Kind` | HTTP Status | Пример |
|:---:|:---:|---|
| `Validation` | 400 | Невалидный ввод |
| `Authentication` | 401 | Не авторизован |
| `Authorization` | 403 | Нет доступа |
| `NotFound` | 404 | Ресурс не найден |
| `Conflict` | 409 | Конфликт версий |
| `DomainRule` | 422 | Бизнес-правило нарушено |
| `Infrastructure` | 503 | Сервис недоступен |
| `Internal` | 500 | Внутренняя ошибка |

```go
// Domain layer
var ErrOrderNotFound = errors.NewCode("ORDER_NOT_FOUND").
    Kind(errors.NotFound).
    New("order {{.ID}} not found")

// Presentation layer
func getOrder(w http.ResponseWriter, r *http.Request) error {
    order, err := service.Find(ctx, id)
    if err != nil {
        return err // → 404 {"code":"ORDER_NOT_FOUND","message":"order 123 not found"}
    }
    httpserver.OK(w, order)
    return nil
}
```

### HTTP Server Module

`app.BackgroundModule` — listener создаётся в `Init`, порт слушается в `Start`.

```go
server := httpserver.NewModule(router, httpserver.Config{
    Host:         "0.0.0.0",
    Port:         8080,
    ReadTimeout:  15 * time.Second,
    WriteTimeout: 15 * time.Second,
    IdleTimeout:  60 * time.Second,
})

// После Init — доступен реальный адрес
server.Init(ctx)
fmt.Println(server.Addr()) // "0.0.0.0:8080"
```

При `Port: 0` — выбирается свободный порт (удобно в тестах).

---

## Database Manager

Множество именованных подключений с отдельными пулами. Реализует `app.Module` и `app.HealthChecker`.

```go
configs := map[string]database.ConnectionConfig{
    "default": {
        Driver:          "postgres",
        DSN:             cfg.GetString("database.connections.default.dsn"),
        MaxOpenConns:    25,
        MaxIdleConns:    5,
        ConnMaxLifetime: 5 * time.Minute,
    },
    "analytics": {
        Driver:       "postgres",
        DSN:          cfg.GetString("database.connections.analytics.dsn"),
        MaxOpenConns: 10,
    },
}

dbm, err := database.NewManager(configs, log)
```

### Работа с подключениями

```go
// По имени
db := dbm.Connection("default")
analyticsDB := dbm.Connection("analytics")

// Shortcut для "default"
db := dbm.Default()

// Драйвер
driver := dbm.Driver("default") // "postgres"

// Проверка существования
if dbm.Has("analytics") { ... }

// Все имена (детерминированный порядок)
names := dbm.Names() // ["analytics", "default"]
```

### Lifecycle

```go
// app.Module
dbm.Init(ctx)    // no-op (подключения открыты в конструкторе)
dbm.Start(ctx)   // Ping всех подключений
dbm.Stop(ctx)    // Close в обратном порядке

// app.HealthChecker
dbm.Health(ctx)   // Ping всех, возвращает errors.Join
```

### Конфигурация

```yaml
database:
  connections:
    default:
      driver: postgres
      dsn: "{{ env \"DATABASE_URL\" }}"
      max_open_conns: 25
      max_idle_conns: 5
      conn_max_lifetime: 5m
    analytics:
      driver: postgres
      dsn: "{{ env \"ANALYTICS_DB_URL\" }}"
      max_open_conns: 10
```

---

## EventBus

Обёртка над `events.Dispatcher` как `app.Module`.

```go
bus := eventbus.NewModule(eventbus.Config{
    Async:      true,
    Workers:    8,
    BufferSize: 256,
    Ordered:    false,
})

// Получение диспетчера для DDD-модулей
dispatcher := bus.Dispatcher()
```

DDD-модуль подписывается на события:

```go
func (m *OrderModule) Listeners(d *events.Dispatcher) {
    events.SubscribeFunc(d, func(ctx context.Context, e *PaymentReceived) error {
        return m.interactor.ConfirmOrder(ctx, e.OrderID)
    })
}
```

### Конфигурация

```yaml
events:
  async: true
  workers: 8
  buffer_size: 256
  ordered: false    # true — события одного агрегата последовательно
```

---

## Event → Queue Relay

Выборочная пересылка доменных событий в очередь.

```go
relay := eventbus.NewRelay(bus.Dispatcher(), broker, log)

// DDD-модуль регистрирует пересылку
func (m *OrderModule) Relays(relay *eventbus.Relay) {
    // Все OrderCreated → в очередь
    relay.Forward("OrderCreated", "order.created")

    // OrderCancelled → только если сумма > 1000
    relay.Forward("OrderCancelled", "order.cancelled",
        eventbus.WithFilter(func(e events.Event) bool {
            oc, ok := e.(*OrderCancelled)
            return ok && oc.Amount > 1000
        }),
    )

    // Кастомная сериализация
    relay.Forward("OrderShipped", "order.shipped",
        eventbus.WithTransform(func(e events.Event) ([]byte, error) {
            return json.Marshal(map[string]string{
                "order_id": e.AggregateID(),
                "event":    e.EventName(),
            })
        }),
    )
}
```

| Опция | Описание |
|-------|----------|
| `WithFilter(fn)` | Дополнительная фильтрация по содержимому события |
| `WithTransform(fn)` | Кастомная сериализация (default: `json.Marshal`) |

**Как работает:** Relay подписывается на `Dispatcher` через `SubscribeAll`. При получении события проверяет, зарегистрировано ли имя, применяет фильтр, сериализует и вызывает `broker.Produce(topic, data)`.

---

## Queue Worker

Управляет lifecycle consumer-ов очередей. Реализует `app.BackgroundModule`.

```go
qw := queueworker.NewModule(log)

// DDD-модуль регистрирует consumers
func (m *PaymentModule) Consumers(qw *queueworker.Module) {
    q, _ := queue.New[*ProcessPayment](broker,
        queue.WithTopic("payment.process"),
        queue.WithMaxRetries(3),
    )

    qw.Register(queueworker.Registration{
        Name: "payment-processor",
        Run: func(ctx context.Context) error {
            return q.Consume(ctx, m.interactor.ProcessPayment)
        },
    })
}
```

### Поведение

- `Start()` — запускает goroutine для каждого consumer
- `Stop()` — отменяет context, ждёт завершения всех goroutine
- `Err()` — первая fatal-ошибка consumer-а триггерит shutdown приложения
- `context.Canceled` / `DeadlineExceeded` — не считаются ошибками (нормальный shutdown)

---

## Migration Runner

Оркестрация миграций по нескольким БД-подключениям.

```go
runner := migration.NewRunner(dbm, log,
    migration.WithMigrationTable("schema_migrations"),
    migration.WithAdvisoryLock(),
)

// DDD-модуль регистрирует миграции
func (m *OrderModule) Migrations(runner *migration.Runner) {
    runner.Register("default",
        migrator.CreateMigration("20240101_001", "Create orders table").
            CreateTable("orders",
                "id UUID PRIMARY KEY",
                "customer_id UUID NOT NULL",
                "status VARCHAR(50) NOT NULL DEFAULT 'draft'",
                "version INTEGER NOT NULL DEFAULT 1",
                "created_at TIMESTAMP NOT NULL DEFAULT NOW()",
            ).MustBuild(),

        migrator.CreateMigration("20240101_002", "Create order items table").
            CreateTable("order_items",
                "id UUID PRIMARY KEY",
                "order_id UUID NOT NULL REFERENCES orders(id)",
                "product_id UUID NOT NULL",
                "quantity INTEGER NOT NULL",
            ).MustBuild(),
    )
}
```

### Диалект определяется автоматически из имени драйвера

| Driver | Dialect |
|--------|---------|
| `postgres`, `pgx` | PostgreSQL |
| `mysql` | MySQL |
| `sqlite`, `sqlite3` | SQLite |

---

## CLI-команды

### Lifecycle-команды (долгоживущие)

Создают `app.Application`, регистрируют модули, вызывают `app.Run()`.

```go
// Полный сервер: HTTP + Events + Queue
command.Serve("myapp", log, 15*time.Second,
    dbm, bus, server, qw,
)

// Только воркеры очередей (без HTTP)
command.QueueWork("myapp", log, 15*time.Second,
    dbm, bus, qw,
)
```

### Run-and-exit команды

Выполняют действие и завершаются. Без `app.Run()`.

```go
command.MigrateUp(runner)       // migrate:up [--connection=default]
command.MigrateDown(runner)     // migrate:down [--steps=1] [--force] [--connection=default]
command.MigrateStatus(runner)   // migrate:status [--connection=default]
command.MigratePlan(runner)     // migrate:plan [--connection=default]
command.Health(dbm, broker)     // health
command.ConfigDump(cfg)         // config:dump [--no-mask]
```

### Таблица всех команд

| Команда | Группа | Описание | Тип |
|---------|--------|----------|-----|
| `serve` | server | HTTP-сервер + фоновые воркеры | lifecycle |
| `queue:work` | queue | Только consumer-ы очередей | lifecycle |
| `migrate:up` | database | Применить pending миграции | run-and-exit |
| `migrate:down` | database | Откатить N миграций | run-and-exit |
| `migrate:status` | database | Таблица статусов миграций | run-and-exit |
| `migrate:plan` | database | Показать SQL без выполнения | run-and-exit |
| `health` | debug | Проверка здоровья сервисов | run-and-exit |
| `config:dump` | debug | Вывод конфига (секреты маскируются) | run-and-exit |

### Маскировка секретов

`config:dump` автоматически маскирует значения ключей, содержащих: `password`, `secret`, `token`, `key`, `dsn`, `credential`.

```sh
myapp config:dump
# database:
#   dsn: ***
# api:
#   token: ***

myapp config:dump --no-mask
# database:
#   dsn: postgres://user:pass@localhost/db
```

---

## 🏛️ Сценарии использования

### Полноценный backend с lazy-инициализацией

```go
package main

import (
    "context"
    "fmt"
    "os"
    "time"

    "github.com/shuldan/cli"

    "github.com/shuldan/framework"
    "github.com/shuldan/framework/command"
    "github.com/shuldan/framework/database"
    "github.com/shuldan/framework/eventbus"
    "github.com/shuldan/framework/httpserver"
    "github.com/shuldan/framework/httpserver/middleware"
    "github.com/shuldan/framework/migration"
    "github.com/shuldan/framework/queueworker"

    memorymq "github.com/shuldan/queue/broker/memory"

    "myapp/internal/module/order"
    "myapp/internal/module/user"
    "myapp/internal/module/payment"
)

func main() {
    // ─── Foundation ──────────────────────────
    k, err := framework.NewKernel(
        framework.WithConfigFile("config.yaml"),
        framework.WithEnvPrefix("APP_"),
        framework.WithProfileEnv("APP_ENV"),
    )
    if err != nil {
        fatal(err)
    }
    cfg := k.Config()
    log := k.Logger()

    // ─── Lazy Infrastructure ─────────────────
    dbm := framework.NewLazy(func() (*database.Manager, error) {
        return database.NewManager(map[string]database.ConnectionConfig{
            "default": {
                Driver:       "postgres",
                DSN:          cfg.GetString("database.connections.default.dsn"),
                MaxOpenConns: cfg.GetInt("database.connections.default.max_open_conns", 25),
            },
        }, log)
    })

    bus := framework.NewLazy(func() (*eventbus.Module, error) {
        return eventbus.NewModule(eventbus.Config{
            Async:      cfg.GetBool("events.async", true),
            Workers:    cfg.GetInt("events.workers", 8),
            BufferSize: cfg.GetInt("events.buffer_size", 256),
        }), nil
    })

    broker := framework.NewLazy(func() (*memorymq.Broker, error) {
        return memorymq.New(), nil  // или redisBroker.New(...)
    })

    // ─── Lazy DDD Modules ────────────────────
    orderMod := framework.NewLazy(func() (*order.Module, error) {
        db := dbm.MustGet()
        ev := bus.MustGet()
        return order.NewModule(db.Default(), ev.Dispatcher(), cfg), nil
    })

    userMod := framework.NewLazy(func() (*user.Module, error) {
        db := dbm.MustGet()
        return user.NewModule(db.Default(), cfg), nil
    })

    paymentMod := framework.NewLazy(func() (*payment.Module, error) {
        db := dbm.MustGet()
        br := broker.MustGet()
        return payment.NewModule(db.Default(), br, cfg), nil
    })

    // ─── Serve Command ──────────────────────
    buildServe := func() cli.Command {
        return command.Serve("myapp", log, 15*time.Second, func() []app.Module {
            // Эти Get() триггерят создание
            db := dbm.MustGet()
            ev := bus.MustGet()
            om := orderMod.MustGet()
            um := userMod.MustGet()
            pm := paymentMod.MustGet()
            br := broker.MustGet()

            // Router
            router := httpserver.NewRouter()
            router.Use(
                middleware.Recovery(log.Error),
                middleware.RequestID(),
                middleware.Logging(log.Info),
            )
            om.Routes(router)
            um.Routes(router)
            pm.Routes(router)
            server := httpserver.NewModule(router, httpserver.Config{
                Port: cfg.GetInt("server.port", 8080),
            })

            // Events
            om.Listeners(ev.Dispatcher())
            pm.Listeners(ev.Dispatcher())

            // Relay
            relay := eventbus.NewRelay(ev.Dispatcher(), br, log)
            om.Relays(relay)

            // Queue
            qw := queueworker.NewModule(log)
            pm.Consumers(qw)

            return []app.Module{db, ev, server, qw}
        }())
    }

    // ─── Migration Runner ───────────────────
    buildRunner := func() *migration.Runner {
        db := dbm.MustGet()
        runner := migration.NewRunner(db, log, migration.WithAdvisoryLock())
        orderMod.MustGet().Migrations(runner)
        userMod.MustGet().Migrations(runner)
        paymentMod.MustGet().Migrations(runner)
        return runner
    }

    // ─── CLI Commands ───────────────────────
    k.Command(
        buildServe(),
        command.MigrateUp(buildRunner()),
        command.MigrateDown(buildRunner()),
        command.MigrateStatus(buildRunner()),
        command.MigratePlan(buildRunner()),
        command.Health(dbm.MustGet()),
        command.ConfigDump(cfg),
    )

    // ─── Shutdown ───────────────────────────
    k.OnShutdown(func() {
        broker.IfCreated(func(b *memorymq.Broker) { _ = b.Close() })
        dbm.IfCreated(func(m *database.Manager) { _ = m.Stop(context.Background()) })
    })

    // ─── Run ────────────────────────────────
    if err := k.Run(context.Background(), os.Args[1:]); err != nil {
        fatal(err)
    }
}

func fatal(err error) {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
}
```

> **Примечание:** в реальном проекте `main.go` делегирует сборку в `internal/bootstrap/`, оставляя себе ~10 строк.

### API Gateway (только HTTP, без БД)

```go
func main() {
    k, _ := framework.NewKernel(
        framework.WithConfigFile("gateway.yaml"),
    )

    router := httpserver.NewRouter()
    router.Use(
        middleware.Recovery(k.Logger().Error),
        middleware.CORS(middleware.CORSConfig{AllowedOrigins: []string{"*"}}),
    )

    // Proxy routes, rate limiting — без БД, событий, очередей
    gateway.RegisterRoutes(router, k.Config())

    server := httpserver.NewModule(router, httpserver.Config{Port: 443})

    k.Command(command.Serve("gateway", k.Logger(), 10*time.Second, server))
    k.Run(context.Background(), os.Args[1:])
}
```

### Worker (только очереди, без HTTP)

```go
func main() {
    k, _ := framework.NewKernel(framework.WithConfigFile("config.yaml"))

    dbm, _ := database.NewManager(configs, k.Logger())
    broker := redisBroker.New(redisClient)

    qw := queueworker.NewModule(k.Logger())
    paymentMod := payment.NewModule(dbm.Default(), broker, k.Config())
    paymentMod.Consumers(qw)

    k.Command(command.QueueWork("worker", k.Logger(), 15*time.Second, dbm, qw))
    k.Run(context.Background(), os.Args[1:])
}
```

---

## Структура DDD-модуля

```
internal/module/order/
├── application/
│   ├── business/
│   │   ├── emitter/       ← публикация доменных событий
│   │   ├── operation/     ← use-case операции
│   │   └── policy/        ← application-level политики
│   ├── interactor/        ← оркестрация use-cases
│   └── port/              ← интерфейсы внешних зависимостей
├── domain/
│   ├── business/
│   │   ├── emitter/       ← интерфейсы публикаторов событий
│   │   ├── operation/     ← интерфейсы use-case операций
│   │   └── policy/        ← интерфейсы политик
│   ├── model/             ← агрегаты, value objects, entities
│   └── persistence/       ← интерфейсы репозиториев
├── infrastructure/
│   ├── adapter/           ← реализация портов
│   ├── migration/         ← миграции модуля
│   └── persistence/       ← реализация репозиториев
├── presentation/
│   ├── api/               ← HTTP обработчики
│   ├── command/           ← CLI команды модуля
│   ├── listener/          ← обработчики событий
│   └── job/               ← обработчики очередей
└── module.go              ← регистрация модуля
```

### Пример модуля

```go
// internal/module/order/module.go
package order

type Module struct {
    interactor *interactor.OrderInteractor
    repo       *persistence.OrderRepository
}

func NewModule(db *sql.DB, dispatcher *events.Dispatcher, cfg config.ConfigProvider) *Module {
    repo := persistence.NewOrderRepository(db, repository.Postgres())
    publisher := adapter.NewEventPublisher(dispatcher)
    inter := interactor.New(repo, publisher, cfg)
    return &Module{interactor: inter, repo: repo}
}

func (m *Module) Routes(router *httpserver.Router) {
    api := router.Group("/api/v1/orders")
    api.GET("", httpserver.Wrap(m.listOrders))
    api.GET("/{id}", httpserver.Wrap(m.getOrder))
    api.POST("", httpserver.Wrap(m.createOrder))
}

func (m *Module) Listeners(d *events.Dispatcher) {
    events.SubscribeFunc(d, m.onPaymentReceived)
}

func (m *Module) Relays(relay *eventbus.Relay) {
    relay.Forward("OrderCreated", "order.created")
}

func (m *Module) Migrations(runner *migration.Runner) {
    runner.Register("default", m.migrations()...)
}
```

---

## Структура пакетов

```
shuldan/framework/
├── lazy.go                    — Lazy[T] (ленивая инициализация)
├── kernel.go                  — Kernel (cfg + log + CLI)
├── kernel_option.go           — WithConfigFile, WithEnvPrefix, ...
├── kernel_build.go            — buildConfig, buildLogger, buildConsole
│
├── logger/
│   └── logger.go              — slog-обёртка, Config, New, With
│
├── database/
│   ├── config.go              — ConnectionConfig
│   ├── errors.go              — ErrNoConnections, ErrConnectionNotFound
│   ├── logger.go              — Logger interface
│   └── manager.go             — Manager: app.Module + HealthChecker
│
├── httpserver/
│   ├── config.go              — Config (host, port, timeouts)
│   ├── errors.go              — ErrEmptyBody, ErrBodyTooLarge, ErrInvalidJSON
│   ├── middleware.go           — Middleware type, applyChain
│   ├── router.go              — Router: обёртка ServeMux
│   ├── server.go              — Module: app.BackgroundModule
│   ├── request.go             — Bind, PathParam, QueryParam
│   ├── response.go            — JSON, OK, Created, Error, Wrap
│   └── middleware/
│       ├── recovery.go        — перехват паник
│       ├── requestid.go       — X-Request-Id + context
│       ├── logging.go         — лог запросов
│       └── cors.go            — CORS
│
├── eventbus/
│   ├── config.go              — Config → events.Option
│   ├── module.go              — Module: app.Module
│   └── relay.go               — Relay: Event → Queue
│
├── queueworker/
│   └── module.go              — Module: app.BackgroundModule
│
├── migration/
│   ├── runner.go              — Runner: миграции по подключениям
│   └── option.go              — WithMigrationTable, WithAdvisoryLock
│
└── command/
    ├── serve.go               — serve (lifecycle)
    ├── queue_work.go          — queue:work (lifecycle)
    ├── migrate_up.go          — migrate:up
    ├── migrate_down.go        — migrate:down
    ├── migrate_status.go      — migrate:status
    ├── migrate_plan.go        — migrate:plan
    ├── health.go              — health
    └── config_dump.go         — config:dump
```

---

## Базовая конфигурация

```yaml
# config.yaml
app:
  name: myapp
  version: 1.0.0
  environment: development

server:
  host: 0.0.0.0
  port: 8080
  read_timeout: 15s
  write_timeout: 15s
  idle_timeout: 60s

database:
  connections:
    default:
      driver: postgres
      dsn: "{{ env \"DATABASE_URL\" | default \"postgres://localhost:5432/myapp?sslmode=disable\" }}"
      max_open_conns: 25
      max_idle_conns: 5
      conn_max_lifetime: 5m

events:
  async: true
  workers: 8
  buffer_size: 256
  ordered: false

queue:
  broker: memory
  workers: 4
  max_retries: 3
  dlq: true

log:
  level: info
  format: json
  output: stdout
```

```yaml
# config.production.yaml
server:
  port: 443

database:
  connections:
    default:
      max_open_conns: 100

log:
  level: warn
```

```sh
APP_ENV=production myapp serve
# Loads: config.yaml → config.production.yaml → ENV
```

---

## 🛠️ Разработка

### Установка инструментов

```sh
make install-tools
```

Устанавливает:
- `golangci-lint` (v2.4.0)
- `goimports`
- `gosec`

### Команды

| Команда | Описание |
|---------|----------|
| `make all` | Форматирование, линтер, security, тесты |
| `make ci` | CI-пайплайн |
| `make fmt` | Форматирование кода |
| `make test` | Запуск тестов |
| `make test-coverage` | Тесты с отчётом о покрытии |

### Требования к PR

- Покрытие тестами нового функционала (≥ 70%)
- Соответствие `golangci-lint`
- Функции ≤ 60 строк, ≤ 40 выражений
- Нет circular imports между пакетами
- Middleware принимают функции, не конкретные типы

---

## 📄 Лицензия

Проект распространяется под лицензией [MIT](LICENSE).

---

## 🤝 Вклад в проект

PR и issue приветствуются! Обязательно соблюдайте стиль кода и покрывайте новый функционал тестами.

---

> **Автор**: MSeytumerov
> **Репозиторий**: `github.com/shuldan/framework`
> **Go**: `1.24.2`