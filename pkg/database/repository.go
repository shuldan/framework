package database

import (
	"context"
	"database/sql"

	"github.com/shuldan/framework/pkg/contracts"
	"github.com/shuldan/framework/pkg/errors"
)

type repository[T contracts.Aggregate, I contracts.ID, M contracts.Memento] struct {
	db     *sql.DB
	mapper Mapper[T, M]
}

func NewRepository[T contracts.Aggregate, I contracts.ID, M contracts.Memento](
	db *sql.DB,
	mapper Mapper[T, M],
) contracts.Repository[T, I] {
	return &repository[T, I, M]{
		db:     db,
		mapper: mapper,
	}
}

func (r repository[T, I, M]) Find(ctx context.Context, id I) (T, error) {
	var aggregate T
	row := r.mapper.Find(ctx, r.db, id)
	memento, err := r.mapper.FromRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return aggregate, ErrEntityNotFound
		}
		return aggregate, err
	}
	return r.mapper.FromMemento(memento)
}

func (r repository[T, I, M]) FindAll(ctx context.Context, limit, offset int) ([]T, error) {
	rows, err := r.mapper.FindAll(ctx, r.db, limit, offset)
	if err != nil {
		return nil, err
	}

	return r.rowsToAggregates(rows)
}

func (r repository[T, I, M]) FindBy(ctx context.Context, conditions string, args []any) ([]T, error) {
	rows, err := r.mapper.FindBy(ctx, r.db, conditions, args)
	if err != nil {
		return nil, err
	}
	return r.rowsToAggregates(rows)
}

func (r repository[T, I, M]) ExistsBy(ctx context.Context, conditions string, args []any) (bool, error) {
	exists, err := r.mapper.ExistsBy(ctx, r.db, conditions, args)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r repository[T, I, M]) CountBy(ctx context.Context, conditions string, args []any) (int64, error) {
	count, err := r.mapper.CountBy(ctx, r.db, conditions, args)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r repository[T, I, M]) Save(ctx context.Context, aggregate T) error {
	memento, err := r.mapper.ToMemento(aggregate)
	if err != nil {
		return err
	}
	return r.mapper.Save(ctx, r.db, memento)
}

func (r repository[T, I, M]) Delete(ctx context.Context, id I) error {
	return r.mapper.Delete(ctx, r.db, id)
}

func (r repository[T, I, M]) rowsToAggregates(rows *sql.Rows) ([]T, error) {
	mementos, err := r.mapper.FromRows(rows)
	if err != nil {
		return nil, err
	}

	aggregates := make([]T, len(mementos))
	for i, memento := range mementos {
		aggregate, err := r.mapper.FromMemento(memento)
		if err != nil {
			return nil, err
		}
		aggregates[i] = aggregate
	}

	return aggregates, nil
}
