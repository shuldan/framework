package contracts

import (
	"context"
	"database/sql"
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

type Migration interface {
	ID() string
	ConnectionName() string
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

type MigrationsProvider interface {
	Migrations() []Migration
}

type Database interface {
	Connect() error
	Close() error
	Ping(ctx context.Context) error
	Migrate(migrations []Migration) error
	Rollback(steps int, migrations []Migration) error
	Status() ([]MigrationStatus, error)
	Connection() *sql.DB
}

type Finder[T Aggregate, I ID] interface {
	Find(ctx context.Context, id I) (T, error)
	FindAll(ctx context.Context, limit, offset int) ([]T, error)
	FindBy(ctx context.Context, conditions string, args []any) ([]T, error)
	ExistsBy(ctx context.Context, conditions string, args []any) (bool, error)
	CountBy(ctx context.Context, conditions string, args []any) (int64, error)
}

type Writer[T Aggregate, I ID] interface {
	Save(ctx context.Context, aggregate T) error
	Delete(ctx context.Context, id I) error
}

type Repository[T Aggregate, I ID] interface {
	Finder[T, I]
	Writer[T, I]
}
