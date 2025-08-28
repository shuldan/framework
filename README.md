# Shuldan Framework

[![Go CI](https://github.com/shuldan/framework/workflows/Go%20CI/badge.svg)](https://github.com/shuldan/framework/actions)
[![codecov](https://codecov.io/gh/shuldan/framework/branch/main/graph/badge.svg)](https://codecov.io/gh/shuldan/framework)
[![Go Report Card](https://goreportcard.com/badge/github.com/shuldan/framework)](https://goreportcard.com/report/github.com/shuldan/framework)
[![GoDoc](https://godoc.org/github.com/shuldan/framework?status.svg)](https://godoc.org/github.com/shuldan/framework)

Shuldan — это современный, легковесный и модульный фреймворк для создания приложений на Go с поддержкой внедрения зависимостей, управления жизненным циклом и множества встроенных компонентов.

## ✨ Основные возможности

### 🏗️ Архитектура
- **Модульная система** — легко расширяемые компоненты через унифицированный интерфейс
- **Dependency Injection** — мощный контейнер зависимостей с поддержкой фабрик и синглтонов
- **Graceful Shutdown** — корректное завершение работы с настраиваемым таймаутом

### 🛠️ Встроенные модули
- **CLI** — полнофункциональная система консольных команд с автоматической справкой
- **Logger** — структурированное логирование с поддержкой уровней и цветного вывода
- **Config** — гибкая система конфигурации (YAML, JSON, ENV с приоритетами)
- **Database** — ORM-агностик работа с БД, миграции, репозитории и стратегии загрузки
- **Events** — событийная система с асинхронной обработкой
- **Queue** — система очередей с retry, DLQ и метриками

### 🔧 Дополнительные возможности
- **Structured Errors** — богатая система ошибок с деталями и стеком вызовов
- **Type Safety** — максимальное использование дженериков для типобезопасности
- **Concurrency** — безопасная работа в многопоточной среде
- **Testing** — встроенные моки и тестовые утилиты

## 🚀 Быстрый старт

### Установка

```bash
go get github.com/shuldan/framework
```

### Простое приложение

```go
package main

import (
    "log"
    "time"
    
    "github.com/shuldan/framework/pkg/app"
    "github.com/shuldan/framework/pkg/cli"
    "github.com/shuldan/framework/pkg/logger"
)

func main() {
    // Создаём приложение
    application := app.New(
        app.AppInfo{
            AppName:     "MyApp",
            Version:     "1.0.0",
            Environment: "development",
        },
        nil, nil,
        app.WithGracefulTimeout(time.Second*10),
    )

    // Регистрируем модули
    if err := application.Register(logger.NewModule()); err != nil {
        log.Fatal(err)
    }
    
    if err := application.Register(cli.NewModule()); err != nil {
        log.Fatal(err)
    }

    // Запускаем приложение
    if err := application.Run(); err != nil {
        log.Fatal(err)
    }
}
```

## 📚 Подробное руководство

### Система модулей

Каждый модуль реализует интерфейс `AppModule`:

```go
type AppModule interface {
    Name() string
    Register(container DIContainer) error  // Регистрация зависимостей
    Start(ctx AppContext) error            // Запуск сервисов
    Stop(ctx AppContext) error             // Остановка сервисов
}
```

### CLI команды

```go
type MyCommand struct {
    verbose bool
}

func (c *MyCommand) Name() string { return "hello" }
func (c *MyCommand) Description() string { return "Say hello" }
func (c *MyCommand) Group() string { return "examples" }

func (c *MyCommand) Configure(flags *flag.FlagSet) {
    flags.BoolVar(&c.verbose, "verbose", false, "Verbose output")
}

func (c *MyCommand) Validate(ctx contracts.CliContext) error {
    return nil // валидация аргументов
}

func (c *MyCommand) Execute(ctx contracts.CliContext) error {
    fmt.Fprintln(ctx.Output(), "Hello, World!")
    return nil
}
```

### Конфигурация

```go
// Загрузка из нескольких источников с приоритетами
loader := config.NewChainLoader(
    config.NewYamlConfigLoader("config.yaml", "config.dev.yaml"),
    config.NewEnvConfigLoader("APP_"),
)

cfg := config.NewMapConfig(loader.Load())

// Типизированное получение значений
port := cfg.GetInt("server.port", 8080)
dbUrl := cfg.GetString("database.url")
features := cfg.GetStringSlice("features.enabled")
```

### База данных

```go
// Подключение
db := database.NewDatabase("postgres", dsn,
    database.WithConnectionPool(25, 5, time.Hour),
    database.WithRetry(3, time.Second),
)

// Миграции
migration := database.CreateMigration("001", "create users").
    CreateTable("users",
        "id SERIAL PRIMARY KEY",
        "name VARCHAR(255) NOT NULL",
        "email VARCHAR(255) UNIQUE",
    ).
    CreateIndex("idx_users_email", "users", "email").
    Build()

// Репозитории с разными стратегиями загрузки
repo := database.NewStrategyRepository[User, database.UUID, UserMemento](
    db, mapper, contracts.LoadingStrategyJoin,
)

users := repo.WithStrategy(contracts.LoadingStrategyBatch).
    FindAll(ctx, 100, 0)
```

### События

```go
// Событие
type UserCreated struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
}

// Слушатель
func handleUserCreated(ctx context.Context, event UserCreated) error {
    fmt.Printf("User created: %s\n", event.Email)
    return nil
}

// Подписка и публикация
bus.Subscribe((*UserCreated)(nil), handleUserCreated)
bus.Publish(ctx, UserCreated{UserID: "123", Email: "user@example.com"})
```

### Очереди

```go
type EmailJob struct {
    To      string `json:"to"`
    Subject string `json:"subject"`
    Body    string `json:"body"`
}

// Создание очереди
queue, err := queue.New[*EmailJob](broker,
    queue.WithConcurrency(5),
    queue.WithMaxRetries(3),
    queue.WithBackoff(queue.ExponentialBackoff{
        Base: time.Second, MaxDelay: time.Minute,
    }),
    queue.WithDLQ(true),
)

// Производство
queue.Produce(ctx, &EmailJob{
    To: "user@example.com", 
    Subject: "Welcome!",
    Body: "Welcome to our service!",
})

// Потребление
queue.Consume(ctx, func(ctx context.Context, job *EmailJob) error {
    return sendEmail(job.To, job.Subject, job.Body)
})
```

## 🏗️ Архитектура

```
┌─────────────────────────────────────────────────────────────────┐
│                            App                                  │
│  ┌─────────────────┐    ┌──────────────┐    ┌─────────────────┐ │
│  │   Registry      │    │  Container   │    │   Context       │ │
│  │ (модули)        │    │ (DI)         │    │ (жизненный цикл)│ │
│  └─────────────────┘    └──────────────┘    └─────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
            │                      │                      │
    ┌───────▼──────┐      ┌────────▼────────┐    ┌────────▼────────┐
    │ CLI Module   │      │ Logger Module   │    │ Config Module   │
    │ - Commands   │      │ - Structured    │    │ - Multi-source  │
    │ - Help       │      │ - Levels        │    │ - Type-safe     │
    │ - Validation │      │ - Colors        │    │ - Hierarchical  │
    └──────────────┘      └─────────────────┘    └─────────────────┘
            │                      │                      │
    ┌───────▼──────┐      ┌────────▼────────┐    ┌────────▼────────┐
    │Events Module │      │ Queue Module    │    │Database Module  │
    │ - Pub/Sub    │      │ - Job queues    │    │ - Migrations    │
    │ - Async      │      │ - Retry/DLQ     │    │ - Repositories  │
    │ - Error safe │      │ - Metrics       │    │ - Query builder │
    └──────────────┘      └─────────────────┘    └─────────────────┘
```

## 🧪 Тестирование

```bash
# Запуск всех тестов
make test

# С покрытием
make test-coverage

# Только линтер
make lint

# Форматирование
make fmt

# Полная CI проверка
make ci
```

## 📊 Метрики качества

- **Test Coverage**: >80%
- **Go Report**: A+
- **Cyclomatic Complexity**: <10
- **Maintainability Index**: >70

## 🎯 Roadmap

- [ ] HTTP модуль с middleware
- [ ] Metrics & Monitoring (Prometheus)
- [ ] Distributed tracing
- [ ] GraphQL поддержка
- [ ] gRPC интеграция
- [ ] WebSocket поддержка

## 🤝 Участие в разработке

1. Fork репозитория
2. Создайте feature branch (`git checkout -b feature/amazing-feature`)
3. Commit изменения (`git commit -m 'Add amazing feature'`)
4. Push в branch (`git push origin feature/amazing-feature`)
5. Откройте Pull Request

### Требования к коду

- Покрытие тестами >80%
- Проходит все линтеры
- Следует Go conventions
- Документирован публичный API

## 📄 Лицензия

Этот проект лицензирован под MIT License - подробности в файле [LICENSE](LICENSE).
