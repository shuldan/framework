package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

type dbConfig struct {
	maxOpenConns    int
	maxIdleConns    int
	connMaxLifetime time.Duration
	connMaxIdleTime time.Duration
	pingTimeout     time.Duration
	retryAttempts   int
	retryDelay      time.Duration
}

type Option func(*dbConfig)

func WithConnectionPool(maxOpen, maxIdle int, maxLifetime time.Duration) Option {
	return func(config *dbConfig) {
		config.maxOpenConns = maxOpen
		config.maxIdleConns = maxIdle
		config.connMaxLifetime = maxLifetime
	}
}

func WithConnectionIdleTime(idleTime time.Duration) Option {
	return func(config *dbConfig) {
		config.connMaxIdleTime = idleTime
	}
}

func WithPingTimeout(timeout time.Duration) Option {
	return func(config *dbConfig) {
		config.pingTimeout = timeout
	}
}

func WithRetry(attempts int, delay time.Duration) Option {
	return func(config *dbConfig) {
		config.retryAttempts = attempts
		config.retryDelay = delay
	}
}

type MigrationRunner interface {
	Migrate(migrations []contracts.Migration) error
	Rollback(steps int, migrations []contracts.Migration) error
	Status() ([]contracts.MigrationStatus, error)
	CreateMigrationTable() error
}

type sqlDatabase struct {
	db              *sql.DB
	driver          string
	dsn             string
	migrationRunner MigrationRunner
	config          dbConfig
}

func NewDatabase(driver, dsn string, options ...Option) contracts.Database {
	config := dbConfig{
		maxOpenConns:    25,
		maxIdleConns:    5,
		connMaxLifetime: time.Hour,
		connMaxIdleTime: time.Minute * 5,
		pingTimeout:     time.Second * 5,
		retryAttempts:   3,
		retryDelay:      time.Second,
	}

	for _, option := range options {
		option(&config)
	}

	return &sqlDatabase{
		driver: driver,
		dsn:    dsn,
		config: config,
	}
}

func (d *sqlDatabase) Connect() error {
	if d.db != nil {
		return nil
	}

	var db *sql.DB
	var err error

	for attempt := 0; attempt <= d.config.retryAttempts; attempt++ {
		db, err = sql.Open(d.driver, d.dsn)
		if err == nil {
			db.SetMaxOpenConns(d.config.maxOpenConns)
			db.SetMaxIdleConns(d.config.maxIdleConns)
			db.SetConnMaxLifetime(d.config.connMaxLifetime)
			db.SetConnMaxIdleTime(d.config.connMaxIdleTime)

			ctx, cancel := context.WithTimeout(context.Background(), d.config.pingTimeout)
			err = db.PingContext(ctx)
			cancel()

			if err == nil {
				d.db = db
				d.migrationRunner = newMigrationRunner(db)
				return nil
			}
		}

		if attempt < d.config.retryAttempts {
			time.Sleep(d.config.retryDelay)
		}
	}

	return ErrFailedToOpenDatabase.WithCause(err)
}

func (d *sqlDatabase) Close() error {
	if d.db == nil {
		return nil
	}
	return d.db.Close()
}

func (d *sqlDatabase) Ping(ctx context.Context) error {
	if d.db == nil {
		return ErrDatabaseNotConnected
	}
	return d.db.PingContext(ctx)
}

func (d *sqlDatabase) Migrate(migrations []contracts.Migration) error {
	if d.db == nil {
		return ErrDatabaseNotConnected
	}
	return d.migrationRunner.Migrate(migrations)
}

func (d *sqlDatabase) Rollback(steps int, migrations []contracts.Migration) error {
	if d.db == nil {
		return ErrDatabaseNotConnected
	}
	return d.migrationRunner.Rollback(steps, migrations)
}

func (d *sqlDatabase) Status() ([]contracts.MigrationStatus, error) {
	return d.migrationRunner.Status()
}

func (d *sqlDatabase) BeginTx(ctx context.Context) (contracts.Transaction, error) {
	if d.db == nil {
		return nil, ErrDatabaseNotConnected
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, ErrTransactionFailed.
			WithDetail("reason", err.Error()).
			WithCause(err)
	}

	return &sqlTransaction{tx: tx}, nil
}

type sqlTransaction struct {
	tx *sql.Tx
}

func (t *sqlTransaction) Commit() error {
	err := t.tx.Commit()
	if err != nil {
		return ErrTransactionFailed.
			WithDetail("reason", "commit failed").
			WithCause(err)
	}
	return nil
}

func (t *sqlTransaction) Rollback() error {
	err := t.tx.Rollback()
	if err != nil {
		return ErrTransactionFailed.
			WithDetail("reason", "rollback failed").
			WithCause(err)
	}
	return nil
}

func (t *sqlTransaction) getConnection() interface{} {
	return t.tx
}
