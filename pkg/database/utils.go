package database

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type QueryBuilder struct {
	query   strings.Builder
	args    []interface{}
	counter int
}

func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		args:    make([]interface{}, 0),
		counter: 0,
	}
}

func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	qb.query.WriteString("SELECT ")
	qb.query.WriteString(strings.Join(columns, ", "))
	return qb
}

func (qb *QueryBuilder) From(table string) *QueryBuilder {
	qb.query.WriteString(" FROM ")
	qb.query.WriteString(table)
	return qb
}

func (qb *QueryBuilder) Join(table, condition string) *QueryBuilder {
	qb.query.WriteString(" JOIN ")
	qb.query.WriteString(table)
	qb.query.WriteString(" ON ")
	qb.query.WriteString(condition)
	return qb
}

func (qb *QueryBuilder) LeftJoin(table, condition string) *QueryBuilder {
	qb.query.WriteString(" LEFT JOIN ")
	qb.query.WriteString(table)
	qb.query.WriteString(" ON ")
	qb.query.WriteString(condition)
	return qb
}

func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	qb.query.WriteString(" WHERE ")
	qb.query.WriteString(condition)
	qb.args = append(qb.args, args...)
	return qb
}

func (qb *QueryBuilder) And(condition string, args ...interface{}) *QueryBuilder {
	qb.query.WriteString(" AND ")
	qb.query.WriteString(condition)
	qb.args = append(qb.args, args...)
	return qb
}

func (qb *QueryBuilder) Or(condition string, args ...interface{}) *QueryBuilder {
	qb.query.WriteString(" OR ")
	qb.query.WriteString(condition)
	qb.args = append(qb.args, args...)
	return qb
}

func (qb *QueryBuilder) OrderBy(column, direction string) *QueryBuilder {
	qb.query.WriteString(" ORDER BY ")
	qb.query.WriteString(column)
	qb.query.WriteString(" ")
	qb.query.WriteString(direction)
	return qb
}

func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.query.WriteString(" LIMIT ?")
	qb.args = append(qb.args, limit)
	return qb
}

func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.query.WriteString(" OFFSET ?")
	qb.args = append(qb.args, offset)
	return qb
}

func (qb *QueryBuilder) GroupBy(columns ...string) *QueryBuilder {
	qb.query.WriteString(" GROUP BY ")
	qb.query.WriteString(strings.Join(columns, ", "))
	return qb
}

func (qb *QueryBuilder) Having(condition string, args ...interface{}) *QueryBuilder {
	qb.query.WriteString(" HAVING ")
	qb.query.WriteString(condition)
	qb.args = append(qb.args, args...)
	return qb
}

func (qb *QueryBuilder) Build() (string, []interface{}) {
	return qb.query.String(), qb.args
}

func (qb *QueryBuilder) Reset() *QueryBuilder {
	qb.query.Reset()
	qb.args = qb.args[:0]
	qb.counter = 0
	return qb
}

type TransactionManager struct {
	db *sql.DB
}

func NewTransactionManager(db *sql.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

func (tm *TransactionManager) Execute(ctx context.Context, fn func(ctx context.Context, tx *sql.Tx) error) error {
	tx, err := tm.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrTransactionFailed.WithDetail("reason", "failed to begin").WithCause(err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
		}
	}()

	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return ErrTransactionFailed.
				WithDetail("reason", "failed to rollback after error").
				WithCause(fmt.Errorf("original: %w, rollback: %v", err, rbErr))
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return ErrTransactionFailed.WithDetail("reason", "failed to commit").WithCause(err)
	}

	return nil
}

type BatchProcessor[T any] struct {
	batchSize int
}

func NewBatchProcessor[T any](batchSize int) *BatchProcessor[T] {
	if batchSize <= 0 {
		batchSize = 100
	}
	return &BatchProcessor[T]{batchSize: batchSize}
}

func (bp *BatchProcessor[T]) Process(items []T, processor func(batch []T) error) error {
	for i := 0; i < len(items); i += bp.batchSize {
		end := i + bp.batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		if err := processor(batch); err != nil {
			return fmt.Errorf("batch processing failed at batch starting at index %d: %w", i, err)
		}
	}
	return nil
}

func ConfigureConnectionPool(db *sql.DB, maxOpen, maxIdle int, maxLifetime int) {
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(time.Duration(maxLifetime) * time.Second)
}

func ToNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func FromNullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func ToNullInt64(i int64) sql.NullInt64 {
	return sql.NullInt64{Int64: i, Valid: i != 0}
}

func FromNullInt64(ni sql.NullInt64) int64 {
	if ni.Valid {
		return ni.Int64
	}
	return 0
}

func ToNullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: !t.IsZero()}
}

func FromNullTime(nt sql.NullTime) time.Time {
	if nt.Valid {
		return nt.Time
	}
	return time.Time{}
}

var validColumnPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func validateColumnName(name string) error {
	if !validColumnPattern.MatchString(name) {
		return ErrInvalidCriteria.WithDetail("reason", "invalid column name: "+name)
	}
	return nil
}
