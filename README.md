# Shuldan Framework

[![Go CI](https://github.com/shuldan/framework/workflows/Go%20CI/badge.svg)](https://github.com/shuldan/framework/actions)
[![codecov](https://codecov.io/gh/shuldan/framework/branch/main/graph/badge.svg)](https://codecov.io/gh/shuldan/framework)
[![Go Report Card](https://goreportcard.com/badge/github.com/shuldan/framework)](https://goreportcard.com/report/github.com/shuldan/framework)
[![GoDoc](https://godoc.org/github.com/shuldan/framework?status.svg)](https://godoc.org/github.com/shuldan/framework)

> **Shuldan** — современный, легковесный и модульный фреймворк на Go для создания расширяемых приложений с поддержкой внедрения зависимостей (DI), жизненного цикла модулей, типобезопасных репозиториев, очередей, событий и логирования.

---

## 🧱 Архитектура

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

Shuldan построен на модульной архитектуре, где каждый компонент — это **модуль**, реализующий интерфейс `AppModule`. Это позволяет легко подключать, настраивать и расширять функциональность приложения.

```go
type AppModule interface {
    Name() string
    Register(container DIContainer) error
    Start(ctx AppContext) error
    Stop(ctx AppContext) error
}
```

---

## 🔧 Основные компоненты

---

### 1. **App — Ядро приложения**

#### 📌 Назначение
Центральный компонент, управляющий жизненным циклом приложения: регистрация модулей, запуск, остановка, graceful shutdown.

#### 🧩 Реализация

- **Структура**: `app struct`
    - `container`: DI-контейнер
    - `registry`: реестр модулей
    - `info`: метаинформация (имя, версия, окружение)
    - `appCtx`: контекст приложения
    - `shutdownTimeout`: таймаут для graceful shutdown

- **Интерфейсы**:
  ```go
  type App interface {
      Register(module AppModule) error
      Run() error
  }
  ```

#### 💡 Пример использования

```go
application := app.New(
    app.AppInfo{
        AppName:     "MyApp",
        Version:     "1.0.0",
        Environment: "development",
    },
    nil, nil,
    app.WithGracefulTimeout(10*time.Second),
)

if err := application.Register(logger.NewModule()); err != nil {
    log.Fatal(err)
}

if err := application.Run(); err != nil {
    log.Fatal(err)
}
```

#### ⚠️ Возможные ошибки
| Ошибка | Причина | Решение |
|-------|--------|--------|
| `ErrModuleRegistration` | Ошибка регистрации модуля | Проверьте, реализует ли модуль `AppModule` |
| `context.DeadlineExceeded` | Превышен таймаут остановки | Увеличьте `WithGracefulTimeout` |

#### 🛠️ Рекомендации
- Всегда используйте `WithGracefulTimeout` для корректного завершения.
- Запускайте `Run()` в `main()`, после регистрации всех модулей.

---

### 2. **DI Container — Внедрение зависимостей**

#### 📌 Назначение
Контейнер для управления зависимостями: регистрация фабрик, разрешение зависимостей, кэширование экземпляров.

#### 🧩 Реализация

- **Структура**: `container`
    - `factories`: `map[string]func(DIContainer) (interface{}, error)`
    - `instances`: кэшированные экземпляры
    - `resolving`: защита от циклических зависимостей

- **Интерфейс**:
  ```go
  type DIContainer interface {
      Has(name string) bool
      Instance(name string, value interface{}) error
      Factory(name string, factory func(DIContainer) (interface{}, error)) error
      Get(name string) (interface{}, error)
  }
  ```

#### 💡 Пример использования

```go
container.Factory("logger", func(c DIContainer) (interface{}, error) {
    return logger.NewLogger(), nil
})

logInstance, err := container.Get("logger")
```

#### ⚠️ Возможные ошибки
| Ошибка | Причина | Решение |
|-------|--------|--------|
| `ErrCircularDep` | Циклическая зависимость | Пересмотрите архитектуру модулей |
| `ErrValueNotFound` | Нет зарегистрированной фабрики | Проверьте имя и регистр |

#### 🛠️ Рекомендации
- Используйте осмысленные имена (`logger`, `db`, `event_bus`).
- Избегайте регистрации экземпляров напрямую — используйте `Factory`.

---

### 3. **Logger — Структурированное логирование**

#### 📌 Назначение
Гибкая система логирования с поддержкой уровней, атрибутов и структурированного вывода (JSON).

#### 🧩 Реализация

- **Модуль**: `logger.Module`
- **Интерфейс**:
  ```go
  type Logger interface {
      Trace(msg string, args ...any)
      Debug(msg string, args ...any)
      Info(msg string, args ...any)
      Warn(msg string, args ...any)
      Error(msg string, args ...any)
      Critical(msg string, args ...any)
      With(args ...any) Logger
  }
  ```

- Поддерживает `slog` из стандартной библиотеки.

#### 💡 Пример использования

```go
log := container.Get("logger").(logger.Logger)
log.Info("User logged in", "user_id", "123", "ip", "192.168.1.1")

// С добавлением контекста
scopedLog := log.With("service", "auth")
scopedLog.Error("Failed to authenticate", "error", err)
```

#### ⚠️ Возможные ошибки
| Ошибка | Причина | Решение |
|-------|--------|--------|
| `nil logger` | Логгер не зарегистрирован | Убедитесь, что `logger.NewModule()` добавлен |

#### 🛠️ Рекомендации
- Используйте `With()` для добавления постоянных атрибутов.
- Настройте уровень логирования в зависимости от окружения.

---

### 4. **CLI — Система команд**

#### 📌 Назначение
Поддержка CLI-приложений с командами, флагами, справкой и валидацией.

#### 🧩 Реализация

- **Компоненты**:
    - `CliCommand`: интерфейс команды
    - `CliRegistry`: реестр команд
    - `HelpCommand`: встроенная команда `help`

- **Интерфейс команды**:
  ```go
  type CliCommand interface {
      Name() string
      Description() string
      Group() string
      Configure(*flag.FlagSet)
      Validate(CliContext) error
      Execute(CliContext) error
  }
  ```

#### 💡 Пример использования

```go
type MyCommand struct{}

func (c *MyCommand) Name() string        { return "greet" }
func (c *MyCommand) Description() string { return "Say hello" }
func (c *MyCommand) Configure(f *flag.FlagSet) { f.String("name", "", "Name to greet") }
func (c *MyCommand) Execute(ctx CliContext) error {
    name := ctx.Flag("name").String()
    fmt.Fprintf(ctx.Output(), "Hello, %s!\n", name)
    return nil
}

// Регистрация
cliModule := cli.NewModule()
cliModule.Register(&MyCommand{})
```

#### ⚠️ Возможные ошибки
| Ошибка | Причина | Решение |
|-------|--------|--------|
| `ErrCommandExecution` | Ошибка выполнения | Проверьте `Execute()` |
| `ErrFlagParse` | Ошибка парсинга флагов | Убедитесь, что флаги объявлены корректно |

#### 🛠️ Рекомендации
- Используйте `HelpCommand` для автоматической генерации справки.
- Группируйте команды по функциональности (`system`, `db`, `user`).

---

### 5. **Database — Работа с БД**

#### 📌 Назначение
Поддержка подключения к БД, пулов соединений, миграций, репозиториев и транзакций.

#### 🧩 Реализация

- **Компоненты**:
    - `NewDatabase(dsn, opts...)`: настраиваемое подключение
    - `Migration`: DSL для миграций
    - `TransactionalRepository`: типобезопасный репозиторий

- **Опции**:
    - `WithConnectionPool(maxOpen, maxIdle, maxLifetime)`
    - `WithRetry(attempts, delay)`

#### 💡 Пример использования

```go
db := database.NewDatabase("postgres", dsn,
    database.WithConnectionPool(25, 5, time.Hour),
)

// Миграция
migration := database.CreateMigration("001").
    CreateTable("users", "id SERIAL PRIMARY KEY", "name TEXT").
    Build()

err := migration.Apply(db)
```

#### Репозиторий

```go
type User struct { /* ... */ }
type UserMemento struct { /* ... */ }

repo := database.NewSimpleRepository[User, database.UUID, UserMemento](db, userMapper)

user, err := repo.FindByID(ctx, id)
```

#### ⚠️ Возможные ошибки
| Ошибка | Причина | Решение |
|-------|--------|--------|
| `ErrFailedToPing` | Нет связи с БД | Проверьте DSN и доступность сервера |
| `ErrNoMigrationsToRollback` | Нечего откатывать | Убедитесь, что миграции применялись |

#### 🛠️ Рекомендации
- Всегда используйте `WithRetry` для отказоустойчивости.
- Храните миграции в отдельной директории.

---

### 6. **Config — Система конфигурации**

#### 📌 Назначение
Загрузка конфигурации из нескольких источников: YAML, env, JSON с приоритетами.

#### 🧩 Реализация

- **Лоадеры**:
    - `YamlConfigLoader`
    - `EnvConfigLoader`
    - `JSONConfigLoader`
    - `ChainLoader` — объединяет несколько источников

- **Интерфейс**:
  ```go
  type Loader interface {
      Load() (map[string]any, error)
  }
  ```

#### 💡 Пример использования

```go
loader := config.NewChainLoader(
    config.NewYamlConfigLoader("config.yaml"),
    config.NewEnvConfigLoader("APP_"),
)

cfg := config.NewMapConfig(loader.Load())

port := cfg.GetInt("server.port", 8080)
debug := cfg.GetBool("debug", false)
```

#### ⚠️ Возможные ошибки
| Ошибка | Причина | Решение |
|-------|--------|--------|
| `ErrParseYAML` | Ошибка парсинга YAML | Проверьте синтаксис |
| `ErrParseJSON` | Ошибка JSON | Используйте валидатор |

#### 🛠️ Рекомендации
- Используйте `ChainLoader` для override значений.
- Предпочитайте `env` для production.

---

### 7. **Events — Система событий (Pub/Sub)**

#### 📌 Назначение
Асинхронная передача событий между модулями с гарантией безопасности.

#### 🧩 Реализация

- **Интерфейс**:
  ```go
  type Bus interface {
      Subscribe(eventType any, listener any) error
      Publish(ctx context.Context, event any) error
      Close() error
  }
  ```

- Поддержка типизированных слушателей:
  ```go
  func handleUserCreated(ctx context.Context, event UserCreated) error
  ```

#### 💡 Пример использования

```go
bus.Subscribe((*UserCreated)(nil), handleUserCreated)
bus.Publish(ctx, UserCreated{UserID: "123", Email: "user@example.com"})
```

#### ⚠️ Возможные ошибки
| Ошибка | Причина | Решение |
|-------|--------|--------|
| `ErrInvalidListener` | Неверная сигнатура слушателя | Используйте `func(context.Context, T) error` |

#### 🛠️ Рекомендации
- Слушатели должны быть идемпотентными.
- Не выполняйте долгие операции в слушателях — используйте очереди.

---

### 8. **Queue — Очереди задач**

#### 📌 Назначение
Обработка фоновых задач с поддержкой retry, DLQ (Dead Letter Queue), метрик и Redis-брокера.

#### 🧩 Реализация

- **Брокер**: `redis.Broker`
- **Сообщение**: `IQueueMessage`
- **Обработчики ошибок и паник**

#### 💡 Пример использования

```go
type SendEmailJob struct {
    To      string `json:"to"`
    Subject string `json:"subject"`
}

err := broker.Produce(ctx, "email", job)
```

#### Обработчик

```go
broker.Subscribe("email", func(ctx context.Context, job SendEmailJob) error {
    return sendEmail(job.To, job.Subject)
})
```

#### ⚠️ Возможные ошибки
| Ошибка | Причина | Решение |
|-------|--------|--------|
| `ErrMarshal` | Ошибка сериализации | Убедитесь, что структура экспортируема |
| `ErrSendToDLQ` | Ошибка отправки в DLQ | Проверьте подключение к Redis |

#### 🛠️ Рекомендации
- Используйте `Retry` для временных сбоев.
- Настройте мониторинг DLQ.

---

### 9. **Errors — Структурированные ошибки**

#### 📌 Назначение
Богатая система ошибок с кодами, деталями, стеком и причинами.

#### 🧩 Реализация

- **Типы**:
    - `Code`: уникальный код ошибки
    - `Error`: расширенная ошибка с `Stack`, `Timestamp`, `Details`

- **Функции**:
  ```go
  errors.WithPrefix("AUTH")
  code.New("failed to login")
  err.WithDetail("user_id", "123").WithCause(originalErr)
  ```

#### 💡 Пример использования

```go
var ErrLoginFailed = errors.WithPrefix("AUTH").New("login failed")

func Login(user string) error {
    if !valid {
        return ErrLoginFailed.WithDetail("user", user)
    }
}
```

#### ⚠️ Возможные ошибки
| Ошибка | Причина | Решение |
|-------|--------|--------|
| `nil error` | Забыли вернуть ошибку | Используйте `errors.Is()` для проверки |

#### 🛠️ Рекомендации
- Назначайте уникальные префиксы (`AUTH`, `DB`, `QUEUE`).
- Всегда добавляйте контекст через `WithDetail()`.

---

### 10. **Testing — Встроенные тестовые утилиты**

#### 📌 Назначение
Упрощение тестирования модулей, DI, CLI, событий.

#### 🛠️ Рекомендации
- Используйте `mockModule` для тестирования жизненного цикла.
- Покрывайте тестами >80% (требование CI).
- Используйте `Makefile`:
  ```bash
  make test
  make test-coverage
  make lint
  make ci
  ```

---

## 🧪 CI/CD и инструменты

### Makefile

| Цель | Описание |
|------|---------|
| `fmt` | Форматирование кода |
| `lint` | Проверка стиля и ошибок |
| `test` | Запуск тестов |
| `test-coverage` | С покрытием |
| `ci` | Полная проверка (для CI) |
| `install-tools` | Установка `golangci-lint`, `gosec` и др. |

### GitHub Actions

- Автозапуск тестов, линтеров, security-сканирования (`gosec`)
- Отправка отчётов в Codecov
- Построение SARIF для GitHub Security

---

## 📊 Метрики качества

| Показатель | Требование |
|-----------|-----------|
| Test Coverage | >80% |
| Go Report | A+ |
| Cyclomatic Complexity | <10 |
| Maintainability Index | >70 |


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
