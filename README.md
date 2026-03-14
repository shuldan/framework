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
- **Event ↔ Queue Bridge** — двусторонняя пересылка событий между сервисами через очередь
- **Command Bus** — межсервисная командная шина с идемпотентностью, таймаутами и ответами
- **Стандартный Envelope** — единый формат межсервисных сообщений с поддержкой трассировки
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
- [Event ↔ Queue Bridge](#event--queue-bridge)
  - [Envelope](#envelope)
  - [OutboundRelay](#outboundrelay-события--очередь)
  - [InboundRelay](#inboundrelay-очередь--события)
  - [Межсервисная коммуникация](#межсервисная-коммуникация)
- [Command Bus](#command-bus)
  - [CommandEnvelope / ResultEnvelope](#commandenvelope--resultenvelope)
  - [CommandSender (команды → очередь)](#commandsender-команды--очередь)
  - [CommandReceiver (очередь → выполнение)](#commandreceiver-очередь--выполнение)
  - [ReplyListener (ответы → callback)](#replylistener-ответы--callback)
  - [Межсервисная командная коммуникация](#межсервисная-командная-коммуникация)
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

## Event ↔ Queue Bridge

Двусторонний мост между внутренними доменными событиями и межсервисной очередью. Состоит из трёх компонентов:

```
Service A                              Queue                          Service B
┌─────────────┐                                                  ┌─────────────┐
│ Dispatcher  │──→ OutboundRelay ──→ [topic] ──→ InboundRelay ──→│ Dispatcher  │
│             │←── InboundRelay  ←── [topic] ←── OutboundRelay ←──│             │
└─────────────┘                                                  └─────────────┘
                          ↕ Envelope format ↕
```

### Envelope

Стандартный формат межсервисного события. Все события, проходящие через `OutboundRelay`, оборачиваются в Envelope.

```go
type Envelope struct {
    EventName   string          `json:"event_name"`
    AggregateID string          `json:"aggregate_id"`
    OccurredAt  time.Time       `json:"occurred_at"`
    Source      string          `json:"source,omitempty"`
    Payload     json.RawMessage `json:"payload"`

    // Зарезервированы для трассировки
    CorrelationID string `json:"correlation_id,omitempty"`
    CausationID   string `json:"causation_id,omitempty"`
    SchemaVersion string `json:"schema_version,omitempty"`
}
```

Пример JSON в очереди:

```json
{
  "event_name": "OrderCreated",
  "aggregate_id": "order-123",
  "occurred_at": "2025-01-15T10:30:00Z",
  "source": "order-service",
  "payload": {
    "order_id": "order-123",
    "customer_id": "cust-456",
    "total": 1500
  }
}
```

| Поле | Описание |
|------|----------|
| `event_name` | Имя доменного события |
| `aggregate_id` | ID агрегата-источника |
| `occurred_at` | Время возникновения события |
| `source` | Имя сервиса-отправителя (для фильтрации собственных событий) |
| `payload` | Сериализованное доменное событие |
| `correlation_id` | ID цепочки запросов (для трассировки) |
| `causation_id` | ID события-причины |
| `schema_version` | Версия схемы payload |

### OutboundRelay (события → очередь)

Выборочная пересылка доменных событий в очередь.

```go
outbound := eventbus.NewOutboundRelay(bus.Dispatcher(), broker, log,
    eventbus.WithSource("order-service"),  // записывается в envelope.source
)
defer outbound.Unsubscribe()  // отписка от диспетчера

// DDD-модуль регистрирует пересылку
func (m *OrderModule) Relays(outbound *eventbus.OutboundRelay) {
    // Все OrderCreated → в очередь (оборачиваются в Envelope)
    outbound.Forward("OrderCreated", "order.created")

    // OrderCancelled → только если сумма > 1000
    outbound.Forward("OrderCancelled", "order.cancelled",
        eventbus.WithFilter(func(e events.Event) bool {
            oc, ok := e.(*OrderCancelled)
            return ok && oc.Amount > 1000
        }),
    )

    // Кастомная сериализация (Envelope НЕ используется)
    outbound.Forward("OrderShipped", "order.shipped",
        eventbus.WithTransform(func(e events.Event) ([]byte, error) {
            return json.Marshal(map[string]string{
                "order_id": e.AggregateID(),
                "event":    e.EventName(),
            })
        }),
    )
}
```

| Опция (создание) | Тип | Описание |
|-------------------|-----|----------|
| `WithSource(service)` | `OutboundConfig` | Имя сервиса-источника, записывается в `envelope.source` |

| Опция (маршрут) | Тип | Описание |
|------------------|-----|----------|
| `WithFilter(fn)` | `OutboundOption` | Фильтрация по содержимому события |
| `WithTransform(fn)` | `OutboundOption` | Кастомная сериализация (обходит Envelope) |

**Как работает:** `OutboundRelay` подписывается на `Dispatcher` через `SubscribeAll`. При получении события проверяет регистрацию по имени, применяет фильтр, оборачивает в Envelope (или вызывает transform), отправляет через `broker.Produce(topic, data)`.

### InboundRelay (очередь → события)

Приём событий из очереди и публикация в локальный Dispatcher.

```go
inbound := eventbus.NewInboundRelay(bus.Dispatcher(), broker, log,
    eventbus.WithServiceName("payment-service"),  // фильтрация собственных событий
)

// DDD-модуль регистрирует десериализаторы
func (m *PaymentModule) InboundRelays(inbound *eventbus.InboundRelay) {
    // Из топика "order.created" события "OrderCreated" → десериализация → Dispatcher
    inbound.On("order.created", "OrderCreated",
        func(payload []byte, env *eventbus.Envelope) (events.Event, error) {
            var data struct {
                OrderID    string `json:"order_id"`
                CustomerID string `json:"customer_id"`
                Total      int    `json:"total"`
            }
            if err := json.Unmarshal(payload, &data); err != nil {
                return nil, err
            }
            return &OrderCreatedExternal{
                BaseEvent:  events.NewBaseEvent("OrderCreated", data.OrderID),
                CustomerID: data.CustomerID,
                Total:      data.Total,
            }, nil
        },
    )
}
```

**Регистрация в Queue Worker:**

`RunTopic()` возвращает функцию, совместимую с `queueworker.Registration.Run`:

```go
func (m *PaymentModule) Consumers(qw *queueworker.Module, inbound *eventbus.InboundRelay) {
    // Для каждого топика — отдельный consumer
    for _, topic := range inbound.Topics() {
        qw.Register(queueworker.Registration{
            Name: "inbound-" + topic,
            Run:  inbound.RunTopic(topic),
        })
    }
}
```

| Опция | Тип | Описание |
|-------|-----|----------|
| `WithServiceName(name)` | `InboundOption` | Фильтрация собственных событий по `envelope.source` |

**Как работает:** `InboundRelay` вызывает `broker.Consume(topic, handler)`. Каждое сообщение десериализуется из Envelope, по `event_name` находится зарегистрированный `Deserializer`, payload превращается в доменное событие, которое публикуется в локальный `Dispatcher`.

**Защита от циклов:** если `WithServiceName("my-service")` задан и `envelope.source == "my-service"` — событие пропускается.

### Множество событий в одном топике

`InboundRelay` поддерживает несколько типов событий на одном топике:

```go
inbound.On("order.events", "OrderCreated", orderCreatedDeserializer)
inbound.On("order.events", "OrderCancelled", orderCancelledDeserializer)
inbound.On("order.events", "OrderShipped", orderShippedDeserializer)

// Один consumer на весь топик
qw.Register(queueworker.Registration{
    Name: "inbound-order-events",
    Run:  inbound.RunTopic("order.events"),
})
```

### Межсервисная коммуникация

Полный цикл: Order Service публикует `OrderCreated` → Payment Service получает и обрабатывает.

**Order Service (отправитель):**

```go
// bootstrap
bus := eventbus.NewModule(eventbus.Config{Async: true, Workers: 4})
outbound := eventbus.NewOutboundRelay(bus.Dispatcher(), broker, log,
    eventbus.WithSource("order-service"),
)
outbound.Forward("OrderCreated", "order.created")

// domain — публикация события
dispatcher.Publish(ctx, &OrderCreated{
    BaseEvent:  events.NewBaseEvent("OrderCreated", orderID),
    CustomerID: customerID,
    Total:      total,
})
// → автоматически: Envelope → broker.Produce("order.created", data)
```

**Payment Service (получатель):**

```go
// bootstrap
bus := eventbus.NewModule(eventbus.Config{Async: true, Workers: 4})
inbound := eventbus.NewInboundRelay(bus.Dispatcher(), broker, log,
    eventbus.WithServiceName("payment-service"),
)

// регистрация десериализатора
inbound.On("order.created", "OrderCreated", deserializeOrderCreated)

// подписка на локальное событие
events.SubscribeFunc(bus.Dispatcher(), func(ctx context.Context, e *OrderCreatedExternal) error {
    return paymentService.CreateInvoice(ctx, e.OrderID, e.Total)
})

// регистрация consumer
qw.Register(queueworker.Registration{
    Name: "inbound-order-created",
    Run:  inbound.RunTopic("order.created"),
})
```

---

## Command Bus

Межсервисная командная шина для request/reply взаимодействия через очередь. В отличие от событий (fire-and-forget), команды подразумевают целевой сервис-получатель и опциональный ответ с результатом.

### Отличие от EventBus

| | EventBus | Command Bus |
|---|----------|-------------|
| **Семантика** | Уведомление о случившемся | Запрос на выполнение действия |
| **Получатели** | Много (pub/sub) | Один (point-to-point) |
| **Ответ** | Нет | Опциональный (reply) |
| **Идемпотентность** | На стороне подписчика | Встроенная (`IdempotencyStore`) |
| **Таймауты** | Нет | Встроенные |

```
Order Service                          Queue                       Payment Service
┌─────────────┐                                                  ┌─────────────────┐
│ CommandSender│──→ [commands.CreatePayment] ──→ CommandReceiver ──→│ handler(cmd)   │
│             │                                                  │                 │
│ ReplyListener│←── [replies.order-service]  ←── sendResult ←─────│ result/error   │
└─────────────┘                                                  └─────────────────┘
                      ↕ CommandEnvelope / ResultEnvelope ↕
```

### Компоненты

| Компонент | Назначение |
|-----------|------------|
| `commandbus.Module` | Lifecycle-модуль для локального диспетчера команд (`app.Module`) |
| `commandbus.CommandSender` | Отправка команд в очередь через `CommandEnvelope` |
| `commandbus.CommandReceiver` | Приём команд из очереди, идемпотентность, выполнение, ответ |
| `commandbus.ReplyListener` | Приём ответов (`ResultEnvelope`) и маршрутизация к callback-ам |

### CommandEnvelope / ResultEnvelope

**CommandEnvelope** — конверт команды для межсервисной доставки:

```go
type CommandEnvelope struct {
    IdempotencyKey string            `json:"idempotency_key"`
    CommandName    string            `json:"command_name"`
    ReplyTo        string            `json:"reply_to,omitempty"`
    CorrelationID  string            `json:"correlation_id"`
    Sender         string            `json:"sender,omitempty"`
    CreatedAt      time.Time         `json:"created_at"`
    Timeout        time.Duration     `json:"timeout"`
    Payload        json.RawMessage   `json:"payload"`
    SchemaVersion  string            `json:"schema_version,omitempty"`
    Headers        map[string]string `json:"headers,omitempty"`
}
```

**ResultEnvelope** — конверт результата выполнения команды:

```go
type ResultEnvelope struct {
    CorrelationID string            `json:"correlation_id"`
    CommandName   string            `json:"command_name"`
    ResultName    string            `json:"result_name,omitempty"`
    Sender        string            `json:"sender,omitempty"`
    CreatedAt     time.Time         `json:"created_at"`
    Payload       json.RawMessage   `json:"payload,omitempty"`
    Error         *string           `json:"error"`
    SchemaVersion string            `json:"schema_version,omitempty"`
    Headers       map[string]string `json:"headers,omitempty"`
}
```

Пример `CommandEnvelope` в очереди:

```json
{
  "idempotency_key": "pay-order-123-attempt-1",
  "command_name": "CreatePayment",
  "reply_to": "order-service",
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000",
  "sender": "order-service",
  "created_at": "2025-01-15T10:30:00Z",
  "timeout": 30000000000,
  "payload": {
    "order_id": "order-123",
    "amount": 1500,
    "currency": "RUB"
  }
}
```

Пример `ResultEnvelope`:

```json
{
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000",
  "command_name": "CreatePayment",
  "result_name": "PaymentCreated",
  "created_at": "2025-01-15T10:30:01Z",
  "payload": {
    "payment_id": "pay-789",
    "status": "pending"
  },
  "error": null
}
```

| Поле (CommandEnvelope) | Описание |
|------------------------|----------|
| `idempotency_key` | Ключ идемпотентности (из команды или UUID) |
| `command_name` | Имя команды для маршрутизации |
| `reply_to` | Имя сервиса для ответа (топик `replies.<reply_to>`) |
| `correlation_id` | ID для сквозной трассировки |
| `sender` | Имя сервиса-отправителя |
| `created_at` | Время создания |
| `timeout` | TTL команды — receiver отклоняет просроченные |
| `payload` | Сериализованная команда |
| `headers` | Произвольные заголовки |

| Поле (ResultEnvelope) | Описание |
|-----------------------|----------|
| `correlation_id` | Совпадает с командой для сопоставления |
| `command_name` | Имя исходной команды |
| `result_name` | Имя результата (тип) |
| `payload` | Сериализованный результат (nil при ошибке) |
| `error` | Строка ошибки (nil при успехе) |

### Module (локальный диспетчер)

`commandbus.Module` — lifecycle-обёртка над `commands.Dispatcher` для локальной обработки команд внутри одного сервиса:

```go
cmdBus := commandbus.NewModule(commandbus.Config{
    Async:      true,
    Workers:    4,
    BufferSize: 128,
    Ordered:    false,
})

// Локальный диспетчер — для команд внутри одного сервиса
dispatcher := cmdBus.Dispatcher()
```

### Конфигурация

```yaml
commands:
  async: true
  workers: 4
  buffer_size: 128
  ordered: false
```

### CommandSender (команды → очередь)

Отправка команд в удалённый сервис через очередь. Каждая команда оборачивается в `CommandEnvelope` и отправляется в топик `commands.<command_name>`.

```go
sender := commandbus.NewCommandSender(broker, log,
    commandbus.WithSender("order-service"),           // sender в envelope
    commandbus.WithReplyTo("order-service"),           // куда слать ответ
    commandbus.WithDefaultTimeout(30 * time.Second),   // таймаут по умолчанию
)

// Регистрация маршрутов
sender.Forward("CreatePayment")
sender.Forward("CancelPayment")

// Отправка команды
err := sender.Send(ctx, &CreatePayment{
    OrderID:  "order-123",
    Amount:   1500,
    Currency: "RUB",
})
// → CommandEnvelope → broker.Produce("commands.CreatePayment", data)

// С переопределением опций
err := sender.Send(ctx, &CancelPayment{OrderID: "order-123"},
    commandbus.WithTimeout(5 * time.Second),   // короткий таймаут
    commandbus.WithoutReply(),                  // fire-and-forget
    commandbus.WithHeaders(map[string]string{
        "priority": "high",
    }),
)
```

| Опция (создание) | Описание |
|-------------------|----------|
| `WithSender(name)` | Имя сервиса-отправителя |
| `WithReplyTo(name)` | Имя сервиса для получения ответов (топик `replies.<name>`) |
| `WithDefaultTimeout(d)` | Таймаут по умолчанию для всех команд |

| Опция (отправка) | Описание |
|-------------------|----------|
| `WithTimeout(d)` | Переопределение таймаута для конкретной команды |
| `WithoutReply()` | Отключение ответа (fire-and-forget) |
| `WithHeaders(h)` | Произвольные заголовки |

### CommandReceiver (очередь → выполнение)

Приём команд из очереди, проверка идемпотентности и таймаутов, выполнение и отправка результата.

```go
receiver := commandbus.NewCommandReceiver(broker, log,
    commandbus.WithIdempotencyStore(redisIdemStore),   // или memory (по умолчанию)
    commandbus.WithIdempotencyTTL(24 * time.Hour),     // TTL ключей
)

// Регистрация обработчика
err := receiver.Handle(
    "CreatePayment",                    // имя команды
    deserializeCreatePayment,           // десериализатор
    handleCreatePayment,                // обработчик
    commandbus.WithCommandIdempotencyTTL(1 * time.Hour),  // TTL для этой команды
)

// Десериализатор
func deserializeCreatePayment(
    payload []byte, env *commandbus.CommandEnvelope,
) (commands.Command, error) {
    var cmd CreatePayment
    if err := json.Unmarshal(payload, &cmd); err != nil {
        return nil, err
    }
    return &cmd, nil
}

// Обработчик
func handleCreatePayment(
    ctx context.Context, cmd commands.Command,
) (commands.Result, error) {
    c := cmd.(*CreatePayment)
    payment, err := paymentService.Create(ctx, c.OrderID, c.Amount)
    if err != nil {
        return nil, err  // → ошибка в ResultEnvelope + redelivery
    }
    return &PaymentCreated{PaymentID: payment.ID, Status: payment.Status}, nil
}
```

**Регистрация в Queue Worker:**

`Registrations()` возвращает готовые регистрации для `queueworker.Module`:

```go
func (m *PaymentModule) Consumers(qw *queueworker.Module) {
    for _, reg := range m.receiver.Registrations() {
        qw.Register(reg)
    }
}
```

| Опция (создание) | Описание |
|-------------------|----------|
| `WithIdempotencyStore(store)` | Хранилище идемпотентности (default: in-memory) |
| `WithIdempotencyTTL(ttl)` | Глобальный TTL для ключей (default: 24h) |

| Опция (обработчик) | Описание |
|---------------------|----------|
| `WithCommandIdempotencyTTL(ttl)` | TTL для конкретной команды |

**Поведение:**

1. Сообщение из очереди десериализуется в `CommandEnvelope`
2. Проверка таймаута: если `created_at + timeout < now` — команда отклоняется, ответ с ошибкой
3. Проверка идемпотентности: если ключ уже обработан — пропуск (без ответа)
4. Десериализация payload → вызов обработчика
5. При успехе — ключ помечается в `IdempotencyStore`, результат отправляется в `replies.<reply_to>`
6. При ошибке обработчика — ответ с ошибкой + возврат ошибки (очередь не ACK-ает, redelivery)

### ReplyListener (ответы → callback)

Слушает топик ответов и маршрутизирует `ResultEnvelope` к зарегистрированным callback-ам.

```go
replyListener := commandbus.NewReplyListener(broker, log,
    commandbus.WithListenerServiceName("order-service"),  // топик: replies.order-service
)

// Регистрация обработчика ответов
replyListener.OnResult(
    "CreatePayment",                      // имя команды
    deserializePaymentCreated,            // десериализатор результата
    func(ctx context.Context, result commands.Result, err error) error {
        if err != nil {
            // Команда завершилась с ошибкой на стороне получателя
            log.Error("payment creation failed", "error", err)
            return orderService.MarkPaymentFailed(ctx, correlationID)
        }
        r := result.(*PaymentCreated)
        return orderService.AttachPayment(ctx, r.PaymentID)
    },
)

// Десериализатор результата
func deserializePaymentCreated(
    payload []byte, env *commandbus.ResultEnvelope,
) (commands.Result, error) {
    var r PaymentCreated
    if err := json.Unmarshal(payload, &r); err != nil {
        return nil, err
    }
    return &r, nil
}
```

**Регистрация в Queue Worker:**

```go
qw.Register(queueworker.Registration{
    Name: "reply-listener",
    Run:  replyListener.Run,
})
```

| Опция | Описание |
|-------|----------|
| `WithListenerServiceName(name)` | Имя сервиса — определяет топик `replies.<name>` |

### Межсервисная командная коммуникация

Полный цикл: Order Service отправляет команду `CreatePayment` → Payment Service выполняет → Order Service получает результат.

**Order Service (отправитель):**

```go
// bootstrap
broker := redisBroker.New(redisClient)

// Sender — отправка команд
sender := commandbus.NewCommandSender(broker, log,
    commandbus.WithSender("order-service"),
    commandbus.WithReplyTo("order-service"),
    commandbus.WithDefaultTimeout(30 * time.Second),
)
sender.Forward("CreatePayment")
sender.Forward("CancelPayment")

// ReplyListener — приём ответов
replyListener := commandbus.NewReplyListener(broker, log,
    commandbus.WithListenerServiceName("order-service"),
)
replyListener.OnResult("CreatePayment", deserializePaymentResult, handlePaymentResult)
replyListener.OnResult("CancelPayment", deserializeCancelResult, handleCancelResult)

// Queue Worker для ответов
qw := queueworker.NewModule(log)
qw.Register(queueworker.Registration{
    Name: "reply-listener",
    Run:  replyListener.Run,
})

// domain — отправка команды
err := sender.Send(ctx, &CreatePayment{
    OrderID:  orderID,
    Amount:   total,
    Currency: "RUB",
})
```

**Payment Service (получатель):**

```go
// bootstrap
broker := redisBroker.New(redisClient)

// Receiver — приём и выполнение команд
receiver := commandbus.NewCommandReceiver(broker, log,
    commandbus.WithIdempotencyStore(commands.NewMemoryIdempotencyStore()),
    commandbus.WithIdempotencyTTL(24 * time.Hour),
)

receiver.Handle("CreatePayment", deserializeCreatePayment, func(
    ctx context.Context, cmd commands.Command,
) (commands.Result, error) {
    c := cmd.(*CreatePayment)
    payment, err := paymentService.Create(ctx, c.OrderID, c.Amount, c.Currency)
    if err != nil {
        return nil, err
    }
    return &PaymentCreated{PaymentID: payment.ID, Status: "pending"}, nil
})

// Queue Worker
qw := queueworker.NewModule(log)
for _, reg := range receiver.Registrations() {
    qw.Register(reg)
}
```

### Совместное использование EventBus и Command Bus

События и команды решают разные задачи и отлично дополняют друг друга:

```go
// Событие: "Заказ создан" — уведомление, 0+ подписчиков
outbound.Forward("OrderCreated", "order.created")

// Команда: "Создай платёж" — запрос к конкретному сервису, ожидание ответа
sender.Forward("CreatePayment")

// В обработчике события OrderCreated → отправляем команду
events.SubscribeFunc(bus.Dispatcher(), func(ctx context.Context, e *OrderCreated) error {
    return sender.Send(ctx, &CreatePayment{
        OrderID: e.OrderID,
        Amount:  e.Total,
    })
})
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
    dbm, bus, cmdBus, server, qw,
)

// Только воркеры очередей (без HTTP)
command.QueueWork("myapp", log, 15*time.Second,
    dbm, bus, cmdBus, qw,
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

    "github.com/shuldan/framework"
    "github.com/shuldan/framework/command"
    "github.com/shuldan/framework/commandbus"
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

    cmdBus := framework.NewLazy(func() (*commandbus.Module, error) {
        return commandbus.NewModule(commandbus.Config{
            Async:      cfg.GetBool("commands.async", true),
            Workers:    cfg.GetInt("commands.workers", 4),
            BufferSize: cfg.GetInt("commands.buffer_size", 128),
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
            db := dbm.MustGet()
            ev := bus.MustGet()
            cb := cmdBus.MustGet()
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

            // OutboundRelay — события → очередь
            outbound := eventbus.NewOutboundRelay(ev.Dispatcher(), br, log,
                eventbus.WithSource("myapp"),
            )
            om.Relays(outbound)

            // InboundRelay — очередь → события
            inbound := eventbus.NewInboundRelay(ev.Dispatcher(), br, log,
                eventbus.WithServiceName("myapp"),
            )
            pm.InboundRelays(inbound)

            // CommandSender — команды → очередь
            sender := commandbus.NewCommandSender(br, log,
                commandbus.WithSender("myapp"),
                commandbus.WithReplyTo("myapp"),
            )
            om.CommandRoutes(sender)

            // CommandReceiver — очередь → выполнение
            receiver := commandbus.NewCommandReceiver(br, log)
            pm.CommandHandlers(receiver)

            // ReplyListener — ответы → callback
            replyListener := commandbus.NewReplyListener(br, log,
                commandbus.WithListenerServiceName("myapp"),
            )
            om.ReplyHandlers(replyListener)

            // Queue Workers
            qw := queueworker.NewModule(log)
            pm.Consumers(qw)

            // Inbound event consumers
            for _, topic := range inbound.Topics() {
                qw.Register(queueworker.Registration{
                    Name: "inbound-" + topic,
                    Run:  inbound.RunTopic(topic),
                })
            }

            // Command receiver consumers
            for _, reg := range receiver.Registrations() {
                qw.Register(reg)
            }

            // Reply listener consumer
            qw.Register(queueworker.Registration{
                Name: "reply-listener",
                Run:  replyListener.Run,
            })

            return []app.Module{db, ev, cb, server, qw}
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

### Микросервис с двусторонним Event Bridge

```go
func main() {
    k, _ := framework.NewKernel(framework.WithConfigFile("config.yaml"))
    log := k.Logger()

    bus := eventbus.NewModule(eventbus.Config{Async: true, Workers: 4})
    broker := redisBroker.New(redisClient)

    // Outbound: наши события → очередь
    outbound := eventbus.NewOutboundRelay(bus.Dispatcher(), broker, log,
        eventbus.WithSource("payment-service"),
    )
    outbound.Forward("PaymentCompleted", "payment.completed")
    outbound.Forward("PaymentFailed", "payment.failed")

    // Inbound: чужие события → наш Dispatcher
    inbound := eventbus.NewInboundRelay(bus.Dispatcher(), broker, log,
        eventbus.WithServiceName("payment-service"),  // пропускаем свои
    )
    inbound.On("order.created", "OrderCreated", deserializeOrderCreated)
    inbound.On("order.cancelled", "OrderCancelled", deserializeOrderCancelled)

    // Локальные обработчики входящих событий
    events.SubscribeFunc(bus.Dispatcher(), handleOrderCreated)
    events.SubscribeFunc(bus.Dispatcher(), handleOrderCancelled)

    // Queue Worker
    qw := queueworker.NewModule(log)
    for _, topic := range inbound.Topics() {
        qw.Register(queueworker.Registration{
            Name: "inbound-" + topic,
            Run:  inbound.RunTopic(topic),
        })
    }

    k.Command(command.Serve("payment-svc", log, 15*time.Second, bus, qw))
    k.Run(context.Background(), os.Args[1:])
}
```

### Микросервис с Command Bus (request/reply)

```go
func main() {
    k, _ := framework.NewKernel(framework.WithConfigFile("config.yaml"))
    log := k.Logger()

    broker := redisBroker.New(redisClient)

    // ─── Command Sender (отправляем команды в Payment Service) ───
    sender := commandbus.NewCommandSender(broker, log,
        commandbus.WithSender("order-service"),
        commandbus.WithReplyTo("order-service"),
        commandbus.WithDefaultTimeout(30 * time.Second),
    )
    sender.Forward("CreatePayment")
    sender.Forward("RefundPayment")

    // ─── Reply Listener (получаем ответы) ───
    replyListener := commandbus.NewReplyListener(broker, log,
        commandbus.WithListenerServiceName("order-service"),
    )
    replyListener.OnResult("CreatePayment", deserializePaymentResult,
        func(ctx context.Context, result commands.Result, err error) error {
            if err != nil {
                return orderService.MarkPaymentFailed(ctx, err)
            }
            r := result.(*PaymentCreated)
            return orderService.AttachPayment(ctx, r.PaymentID)
        },
    )

    // ─── Queue Worker ───
    qw := queueworker.NewModule(log)
    qw.Register(queueworker.Registration{
        Name: "reply-listener",
        Run:  replyListener.Run,
    })

    k.Command(command.Serve("order-svc", log, 15*time.Second, qw))
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

func (m *Module) Relays(outbound *eventbus.OutboundRelay) {
    outbound.Forward("OrderCreated", "order.created")
    outbound.Forward("OrderCancelled", "order.cancelled")
}

func (m *Module) InboundRelays(inbound *eventbus.InboundRelay) {
    inbound.On("payment.completed", "PaymentCompleted", m.deserializePaymentCompleted)
}

func (m *Module) CommandRoutes(sender *commandbus.CommandSender) {
    sender.Forward("CreatePayment")
}

func (m *Module) ReplyHandlers(listener *commandbus.ReplyListener) {
    listener.OnResult("CreatePayment", m.deserializePaymentResult, m.onPaymentResult)
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
│   ├── config.go              — Config → events.Option
│   ├── module.go              — Module: app.Module
│   ├── envelope.go            — Envelope: стандартный формат событий
│   ├── logger.go              — Logger interface
│   ├── outbound.go            — OutboundRelay: Event → Queue
│   ├── outbound_option.go     — WithSource, WithFilter, WithTransform
│   ├── inbound.go             — InboundRelay: Queue → Event
│   └── inbound_option.go      — WithServiceName
│
├── commandbus/
│   ├── config.go              — Config → commands.Option
│   ├── module.go              — Module: app.Module (локальный диспетчер)
│   ├── envelope.go            — CommandEnvelope, ResultEnvelope
│   ├── logger.go              — Logger interface
│   ├── outbound.go            — CommandSender: Command → Queue
│   ├── outbound_option.go     — WithSender, WithReplyTo, WithDefaultTimeout
│   ├── inbound.go             — CommandReceiver: Queue → Execute → Reply
│   ├── inbound_option.go      — WithIdempotencyStore, WithIdempotencyTTL
│   ├── reply_listener.go      — ReplyListener: Reply Queue → Callback
│   └── reply_listener_option.go — WithListenerServiceName
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

commands:
  async: true
  workers: 4
  buffer_size: 128
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
