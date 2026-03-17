# `framework` — Модульный фреймворк для DDD-приложений на Go

[![Go CI](https://github.com/shuldan/framework/workflows/Go%20CI/badge.svg)](https://github.com/shuldan/framework/actions)
[![codecov](https://codecov.io/gh/shuldan/framework/branch/main/graph/badge.svg)](https://codecov.io/gh/shuldan/framework)
[![Go Report Card](https://goreportcard.com/badge/github.com/shuldan/framework)](https://goreportcard.com/report/github.com/shuldan/framework)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

Фреймворк для создания Go-приложений по принципам Domain-Driven Design. Тонкий Kernel, manual wiring, ленивая инициализация. Один бинарь — HTTP-сервер, воркеры очередей, миграции и утилиты через CLI-команды.

Построен на экосистеме пакетов [`shuldan`](https://github.com/shuldan): `app`, `cli`, `commands`, `config`, `errors`, `events`, `migrator`, `queue`, `repository`.

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
- **EventBus** — доменная шина событий с async/sync режимами, middleware, retry, ordered delivery
- **Command Bus** — командная шина с client/server, типизированными handler-ами, Future/TypedFuture и pluggable transport
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
  ├── commandbus.Module    ← опционально
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
queue:work    → БД + Events + Commands + Queue
serve         → всё
```

`Lazy[T]` гарантирует: каждый компонент создаётся **ровно один раз** (`sync.Once`), даже при запросе из нескольких модулей.

### Structural typing для логгеров

Каждый пакет определяет собственный минимальный интерфейс `Logger`:

```go
type Logger interface {
    Info(msg string, args ...any)
    Error(msg string, args ...any)
}
```

`framework/logger.Logger` удовлетворяет всем этим интерфейсам автоматически — никаких импортных зависимостей между пакетами, никаких адаптеров. Nil-safe: при передаче `nil` используется `noopLogger`.

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
  - [Dispatcher](#dispatcher)
  - [Подписка на события](#подписка-на-события)
  - [Middleware событий](#middleware-событий)
  - [Retry и Timeout](#retry-и-timeout)
  - [Ordered Delivery](#ordered-delivery)
  - [Transport](#event-transport)
- [Command Bus](#command-bus)
  - [Transport](#command-transport)
  - [Codec](#command-codec)
  - [Command и Result](#command-и-result)
  - [CommandServer](#commandserver)
  - [CommandClient](#commandclient)
  - [Future и TypedFuture](#future-и-typedfuture)
  - [ReplySender и отложенные ответы](#replysender-и-отложенные-ответы)
  - [Ошибки командной шины](#ошибки-командной-шины)
  - [Commandbus Module (lifecycle)](#commandbus-module-lifecycle)
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

Множество именованных подключений с отдельными пулами. Реализует `app.Module` и `HealthChecker`.

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

// HealthChecker
dbm.Health(ctx)   // Ping всех, возвращает errors.Join
```

### HealthChecker interface

Любой компонент может участвовать в проверке здоровья, реализовав:

```go
type HealthChecker interface {
    Name() string
    Health(ctx context.Context) error
}
```

`database.Manager` реализует этот интерфейс автоматически. Собственные компоненты (Redis, внешние API) подключаются так же.

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

Обёртка над `events.Dispatcher` как `app.Module`. Dispatcher создаётся из пакета `shuldan/events` с нужными опциями и передаётся в модуль.

### Dispatcher

```go
import (
    "github.com/shuldan/events"
    "github.com/shuldan/framework/eventbus"
)

// Создание диспетчера с нужными опциями
dispatcher := events.New(
    events.WithAsyncMode(),
    events.WithWorkerPool(8),
    events.WithErrorHandler(func(ctx context.Context, event events.Event, err error) {
        log.Error("event handler failed", "error", err)
    }),
    events.WithMiddleware(
        middleware.NewRecovery(),
        middleware.NewLogging(log),
    ),
)

// Обёртка в app.Module для lifecycle-управления
bus := eventbus.NewModule(dispatcher)

// Получение диспетчера для DDD-модулей
d := bus.Dispatcher()
```

`eventbus.Module` реализует `app.Module`:
- `Init()` — no-op
- `Start()` — no-op
- `Stop(ctx)` — вызывает `dispatcher.Close(ctx)`, дожидается завершения in-flight обработчиков

### Подписка на события

Типизированная подписка через generic-функцию `events.Subscribe[E]()`:

```go
import "github.com/shuldan/events"

// Доменное событие — любой тип
type OrderCreated struct {
    OrderID    string
    CustomerID string
    Total      int
}

// Обработчик реализует events.Handler[E]
type OrderCreatedHandler struct {
    service *PaymentService
}

func (h *OrderCreatedHandler) Handle(ctx context.Context, e *OrderCreated) error {
    return h.service.CreateInvoice(ctx, e.OrderID, e.Total)
}

// Подписка
sub := events.Subscribe[*OrderCreated](dispatcher, &OrderCreatedHandler{
    service: paymentService,
})

// Отписка
sub.Unsubscribe()
```

Публикация событий:

```go
// Одно событие
err := dispatcher.Publish(ctx, &OrderCreated{
    OrderID:    "order-123",
    CustomerID: "cust-456",
    Total:      1500,
})

// Несколько событий
err := dispatcher.PublishAll(ctx,
    &OrderCreated{OrderID: "order-123"},
    &InventoryReserved{OrderID: "order-123"},
)
```

### Middleware событий

Система middleware для цепочки обработки событий. Middleware оборачивают `events.Next` и могут добавлять логику до/после обработчика.

```go
import "github.com/shuldan/events/middleware"

// Recovery — перехват паник в обработчиках
recovery := middleware.NewRecovery()

// Logging — логирование обработки
logging := middleware.NewLogging(log)

// Metrics — запись метрик (duration, errors)
recorder := middleware.NewInMemoryRecorder()
metrics := middleware.NewMetrics(recorder)

// Глобальный middleware — применяется ко всем подписчикам
dispatcher := events.New(
    events.WithMiddleware(recovery, logging, metrics),
)

// Per-subscriber middleware — только для конкретной подписки
events.Subscribe[*OrderCreated](dispatcher, handler,
    events.WithSubscribeMiddleware(customMiddleware),
)
```

Порядок выполнения: `global middleware → per-subscriber middleware → retry → timeout → handler`.

**Собственный middleware:**

```go
type auditMiddleware struct{}

func (m *auditMiddleware) Wrap(next events.Next) events.Next {
    return &auditNext{next: next}
}

type auditNext struct {
    next events.Next
}

func (n *auditNext) Handle(ctx context.Context, event events.Event) error {
    // До обработки
    log.Info("processing event", "type", fmt.Sprintf("%T", event))

    err := n.next.Handle(ctx, event)

    // После обработки
    if err != nil {
        log.Error("event failed", "error", err)
    }
    return err
}
```

### Retry и Timeout

Настраиваются per-subscriber через `SubscribeOption`:

```go
// Retry с exponential backoff
events.Subscribe[*PaymentFailed](dispatcher, handler,
    events.WithRetry(events.RetryPolicy{
        MaxRetries:   3,
        InitialDelay: 100 * time.Millisecond,
        MaxDelay:     5 * time.Second,
        Multiplier:   2.0,
    }),
)

// Timeout на обработку
events.Subscribe[*OrderCreated](dispatcher, handler,
    events.WithTimeout(10 * time.Second),
)

// Комбинация: таймаут охватывает все retry-попытки
events.Subscribe[*OrderCreated](dispatcher, handler,
    events.WithTimeout(30 * time.Second),
    events.WithRetry(events.RetryPolicy{
        MaxRetries:   3,
        InitialDelay: 1 * time.Second,
        Multiplier:   2.0,
    }),
)
```

Дефолтные опции для всех подписчиков:

```go
dispatcher := events.New(
    events.WithDefaultSubscribeOptions(
        events.WithRetry(events.RetryPolicy{MaxRetries: 2, InitialDelay: 100 * time.Millisecond, Multiplier: 2.0}),
        events.WithTimeout(15 * time.Second),
    ),
)
```

### Ordered Delivery

События с одинаковым ключом обрабатываются последовательно (в async-режиме). Событие должно реализовать интерфейс `events.KeyedEvent`:

```go
type OrderStatusChanged struct {
    OrderID string
    Status  string
}

// KeyedEvent — события одного заказа обрабатываются последовательно
func (e *OrderStatusChanged) EventKey() string {
    return e.OrderID
}
```

В async-режиме для каждого уникального ключа создаётся отдельная очередь. События без ключа обрабатываются пулом воркеров без гарантий порядка.

### Event Transport

`events.Transport` — абстракция для передачи событий через внешний брокер. Подключается к Dispatcher для двусторонней пересылки.

```go
import (
    "github.com/shuldan/events"
    "github.com/shuldan/events/codec"
    memtransport "github.com/shuldan/events/transport/memory"
)

transport := memtransport.New()
jsonCodec := codec.NewJSON()

dispatcher := events.New(
    events.WithTransport(transport),
    events.WithCodec(jsonCodec),
    events.WithAsyncMode(),
)
```

**Интерфейс Transport:**

```go
type Transport interface {
    Publish(ctx context.Context, envelope Envelope) error
    Subscribe(ctx context.Context, handler TransportHandler) error
    Close(ctx context.Context) error
}
```

**Envelope — обёртка события для транспорта:**

```go
type Envelope struct {
    ID          string
    Type        string            // reflect.TypeOf(event).String()
    Key         string            // EventKey() если реализован
    Payload     []byte            // сериализованное событие
    ContentType string            // "application/json"
    Metadata    map[string]string
    Timestamp   time.Time
}
```

**Codec — сериализация событий:**

```go
type Codec interface {
    Encode(event Event) ([]byte, error)
    Decode(data []byte, target Event) error
    ContentType() string
}
```

При публикации события Dispatcher автоматически сериализует его и отправляет через Transport. При получении из Transport — десериализует и публикует в локальный Dispatcher.

---

## Command Bus

Командная шина для request/reply взаимодействия. Состоит из `CommandClient` (отправка команд, получение ответов через Future) и `CommandServer` (приём команд, выполнение обработчиков, отправка ответов). Взаимодействие происходит через абстракцию `Transport`.

```
CommandClient                    Transport                    CommandServer
┌─────────────┐                                             ┌───────────────────┐
│ Send(cmd)   │──→ CommandEnvelope ──→ Subscribe handler ──→│ Handler[C].Handle │
│             │                                             │                   │
│ Future.Await│←── ReplyEnvelope  ←── SubscribeReplies   ←──│ ReplySender.Send  │
└─────────────┘                                             └───────────────────┘
```

### Command Transport

Абстракция над брокером сообщений. Разделяет отправку команд и доставку ответов.

```go
type Transport interface {
    Send(ctx context.Context, env CommandEnvelope) error
    Subscribe(ctx context.Context, handler CommandHandler) error
    Reply(ctx context.Context, env ReplyEnvelope) error
    SubscribeReplies(ctx context.Context, handler ReplyHandler) error
    ReplyAddress() string
    Close(ctx context.Context) error
}
```

**In-memory transport** (для тестов и single-process):

```go
import memtransport "github.com/shuldan/commands/transport/memory"

transport := memtransport.New()
// transport.ReplyAddress() == "memory://reply"
```

### Command Codec

Сериализация/десериализация команд и результатов:

```go
type Codec interface {
    Encode(v any) ([]byte, error)
    Decode(data []byte, v any) error
}
```

**JSON codec:**

```go
import jsoncodec "github.com/shuldan/commands/codec/json"

codec := jsoncodec.New()
```

### Command и Result

Команды и результаты — интерфейсы с маркерным методом:

```go
// Команда
type Command interface {
    CommandName() string
}

// Результат
type Result interface {
    ResultName() string
}
```

**Пример:**

```go
type CreatePayment struct {
    OrderID  string `json:"order_id"`
    Amount   int    `json:"amount"`
    Currency string `json:"currency"`
}

func (c *CreatePayment) CommandName() string { return "CreatePayment" }

type PaymentCreated struct {
    PaymentID string `json:"payment_id"`
    Status    string `json:"status"`
}

func (r *PaymentCreated) ResultName() string { return "PaymentCreated" }
```

### CommandServer

Принимает команды через Transport, маршрутизирует к типизированным обработчикам, отправляет ответы.

```go
import "github.com/shuldan/commands"

server, err := commands.NewCommandServer(transport, codec,
    commands.WithServerLogger(log),
)
```

**Регистрация обработчиков** — типобезопасная через generic-функцию `commands.Register[C]()`:

```go
// Обработчик реализует commands.Handler[C]
type CreatePaymentHandler struct {
    service *PaymentService
}

func (h *CreatePaymentHandler) Handle(
    ctx context.Context,
    cmd *CreatePayment,
    reply commands.ReplySender,
) error {
    payment, err := h.service.Create(ctx, cmd.OrderID, cmd.Amount)
    if err != nil {
        // Отправка ошибки клиенту
        return reply.SendError(ctx, err)
    }

    // Отправка результата клиенту
    return reply.Send(ctx, &PaymentCreated{
        PaymentID: payment.ID,
        Status:    payment.Status,
    })
}

// Регистрация (до Open)
err := commands.Register[*CreatePayment](server, &CreatePaymentHandler{
    service: paymentService,
})
```

Ограничения:
- Регистрация обработчиков только **до** вызова `Open()`
- Один обработчик на одно имя команды
- При дублировании — `ErrAlreadyRegistered`

**Lifecycle:**

```go
// Подписка на Transport (начинает приём команд)
err := server.Open(ctx)

// Закрытие: ожидание in-flight обработчиков, закрытие transport
err := server.Close(ctx)
```

Каждая команда обрабатывается в отдельной горутине. `Close()` дожидается завершения всех in-flight обработчиков.

### CommandClient

Отправляет команды через Transport и получает ответы через Future.

```go
client, err := commands.NewCommandClient(transport, codec,
    commands.WithTimeout(30 * time.Second),       // дефолтный таймаут
    commands.WithClientLogger(log),
)
```

**Отправка команды (нетипизированная):**

```go
future, err := client.Send(ctx, &CreatePayment{
    OrderID:  "order-123",
    Amount:   1500,
    Currency: "RUB",
})
if err != nil {
    return err
}

// Ожидание результата
result, err := future.Await(ctx)
```

**Типизированная отправка** — generic-функция `commands.Send[R]()`:

```go
future, err := commands.Send[*PaymentCreated](ctx, client, &CreatePayment{
    OrderID:  "order-123",
    Amount:   1500,
    Currency: "RUB",
},
    commands.WithSendTimeout(5 * time.Second), // переопределение таймаута
)
if err != nil {
    return err
}

// Типобезопасный результат — *PaymentCreated, не Result
payment, err := future.Await(ctx)
if err != nil {
    return err
}
fmt.Println(payment.PaymentID, payment.Status)
```

**Lifecycle:**

```go
// Подписка на ответы через Transport
err := client.Open(ctx)

// Закрытие: все pending futures завершаются с ErrClientClosed
err := client.Close(ctx)
```

### Future и TypedFuture

**Future** — результат отправки команды. Позволяет ожидать ответ синхронно или проверять готовность.

```go
type Future interface {
    // Блокирует до получения результата или отмены контекста
    Await(ctx context.Context) (Result, error)

    // Канал, закрывается при получении результата
    Done() <-chan struct{}

    // Неблокирующая проверка; третий аргумент — готовность
    Result() (Result, error, bool)
}
```

**TypedFuture[R]** — типобезопасная версия, возвращается из `commands.Send[R]()`:

```go
type TypedFuture[R Result] interface {
    Await(ctx context.Context) (R, error)
    Done() <-chan struct{}
    Result() (R, error, bool)
}
```

**Поведение при таймауте:** если ответ не получен в течение заданного таймаута — Future завершается с `commands.ErrTimeout`.

**Поведение при закрытии клиента:** все pending futures завершаются с `commands.ErrClientClosed`.

**Неблокирующая проверка:**

```go
future, _ := client.Send(ctx, cmd)

// Продолжаем работу...
doOtherWork()

// Проверяем результат без блокировки
if result, err, ok := future.Result(); ok {
    // Результат готов
    handleResult(result, err)
} else {
    // Ещё не готов — ждём
    result, err := future.Await(ctx)
    handleResult(result, err)
}
```

**Ожидание через канал:**

```go
select {
case <-future.Done():
    result, err, _ := future.Result()
    // ...
case <-time.After(5 * time.Second):
    // fallback
}
```

### ReplySender и отложенные ответы

`ReplySender` — интерфейс для отправки ответов из обработчика команды:

```go
type ReplySender interface {
    Send(ctx context.Context, result Result) error
    SendError(ctx context.Context, err error) error
    Address() ReplyAddress
}
```

**Немедленный ответ** (внутри обработчика):

```go
func (h *Handler) Handle(ctx context.Context, cmd *MyCommand, reply commands.ReplySender) error {
    result, err := h.service.Do(ctx, cmd)
    if err != nil {
        return reply.SendError(ctx, err)
    }
    return reply.Send(ctx, result)
}
```

**Отложенный ответ** — сохранение `ReplyAddress` и отправка позже:

```go
func (h *Handler) Handle(ctx context.Context, cmd *MyCommand, reply commands.ReplySender) error {
    // Сохраняем адрес для ответа
    addr := reply.Address()
    h.saveReplyAddress(cmd.OrderID, addr)
    // Не отправляем ответ сейчас — он будет отправлен позже
    return nil
}

// Позже, в другом месте (например, по событию):
func (h *Handler) OnPaymentCompleted(ctx context.Context, orderID string) error {
    addr := h.loadReplyAddress(orderID)

    // Создаём ReplySender из сохранённого адреса
    sender := commands.NewReplySender(transport, codec, addr)
    return sender.Send(ctx, &PaymentCompleted{OrderID: orderID})
}
```

`ReplyAddress` содержит `CorrelationID` и `ReplyTo` — всё необходимое для маршрутизации ответа.

**Обработка ошибок в SendError:**

```go
// *commands.ErrorPayload — структурированная ошибка, передаётся клиенту как есть
reply.SendError(ctx, &commands.ErrorPayload{
    Code:    "INSUFFICIENT_FUNDS",
    Message: "not enough balance",
})

// Обычный error — оборачивается в ErrorPayload{Code: "UNKNOWN", Message: err.Error()}
reply.SendError(ctx, fmt.Errorf("something went wrong"))
```

### Ошибки командной шины

Предопределённые ошибки:

| Ошибка | Описание |
|--------|----------|
| `ErrTimeout` | Ответ не получен в течение таймаута |
| `ErrClientClosed` | Клиент закрыт, операция невозможна |
| `ErrServerClosed` | Сервер закрыт |
| `ErrAlreadyOpened` | Повторный вызов `Open()` |
| `ErrNotOpened` | Вызов `Send`/`Close` до `Open()` |
| `ErrAlreadyRegistered` | Повторная регистрация обработчика для того же имени команды |
| `ErrServerStarted` | Регистрация обработчика после `Open()` |

**ErrorPayload** — структурированная бизнес-ошибка, передаётся через transport:

```go
type ErrorPayload struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}
```

Предопределённые:
- `ErrInternal` — `{Code: "INTERNAL", Message: "internal server error"}`
- `ErrCommandNotFound` — `{Code: "COMMAND_NOT_FOUND", Message: "no handler registered"}`

При ошибке декодирования команды или панике обработчика — автоматически отправляется `ErrInternal`.

### Envelope-ы командной шины

**CommandEnvelope** — обёртка команды для передачи через transport:

```go
type CommandEnvelope struct {
    MessageID     string            `json:"message_id"`
    CorrelationID string            `json:"correlation_id"`
    CommandName   string            `json:"command_name"`
    ReplyTo       string            `json:"reply_to"`
    Headers       map[string]string `json:"headers,omitempty"`
    Payload       []byte            `json:"payload"`
}
```

**ReplyEnvelope** — обёртка ответа:

```go
type ReplyEnvelope struct {
    CorrelationID string            `json:"correlation_id"`
    ResultName    string            `json:"result_name,omitempty"`
    Headers       map[string]string `json:"headers,omitempty"`
    Payload       []byte            `json:"payload,omitempty"`
    Error         *ErrorPayload     `json:"error,omitempty"`
}
```

`MessageID` и `CorrelationID` генерируются автоматически (crypto/rand, 16 байт hex). `CorrelationID` связывает команду с ответом.

### Commandbus Module (lifecycle)

`commandbus.Module` — обёртка `app.Module` для lifecycle-управления клиентом и сервером:

```go
import "github.com/shuldan/framework/commandbus"

module := commandbus.NewModule(
    commandbus.WithClient(client),
    commandbus.WithServer(server),
)

// Доступ к компонентам
module.Client() // *commands.CommandClient
module.Server() // *commands.CommandServer
```

Lifecycle:
- `Init()` — no-op
- `Start(ctx)` — вызывает `server.Open(ctx)`, затем `client.Open(ctx)`
- `Stop(ctx)` — вызывает `client.Close(ctx)`, затем `server.Close(ctx)`

Оба компонента опциональны — можно создать Module только с клиентом или только с сервером.

### Полный пример Command Bus

```go
import (
    "github.com/shuldan/commands"
    jsoncodec "github.com/shuldan/commands/codec/json"
    memtransport "github.com/shuldan/commands/transport/memory"
    "github.com/shuldan/framework/commandbus"
)

// --- Transport и Codec ---
transport := memtransport.New()
codec := jsoncodec.New()

// --- Server ---
server, _ := commands.NewCommandServer(transport, codec,
    commands.WithServerLogger(log),
)

commands.Register[*CreatePayment](server, &CreatePaymentHandler{
    service: paymentService,
})

// --- Client ---
client, _ := commands.NewCommandClient(transport, codec,
    commands.WithTimeout(30 * time.Second),
    commands.WithClientLogger(log),
)

// --- Module (lifecycle) ---
module := commandbus.NewModule(
    commandbus.WithClient(client),
    commandbus.WithServer(server),
)

// --- Отправка команды ---
future, err := commands.Send[*PaymentCreated](ctx, client, &CreatePayment{
    OrderID: "order-123",
    Amount:  1500,
})

payment, err := future.Await(ctx)
// payment.PaymentID, payment.Status
```

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

// Интроспекция
fmt.Println(qw.ConsumerCount()) // 1
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
// Полный сервер: HTTP + Events + Commands + Queue
command.Serve("myapp", log, 15*time.Second,
    dbm, bus, cmdModule, server, qw,
)

// Только воркеры очередей (без HTTP)
command.QueueWork("myapp", log, 15*time.Second,
    dbm, bus, cmdModule, qw,
)
```

### Run-and-exit команды

Выполняют действие и завершаются. Без `app.Run()`.

```go
command.MigrateUp(runner)       // migrate:up [--connection=default]
command.MigrateDown(runner)     // migrate:down [--steps=1] [--force] [--connection=default]
command.MigrateStatus(runner)   // migrate:status [--connection=default]
command.MigratePlan(runner)     // migrate:plan [--connection=default]
command.Health(checkers...)     // health (принимает ...HealthChecker)
command.ConfigDump(cfg)         // config:dump [--no-mask]
```

### Health — проверка здоровья

`Health()` принимает любое количество компонентов, реализующих `HealthChecker`:

```go
type HealthChecker interface {
    Name() string
    Health(ctx context.Context) error
}
```

```go
// database.Manager реализует HealthChecker
command.Health(dbm)

// Несколько компонентов
command.Health(dbm, redisChecker, externalAPIChecker)

// Без проверок — всегда healthy
command.Health()
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
    "github.com/shuldan/commands"
    jsoncodec "github.com/shuldan/commands/codec/json"
    memcmdtransport "github.com/shuldan/commands/transport/memory"
    "github.com/shuldan/events"
    eventmiddleware "github.com/shuldan/events/middleware"

    "github.com/shuldan/framework"
    "github.com/shuldan/framework/command"
    "github.com/shuldan/framework/commandbus"
    "github.com/shuldan/framework/database"
    "github.com/shuldan/framework/eventbus"
    "github.com/shuldan/framework/httpserver"
    "github.com/shuldan/framework/httpserver/middleware"
    "github.com/shuldan/framework/migration"
    "github.com/shuldan/framework/queueworker"

    "myapp/internal/module/order"
    "myapp/internal/module/payment"
    "myapp/internal/module/user"
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
        dispatcher := events.New(
            events.WithAsyncMode(),
            events.WithWorkerPool(cfg.GetInt("events.workers", 8)),
            events.WithErrorHandler(func(_ context.Context, event events.Event, err error) {
                log.Error("event handler failed", "event", fmt.Sprintf("%T", event), "error", err)
            }),
            events.WithMiddleware(
                eventmiddleware.NewRecovery(),
                eventmiddleware.NewLogging(log),
            ),
        )
        return eventbus.NewModule(dispatcher), nil
    })

    cmdTransport := framework.NewLazy(func() (*memcmdtransport.Transport, error) {
        return memcmdtransport.New(), nil
    })

    cmdModule := framework.NewLazy(func() (*commandbus.Module, error) {
        t := cmdTransport.MustGet()
        codec := jsoncodec.New()

        server, err := commands.NewCommandServer(t, codec,
            commands.WithServerLogger(log),
        )
        if err != nil {
            return nil, err
        }

        client, err := commands.NewCommandClient(t, codec,
            commands.WithTimeout(30*time.Second),
            commands.WithClientLogger(log),
        )
        if err != nil {
            return nil, err
        }

        return commandbus.NewModule(
            commandbus.WithClient(client),
            commandbus.WithServer(server),
        ), nil
    })

    // ─── Lazy DDD Modules ────────────────────
    orderMod := framework.NewLazy(func() (*order.Module, error) {
        db := dbm.MustGet()
        ev := bus.MustGet()
        cm := cmdModule.MustGet()
        return order.NewModule(db.Default(), ev.Dispatcher(), cm.Client(), cfg), nil
    })

    userMod := framework.NewLazy(func() (*user.Module, error) {
        db := dbm.MustGet()
        return user.NewModule(db.Default(), cfg), nil
    })

    paymentMod := framework.NewLazy(func() (*payment.Module, error) {
        db := dbm.MustGet()
        cm := cmdModule.MustGet()
        return payment.NewModule(db.Default(), cm.Server(), cfg), nil
    })

    // ─── Serve Command ──────────────────────
    buildServe := func() cli.Command {
        return command.Serve("myapp", log, 15*time.Second, func() []app.Module {
            db := dbm.MustGet()
            ev := bus.MustGet()
            cm := cmdModule.MustGet()
            om := orderMod.MustGet()
            um := userMod.MustGet()
            pm := paymentMod.MustGet()

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

            // Commands
            pm.CommandHandlers(cm.Server())
            om.CommandSenders(cm.Client())

            // Queue Workers
            qw := queueworker.NewModule(log)
            pm.Consumers(qw)

            return []app.Module{db, ev, cm, server, qw}
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
        cmdTransport.IfCreated(func(t *memcmdtransport.Transport) { _ = t.Close(context.Background()) })
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

    qw := queueworker.NewModule(k.Logger())
    paymentMod := payment.NewModule(dbm.Default(), k.Config())
    paymentMod.Consumers(qw)

    k.Command(command.QueueWork("worker", k.Logger(), 15*time.Second, dbm, qw))
    k.Run(context.Background(), os.Args[1:])
}
```

### Микросервис с Command Bus

```go
func main() {
    k, _ := framework.NewKernel(framework.WithConfigFile("config.yaml"))
    log := k.Logger()

    transport := memtransport.New()
    codec := jsoncodec.New()

    // --- Payment Service: Server ---
    server, _ := commands.NewCommandServer(transport, codec,
        commands.WithServerLogger(log),
    )
    commands.Register[*CreatePayment](server, &CreatePaymentHandler{})
    commands.Register[*RefundPayment](server, &RefundPaymentHandler{})

    // --- Order Service: Client ---
    client, _ := commands.NewCommandClient(transport, codec,
        commands.WithTimeout(30*time.Second),
        commands.WithClientLogger(log),
    )

    module := commandbus.NewModule(
        commandbus.WithClient(client),
        commandbus.WithServer(server),
    )

    k.Command(command.Serve("payment-svc", log, 15*time.Second, module))
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
│   └── job/               ← обработчики очередей / команд
└── module.go              ← регистрация модуля
```

### Пример модуля

```go
// internal/module/order/module.go
package order

import (
    "context"

    "github.com/shuldan/commands"
    "github.com/shuldan/events"

    "github.com/shuldan/framework/httpserver"
    "github.com/shuldan/framework/migration"
)

type Module struct {
    interactor *interactor.OrderInteractor
    repo       *persistence.OrderRepository
    client     *commands.CommandClient
}

func NewModule(
    db *sql.DB,
    dispatcher *events.Dispatcher,
    client *commands.CommandClient,
    cfg config.ConfigProvider,
) *Module {
    repo := persistence.NewOrderRepository(db, repository.Postgres())
    publisher := adapter.NewEventPublisher(dispatcher)
    inter := interactor.New(repo, publisher, client, cfg)
    return &Module{interactor: inter, repo: repo, client: client}
}

func (m *Module) Routes(router *httpserver.Router) {
    api := router.Group("/api/v1/orders")
    api.GET("", httpserver.Wrap(m.listOrders))
    api.GET("/{id}", httpserver.Wrap(m.getOrder))
    api.POST("", httpserver.Wrap(m.createOrder))
}

func (m *Module) Listeners(d *events.Dispatcher) {
    events.Subscribe[*PaymentCompleted](d, &onPaymentCompleted{
        interactor: m.interactor,
    })
}

func (m *Module) CommandSenders(client *commands.CommandClient) {
    // Команды отправляются из interactor через client
    // Регистрация маршрутов не нужна — client.Send() принимает любую команду
}

func (m *Module) Migrations(runner *migration.Runner) {
    runner.Register("default", m.migrations()...)
}
```

```go
// internal/module/payment/module.go
package payment

import (
    "github.com/shuldan/commands"

    "github.com/shuldan/framework/queueworker"
)

type Module struct {
    service *PaymentService
}

func NewModule(
    db *sql.DB,
    server *commands.CommandServer,
    cfg config.ConfigProvider,
) *Module {
    service := NewPaymentService(db, cfg)
    m := &Module{service: service}
    m.registerCommandHandlers(server)
    return m
}

func (m *Module) registerCommandHandlers(server *commands.CommandServer) {
    commands.Register[*CreatePayment](server, &CreatePaymentHandler{
        service: m.service,
    })
    commands.Register[*RefundPayment](server, &RefundPaymentHandler{
        service: m.service,
    })
}

func (m *Module) Listeners(d *events.Dispatcher) {
    events.Subscribe[*OrderCreated](d, &onOrderCreated{
        service: m.service,
    })
}

func (m *Module) Consumers(qw *queueworker.Module) {
    qw.Register(queueworker.Registration{
        Name: "payment-processor",
        Run:  m.processPayments,
    })
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
│   ├── middleware.go          — Middleware type, applyChain
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
│   └── module.go              — Module: app.Module (обёртка events.Dispatcher)
│
├── commandbus/
│   └── module.go              — Module: app.Module (client + server lifecycle)
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
    ├── health.go              — health (HealthChecker interface)
    └── config_dump.go         — config:dump
```

### Внешние пакеты

```
shuldan/commands/                      — командная шина
├── command.go                         — Command, Result интерфейсы
├── envelope.go                        — CommandEnvelope, ReplyEnvelope, ReplyAddress
├── errors.go                          — ErrorPayload, ErrTimeout, ErrClientClosed, ...
├── transport.go                       — Transport, CommandHandler, ReplyHandler
├── codec.go                           — Codec интерфейс
├── client.go                          — CommandClient, Send[R](), ClientOption, SendOption
├── server.go                          — CommandServer, Register[C](), ServerOption
├── handler.go                         — Handler[C], ReplySender
├── reply_sender.go                    — replySender, NewReplySender
├── future.go                          — Future, TypedFuture[R], future, typedFuture[R]
├── logger.go                          — Logger интерфейс
├── codec/json/                        — JSON Codec
└── transport/memory/                  — In-memory Transport

shuldan/events/                        — шина событий
├── event.go                           — Event, KeyedEvent
├── handler.go                         — Handler[E]
├── dispatcher.go                      — Dispatcher, New(), Publish, PublishAll, Close
├── subscribe.go                       — Subscribe[E](), Subscription
├── subscribe_options.go               — WithRetry, WithTimeout, WithSubscribeMiddleware
├── options.go                         — WithAsyncMode, WithWorkerPool, WithMiddleware, ...
├── middleware.go                      — Next, Middleware, buildChain
├── transport.go                       — Transport, Envelope, TransportHandler
├── codec.go                           — Codec
├── retry.go                           — RetryPolicy
├── errors.go                          — ErrDispatcherClosed, ErrNilHandler
├── middleware/
│   ├── logging.go                     — NewLogging
│   ├── metrics.go                     — NewMetrics, InMemoryRecorder
│   └── recovery.go                    — NewRecovery
├── codec/                             — JSON Codec
└── transport/memory/                  — In-memory Transport
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
- Каждый пакет определяет свой Logger interface (structural typing)

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
