package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
)

type Manager struct {
	conns   map[string]*sql.DB
	drivers map[string]string
	order   []string
	logger  Logger
}

func NewManager(
	configs map[string]ConnectionConfig, log Logger,
) (*Manager, error) {
	if len(configs) == 0 {
		return nil, ErrNoConnections
	}

	m := &Manager{
		conns:   make(map[string]*sql.DB, len(configs)),
		drivers: make(map[string]string, len(configs)),
		order:   make([]string, 0, len(configs)),
		logger:  ensureLogger(log),
	}

	if err := m.openAll(configs); err != nil {
		_ = m.closeAll()
		return nil, err
	}

	return m, nil
}

func (m *Manager) openAll(
	configs map[string]ConnectionConfig,
) error {
	names := sortedKeys(configs)

	for _, name := range names {
		if err := m.openOne(name, configs[name]); err != nil {
			return fmt.Errorf(
				"database: connection %q: %w", name, err,
			)
		}
	}

	return nil
}

func (m *Manager) openOne(
	name string, cfg ConnectionConfig,
) error {
	db, err := sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		return err
	}

	applyPoolConfig(db, cfg)

	m.conns[name] = db
	m.drivers[name] = cfg.Driver
	m.order = append(m.order, name)

	m.logger.Info("database connection registered",
		"name", name, "driver", cfg.Driver,
	)

	return nil
}

func (m *Manager) Connection(name string) *sql.DB {
	db, ok := m.conns[name]
	if !ok {
		panic(fmt.Sprintf(
			"database: connection %q not found", name,
		))
	}

	return db
}

func (m *Manager) Default() *sql.DB {
	return m.Connection(defaultConn)
}

func (m *Manager) Driver(name string) string {
	return m.drivers[name]
}

func (m *Manager) Names() []string {
	cp := make([]string, len(m.order))
	copy(cp, m.order)

	return cp
}

func (m *Manager) Has(name string) bool {
	_, ok := m.conns[name]
	return ok
}

func (m *Manager) Name() string { return "database" }

func (m *Manager) Init(_ context.Context) error { return nil }

func (m *Manager) Start(ctx context.Context) error {
	return m.pingAll(ctx)
}

func (m *Manager) Stop(_ context.Context) error {
	return m.closeAll()
}

func (m *Manager) Health(ctx context.Context) error {
	return m.pingAll(ctx)
}

func (m *Manager) pingAll(ctx context.Context) error {
	var errs []error

	for _, name := range m.order {
		if err := m.conns[name].PingContext(ctx); err != nil {
			m.logger.Error("database ping failed",
				"name", name, "error", err,
			)
			errs = append(errs,
				fmt.Errorf("connection %q: %w", name, err),
			)
		}
	}

	return errors.Join(errs...)
}

func (m *Manager) closeAll() error {
	var errs []error

	for i := len(m.order) - 1; i >= 0; i-- {
		name := m.order[i]
		if err := m.conns[name].Close(); err != nil {
			errs = append(errs,
				fmt.Errorf("close %q: %w", name, err),
			)
		} else {
			m.logger.Info("database connection closed",
				"name", name,
			)
		}
	}

	return errors.Join(errs...)
}

func applyPoolConfig(db *sql.DB, cfg ConnectionConfig) {
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}

	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}

	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
}

func ensureLogger(log Logger) Logger {
	if log == nil {
		return noopLogger{}
	}

	return log
}

func sortedKeys(
	m map[string]ConnectionConfig,
) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}
