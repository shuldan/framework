package database

import (
	"context"
	"database/sql"
)

type QueryExecutor interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

var _ QueryExecutor = (*sql.DB)(nil)
var _ QueryExecutor = (*sql.Tx)(nil)
