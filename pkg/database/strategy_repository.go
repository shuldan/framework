package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/shuldan/framework/pkg/contracts"
)

type strategyRepository[T contracts.Aggregate, I contracts.ID, M contracts.Memento] struct {
	db        *sql.DB
	tx        *sql.Tx
	tableName string
	mapper    StrategyMapper[T, M]
	strategy  contracts.LoadingStrategy
}

func NewStrategyRepository[T contracts.Aggregate, I contracts.ID, M contracts.Memento](
	db *sql.DB,
	mapper StrategyMapper[T, M],
	defaultStrategy contracts.LoadingStrategy,
) contracts.StrategyRepository[T, I] {
	return &strategyRepository[T, I, M]{
		db:        db,
		tableName: mapper.TableName(),
		mapper:    mapper,
		strategy:  defaultStrategy,
	}
}

func (r *strategyRepository[T, I, M]) WithStrategy(strategy contracts.LoadingStrategy) contracts.Repository[T, I] {
	return &strategyRepository[T, I, M]{
		db:        r.db,
		tx:        r.tx,
		tableName: r.tableName,
		mapper:    r.mapper,
		strategy:  strategy,
	}
}

func (r *strategyRepository[T, I, M]) GetStrategy() contracts.LoadingStrategy {
	return r.strategy
}

func (r *strategyRepository[T, I, M]) getExecutor() QueryExecutor {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *strategyRepository[T, I, M]) Find(ctx context.Context, id I) (T, error) {
	var zero T
	var memento M
	var err error

	executor := r.getExecutor()

	switch r.strategy {
	case contracts.LoadingStrategyMultiple:
		memento, err = r.mapper.FindByIDMultiple(ctx, executor, id)
	case contracts.LoadingStrategyJoin:
		memento, err = r.mapper.FindByIDJoin(ctx, executor, id)
	case contracts.LoadingStrategyBatch:
		results, batchErr := r.mapper.FindByIDBatch(ctx, executor, []contracts.ID{id})
		switch {
		case batchErr != nil:
			err = batchErr
		case len(results) == 0:
			err = sql.ErrNoRows
		default:
			memento = results[0]
		}
	default:
		memento, err = r.mapper.FindByIDMultiple(ctx, executor, id)
	}

	if errors.Is(err, sql.ErrNoRows) {
		return zero, ErrEntityNotFound.WithDetail("id", id.String())
	}
	if err != nil {
		return zero, err
	}

	return r.mapper.RestoreAggregate(memento)
}

func (r *strategyRepository[T, I, M]) FindAll(ctx context.Context, limit, offset int) ([]T, error) {
	var mementos []M
	var err error

	executor := r.getExecutor()

	switch r.strategy {
	case contracts.LoadingStrategyMultiple:
		mementos, err = r.mapper.FindAllMultiple(ctx, executor, limit, offset)
	case contracts.LoadingStrategyJoin:
		mementos, err = r.mapper.FindAllJoin(ctx, executor, limit, offset)
	case contracts.LoadingStrategyBatch:
		mementos, err = r.mapper.FindAllBatch(ctx, executor, limit, offset)
	default:
		mementos, err = r.mapper.FindAllMultiple(ctx, executor, limit, offset)
	}

	if err != nil {
		return nil, err
	}

	aggregates := make([]T, len(mementos))
	for i, memento := range mementos {
		aggregate, err := r.mapper.RestoreAggregate(memento)
		if err != nil {
			return nil, err
		}

		aggregates[i] = aggregate
	}

	return aggregates, nil
}

func (r *strategyRepository[T, I, M]) FindBy(ctx context.Context, criteria map[string]interface{}) ([]T, error) {
	var mementos []M
	var err error

	executor := r.getExecutor()

	switch r.strategy {
	case contracts.LoadingStrategyMultiple:
		mementos, err = r.mapper.FindByMultiple(ctx, executor, criteria)
	case contracts.LoadingStrategyJoin:
		mementos, err = r.mapper.FindByJoin(ctx, executor, criteria)
	case contracts.LoadingStrategyBatch:
		mementos, err = r.mapper.FindByBatch(ctx, executor, criteria)
	default:
		mementos, err = r.mapper.FindByMultiple(ctx, executor, criteria)
	}

	if err != nil {
		return nil, err
	}

	aggregates := make([]T, len(mementos))
	for i, memento := range mementos {
		aggregate, err := r.mapper.RestoreAggregate(memento)
		if err != nil {
			return nil, err
		}

		aggregates[i] = aggregate
	}

	return aggregates, nil
}

func (r *strategyRepository[T, I, M]) Exists(ctx context.Context, id I) (bool, error) {
	query := fmt.Sprintf("SELECT 1 FROM %s WHERE %s = ? LIMIT 1",
		r.tableName, r.mapper.IDColumn())

	var exists int
	executor := r.getExecutor()
	err := executor.QueryRowContext(ctx, query, id.String()).Scan(&exists)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *strategyRepository[T, I, M]) Count(ctx context.Context, criteria map[string]interface{}) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", r.tableName)
	args := make([]interface{}, len(criteria))

	if len(criteria) > 0 {
		conditions := make([]string, len(criteria))
		i := 0
		for field, value := range criteria {
			if err := ValidateColumnName(field); err != nil {
				return 0, err
			}
			conditions[i] = fmt.Sprintf("%s = ?", field)
			args[i] = value
			i++
		}
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int64
	executor := r.getExecutor()
	err := executor.QueryRowContext(ctx, query, args...).Scan(&count)

	return count, err
}

func (r *strategyRepository[T, I, M]) Save(ctx context.Context, aggregate T) error {
	memento, err := r.mapper.CreateMemento(aggregate)
	if err != nil {
		return err
	}

	id := memento.GetID()

	if !id.IsValid() {
		return ErrInvalidID.WithDetail("id", id.String())
	}

	exists, err := r.Exists(ctx, id.(I))
	if err != nil {
		return err
	}

	return r.mapper.SaveWithRelations(ctx, r.getExecutor(), memento, exists)
}

func (r *strategyRepository[T, I, M]) Delete(ctx context.Context, id I) error {
	exists, err := r.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrEntityNotFound.WithDetail("id", id.String())
	}

	return r.mapper.DeleteWithRelations(ctx, r.getExecutor(), id)
}

func (r *strategyRepository[T, I, M]) DeleteBy(ctx context.Context, criteria map[string]interface{}) (int64, error) {
	if len(criteria) == 0 {
		return 0, ErrInvalidCriteria.WithDetail("reason", "empty criteria")
	}

	conditions := make([]string, len(criteria))
	args := make([]interface{}, len(criteria))

	i := 0
	for field, value := range criteria {
		if err := ValidateColumnName(field); err != nil {
			return 0, err
		}
		conditions[i] = fmt.Sprintf("%s = ?", field)
		args[i] = value
		i++
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s",
		r.tableName, strings.Join(conditions, " AND "))

	executor := r.getExecutor()
	result, err := executor.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (r *strategyRepository[T, I, M]) WithTx(tx contracts.Transaction) contracts.Repository[T, I] {
	provider, ok := tx.(*sqlTransaction)
	if !ok {
		return r
	}
	sqlTx := provider.getConnection()

	return &strategyRepository[T, I, M]{
		db:        r.db,
		tx:        sqlTx,
		tableName: r.tableName,
		mapper:    r.mapper,
		strategy:  r.strategy,
	}
}
