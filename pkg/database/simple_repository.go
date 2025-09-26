package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/shuldan/framework/pkg/contracts"
)

type simpleRepository[T contracts.Aggregate, I contracts.ID, M contracts.Memento] struct {
	db        *sql.DB
	tx        *sql.Tx
	tableName string
	mapper    AggregateMapper[T, M]
}

func NewSimpleRepository[T contracts.Aggregate, I contracts.ID, M contracts.Memento](
	db *sql.DB,
	mapper AggregateMapper[T, M],
) contracts.TransactionalRepository[T, I] {
	return &simpleRepository[T, I, M]{
		db:        db,
		tableName: mapper.TableName(),
		mapper:    mapper,
	}
}

func (r *simpleRepository[T, I, M]) getExecutor() QueryExecutor {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *simpleRepository[T, I, M]) Find(ctx context.Context, id I) (T, error) {
	var zero T

	columns := r.mapper.GetColumns()
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?",
		strings.Join(columns, ", "), r.tableName, r.mapper.IDColumn())

	executor := r.getExecutor()
	row := executor.QueryRowContext(ctx, query, id.String())

	memento, err := r.mapper.FromRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return zero, ErrEntityNotFound.WithDetail("id", id.String())
	}
	if err != nil {
		return zero, err
	}

	return r.mapper.RestoreAggregate(memento), nil
}

func (r *simpleRepository[T, I, M]) FindAll(ctx context.Context, limit, offset int) ([]T, error) {
	columns := r.mapper.GetColumns()
	query := fmt.Sprintf("SELECT %s FROM %s LIMIT ? OFFSET ?",
		strings.Join(columns, ", "), r.tableName)

	executor := r.getExecutor()
	rows, err := executor.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}

	defer func() {
		if rows != nil {
			_ = rows.Close()
			// TODO: log error
		}
	}()

	var aggregates []T
	for rows.Next() {
		memento, err := r.mapper.FromRows(rows)
		if err != nil {
			return nil, err
		}
		aggregate := r.mapper.RestoreAggregate(memento)
		aggregates = append(aggregates, aggregate)
	}

	return aggregates, rows.Err()
}

func (r *simpleRepository[T, I, M]) FindBy(ctx context.Context, criteria map[string]interface{}) ([]T, error) {
	if len(criteria) == 0 {
		return nil, ErrInvalidCriteria.WithDetail("reason", "empty criteria not allowed")
	}

	conditions := make([]string, len(criteria))
	args := make([]interface{}, len(criteria))

	i := 0
	for field, value := range criteria {
		if err := validateColumnName(field); err != nil {
			return nil, err
		}
		conditions[i] = fmt.Sprintf("%s = ?", field)
		args[i] = value
		i++
	}

	columns := r.mapper.GetColumns()
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s",
		strings.Join(columns, ", "), r.tableName, strings.Join(conditions, " AND "))

	executor := r.getExecutor()
	rows, err := executor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer func() {
		if rows != nil {
			_ = rows.Close()
			// TODO: log error
		}
	}()

	var aggregates []T
	for rows.Next() {
		memento, err := r.mapper.FromRows(rows)
		if err != nil {
			return nil, err
		}
		aggregate := r.mapper.RestoreAggregate(memento)
		aggregates = append(aggregates, aggregate)
	}

	return aggregates, rows.Err()
}

func (r *simpleRepository[T, I, M]) Exists(ctx context.Context, id I) (bool, error) {
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

func (r *simpleRepository[T, I, M]) Count(ctx context.Context, criteria map[string]interface{}) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", r.tableName)
	args := make([]interface{}, len(criteria))

	if len(criteria) > 0 {
		conditions := make([]string, len(criteria))
		i := 0
		for field, value := range criteria {
			if err := validateColumnName(field); err != nil {
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

func (r *simpleRepository[T, I, M]) Save(ctx context.Context, aggregate T) error {
	memento := r.mapper.CreateMemento(aggregate)
	columns := r.mapper.GetColumns()
	values := r.mapper.GetValues(memento)
	id := memento.GetID()

	if !id.IsValid() {
		return ErrInvalidID.WithDetail("id", id.String())
	}

	if _, ok := id.(I); !ok {
		return ErrInvalidIDType.
			WithDetail("expected", (*I)(nil)).
			WithDetail("actual", id)
	}
	exists, err := r.Exists(ctx, id.(I))
	if err != nil {
		return err
	}

	executor := r.getExecutor()

	if !exists {
		placeholders := make([]string, len(columns))
		for i := range placeholders {
			placeholders[i] = "?"
		}

		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			r.tableName,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))

		_, err := executor.ExecContext(ctx, query, values...)
		return err
	} else {
		var setParts []string
		var updateValues []interface{}

		for i, col := range columns {
			if col != r.mapper.IDColumn() {
				setParts = append(setParts, fmt.Sprintf("%s = ?", col))
				updateValues = append(updateValues, values[i])
			}
		}
		updateValues = append(updateValues, id.String())

		query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?",
			r.tableName,
			strings.Join(setParts, ", "),
			r.mapper.IDColumn())

		_, err := executor.ExecContext(ctx, query, updateValues...)
		return err
	}
}

func (r *simpleRepository[T, I, M]) Delete(ctx context.Context, id I) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = ?",
		r.tableName, r.mapper.IDColumn())

	executor := r.getExecutor()
	result, err := executor.ExecContext(ctx, query, id.String())
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return ErrEntityNotFound.WithDetail("id", id.String())
	}

	return nil
}

func (r *simpleRepository[T, I, M]) DeleteBy(ctx context.Context, criteria map[string]interface{}) (int64, error) {
	if len(criteria) == 0 {
		return 0, ErrInvalidCriteria.WithDetail("reason", "empty criteria")
	}

	conditions := make([]string, len(criteria))
	args := make([]interface{}, len(criteria))

	i := 0
	for field, value := range criteria {
		if err := validateColumnName(field); err != nil {
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

type connectionProvider interface {
	getConnection() *sql.Tx
}

func (r *simpleRepository[T, I, M]) WithTx(tx contracts.Transaction) contracts.Repository[T, I] {
	if provider, ok := tx.(connectionProvider); ok {
		sqlTx := provider.getConnection()
		return &simpleRepository[T, I, M]{
			db:        r.db,
			tx:        sqlTx,
			tableName: r.tableName,
			mapper:    r.mapper,
		}
	}
	return r
}
