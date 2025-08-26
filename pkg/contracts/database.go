package contracts

import (
	"context"
	"time"
)

type LoadingStrategy string

const (
	LoadingStrategyMultiple LoadingStrategy = "multiple"
	LoadingStrategyJoin     LoadingStrategy = "join"
	LoadingStrategyBatch    LoadingStrategy = "batch"
)

type ID interface {
	String() string
	IsValid() bool
}

type Aggregate interface {
	ID() ID
}

type Memento interface {
	GetID() ID
}

type Transaction interface {
	Commit() error
	Rollback() error
}

type Database interface {
	Connect() error
	Close() error
	Ping(ctx context.Context) error
	Migrate(migrations []Migration) error
	GetMigrationRunner() MigrationRunner
	BeginTx(ctx context.Context) (Transaction, error)
}

type Finder[T Aggregate, I ID] interface {
	Find(ctx context.Context, id I) (T, error)
	FindAll(ctx context.Context, limit, offset int) ([]T, error)
	FindBy(ctx context.Context, criteria map[string]interface{}) ([]T, error)
	Exists(ctx context.Context, id I) (bool, error)
	Count(ctx context.Context, criteria map[string]interface{}) (int64, error)
}

type Writer[T Aggregate, I ID] interface {
	Save(ctx context.Context, aggregate T) error
	Delete(ctx context.Context, id I) error
	DeleteBy(ctx context.Context, criteria map[string]interface{}) (int64, error)
}

type Repository[T Aggregate, I ID] interface {
	Finder[T, I]
	Writer[T, I]
}

type TransactionalRepository[T Aggregate, I ID] interface {
	Repository[T, I]
	WithTx(tx Transaction) Repository[T, I]
}

type StrategyRepository[T Aggregate, I ID] interface {
	TransactionalRepository[T, I]
	WithStrategy(strategy LoadingStrategy) Repository[T, I]
	GetStrategy() LoadingStrategy
}

type Migration interface {
	ID() string
	Description() string
	Up() []string
	Down() []string
}

type MigrationStatus struct {
	ID          string
	Description string
	AppliedAt   *time.Time
	Batch       int
}

type MigrationRunner interface {
	Run(migrations []Migration) error
	Rollback(steps int, migrations []Migration) error
	Status() ([]MigrationStatus, error)
	CreateMigrationTable() error
}
