package database

import (
	"context"
	"database/sql"

	"github.com/shuldan/framework/pkg/contracts"
)

type Mapper[T contracts.Aggregate, M contracts.Memento] interface {
	Find(ctx context.Context, db *sql.DB, id contracts.ID) *sql.Row
	FindAll(ctx context.Context, db *sql.DB, limit, offset int) (*sql.Rows, error)
	FindBy(ctx context.Context, db *sql.DB, conditions string, args []any) (*sql.Rows, error)
	ExistsBy(ctx context.Context, db *sql.DB, conditions string, args []any) (bool, error)
	CountBy(ctx context.Context, db *sql.DB, conditions string, args []any) (int64, error)
	Save(ctx context.Context, db *sql.DB, memento M) error
	Delete(ctx context.Context, db *sql.DB, id contracts.ID) error
	ToMemento(aggregate T) (M, error)
	FromMemento(memento M) (T, error)
	FromRow(row *sql.Row) (M, error)
	FromRows(rows *sql.Rows) ([]M, error)
}
