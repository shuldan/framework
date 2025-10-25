package database

import (
	"context"
	"database/sql"
	"sync"
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
}

type sqlDatabase struct {
	db              *sql.DB
	driver          string
	dsn             string
	migrationRunner MigrationRunner
	config          dbConfig
	mu              sync.Mutex
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
	d.mu.Lock()
	defer d.mu.Unlock()

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
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.db.Close()
}

func (d *sqlDatabase) Ping(ctx context.Context) error {
	if d.db == nil {
		return ErrDatabaseNotConnected
	}
	d.mu.Lock()
	db := d.db
	d.mu.Unlock()

	return db.PingContext(ctx)
}

func (d *sqlDatabase) Migrate(migrations []contracts.Migration) error {
	if d.db == nil {
		return ErrDatabaseNotConnected
	}

	d.mu.Lock()
	runner := d.migrationRunner
	d.mu.Unlock()

	return runner.Migrate(migrations)
}

func (d *sqlDatabase) Rollback(steps int, migrations []contracts.Migration) error {
	if d.db == nil {
		return ErrDatabaseNotConnected
	}

	d.mu.Lock()
	runner := d.migrationRunner
	d.mu.Unlock()

	return runner.Rollback(steps, migrations)
}

func (d *sqlDatabase) Status() ([]contracts.MigrationStatus, error) {
	if d.db == nil {
		return nil, ErrDatabaseNotConnected
	}

	d.mu.Lock()
	runner := d.migrationRunner
	d.mu.Unlock()

	return runner.Status()
}

func (d *sqlDatabase) Connection() *sql.DB {
	return d.db
}
