package database

import (
	"context"
	"database/sql"
	"github.com/shuldan/framework/pkg/contracts"
)

type AggregateMapper[T contracts.Aggregate, M contracts.Memento] interface {
	TableName() string
	IDColumn() string
	CreateMemento(aggregate T) M
	RestoreAggregate(memento M) T
	ToColumns(memento M) (columns []string, values []interface{})
	FromRow(row *sql.Row) (M, error)
	FromRows(rows *sql.Rows) (M, error)
}

type StrategyMapper[T contracts.Aggregate, M contracts.Memento] interface {
	AggregateMapper[T, M]

	FindByIDMultiple(ctx context.Context, db QueryExecutor, id contracts.ID) (M, error)
	FindByIDJoin(ctx context.Context, db QueryExecutor, id contracts.ID) (M, error)
	FindByIDBatch(ctx context.Context, db QueryExecutor, ids []contracts.ID) ([]M, error)

	FindAllMultiple(ctx context.Context, db QueryExecutor, limit, offset int) ([]M, error)
	FindAllJoin(ctx context.Context, db QueryExecutor, limit, offset int) ([]M, error)
	FindAllBatch(ctx context.Context, db QueryExecutor, limit, offset int) ([]M, error)

	FindByMultiple(ctx context.Context, db QueryExecutor, criteria map[string]interface{}) ([]M, error)
	FindByJoin(ctx context.Context, db QueryExecutor, criteria map[string]interface{}) ([]M, error)
	FindByBatch(ctx context.Context, db QueryExecutor, criteria map[string]interface{}) ([]M, error)

	SaveWithRelations(ctx context.Context, db QueryExecutor, memento M, isUpdate bool) error
	DeleteWithRelations(ctx context.Context, db QueryExecutor, id contracts.ID) error
}
