package migration

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/shuldan/migrator"

	"github.com/shuldan/framework/database"
)

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

type StatusResult struct {
	Connection string
	Migrations []migrator.MigrationStatus
}

type PlanResult struct {
	Connection string
	Migrations []migrator.PlannedMigration
}

type Runner struct {
	dbm       *database.Manager
	logger    Logger
	sets      map[string][]migrator.Migration
	tableName string
	lock      bool
}

func NewRunner(
	dbm *database.Manager, log Logger, opts ...RunnerOption,
) *Runner {
	r := &Runner{
		dbm:    dbm,
		logger: ensureLog(log),
		sets:   make(map[string][]migrator.Migration),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func (r *Runner) Register(
	connection string, migrations ...migrator.Migration,
) {
	r.sets[connection] = append(
		r.sets[connection], migrations...,
	)
}

func (r *Runner) ConnectionNames() []string {
	names := make([]string, 0, len(r.sets))
	for name := range r.sets {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

func (r *Runner) Up(ctx context.Context, connection string) error {
	return r.forEachTarget(connection, func(name string) error {
		m, err := r.buildMigrator(name)
		if err != nil {
			return err
		}

		r.logger.Info("running migrations up", "connection", name)

		return m.Up(ctx)
	})
}

func (r *Runner) Down(
	ctx context.Context,
	connection string,
	steps int,
	force bool,
) error {
	return r.forEachTarget(connection, func(name string) error {
		m, err := r.buildMigrator(name)
		if err != nil {
			return err
		}

		var opts []migrator.DownOption
		if force {
			opts = append(opts, migrator.WithForce())
		}

		r.logger.Info("rolling back migrations",
			"connection", name, "steps", steps,
		)

		return m.Down(ctx, steps, opts...)
	})
}

func (r *Runner) Status(
	ctx context.Context, connection string,
) ([]StatusResult, error) {
	targets := r.targets(connection)
	results := make([]StatusResult, 0, len(targets))

	for _, name := range targets {
		m, err := r.buildMigrator(name)
		if err != nil {
			return nil, err
		}

		statuses, err := m.Status(ctx)
		if err != nil {
			return nil, fmt.Errorf("connection %q: %w", name, err)
		}

		results = append(results, StatusResult{
			Connection: name,
			Migrations: statuses,
		})
	}

	return results, nil
}

func (r *Runner) Plan(
	ctx context.Context, connection string,
) ([]PlanResult, error) {
	targets := r.targets(connection)
	results := make([]PlanResult, 0, len(targets))

	for _, name := range targets {
		m, err := r.buildMigrator(name)
		if err != nil {
			return nil, err
		}

		planned, err := m.Plan(ctx)
		if err != nil {
			return nil, fmt.Errorf("connection %q: %w", name, err)
		}

		results = append(results, PlanResult{
			Connection: name,
			Migrations: planned,
		})
	}

	return results, nil
}

func (r *Runner) buildMigrator(
	connName string,
) (*migrator.Migrator, error) {
	if !r.dbm.Has(connName) {
		return nil, fmt.Errorf(
			"%w: %s", database.ErrConnectionNotFound, connName,
		)
	}

	db := r.dbm.Connection(connName)
	m := migrator.New(db, r.migratorOpts(connName)...)

	if migrations, ok := r.sets[connName]; ok {
		if err := m.Register(migrations...); err != nil {
			return nil, fmt.Errorf(
				"connection %q: %w", connName, err,
			)
		}
	}

	return m, nil
}

func (r *Runner) migratorOpts(
	connName string,
) []migrator.Option {
	driver := r.dbm.Driver(connName)
	dialect := driverToDialect(driver)

	opts := []migrator.Option{
		migrator.WithDialect(dialect),
		migrator.WithLogger(r.logger),
	}

	if r.tableName != "" {
		opts = append(opts, migrator.WithTableName(r.tableName))
	}

	if r.lock {
		opts = append(opts, migrator.WithAdvisoryLock())
	}

	return opts
}

func (r *Runner) targets(connection string) []string {
	if connection != "" {
		return []string{connection}
	}

	return r.ConnectionNames()
}

func (r *Runner) forEachTarget(
	connection string, fn func(string) error,
) error {
	for _, name := range r.targets(connection) {
		if err := fn(name); err != nil {
			return fmt.Errorf("connection %q: %w", name, err)
		}
	}

	return nil
}

func driverToDialect(driver string) migrator.Dialect {
	d := strings.ToLower(driver)

	switch {
	case strings.Contains(d, "postgres"),
		strings.Contains(d, "pgx"):
		return migrator.DialectPostgreSQL
	case strings.Contains(d, "mysql"):
		return migrator.DialectMySQL
	case strings.Contains(d, "sqlite"):
		return migrator.DialectSQLite
	default:
		return migrator.DialectUnknown
	}
}

func ensureLog(log Logger) Logger {
	if log == nil {
		return noopLogger{}
	}

	return log
}
