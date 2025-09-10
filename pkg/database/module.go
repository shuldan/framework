package database

import (
	"fmt"
	"strings"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

type Module struct {
	pool *databasePool
}

func NewModule() contracts.AppModule {
	return &Module{
		pool: newDatabasePool(),
	}
}

func (m *Module) Name() string {
	return contracts.DatabaseModuleName
}

func (m *Module) Register(container contracts.DIContainer) error {
	cfg, err := container.Resolve(contracts.ConfigModuleName)
	if err != nil {
		return ErrResolveConfig.
			WithDetail("reason", err.Error()).
			WithCause(err)
	}

	config := cfg.(contracts.Config)
	dbConfig, ok := config.GetSub("database")
	if !ok {
		return ErrConfigNotFound.WithDetail("reason", "database config not found")
	}

	defaultConnectionName := dbConfig.GetString("default", "primary")
	connectionsConfig, ok := dbConfig.GetSub("connections")
	if !ok {
		return ErrConnectionsNotFound
	}
	allConnections := connectionsConfig.All()
	for name, connData := range allConnections {
		connMap, ok := connData.(map[string]interface{})
		if !ok {
			continue
		}
		db, err := m.createConnection(name, connMap)
		if err != nil {
			return ErrCreateConnection.
				WithDetail("name", name).
				WithDetail("reason", err.Error()).
				WithCause(err)
		}
		if err := m.pool.registerDatabase(name, db); err != nil {
			return err
		}
		if err := container.Instance(contracts.DatabaseModuleName+".connections."+name, db); err != nil {
			return err
		}
		if name == defaultConnectionName {
			if err := container.Instance(contracts.DatabaseModuleName+"."+name, db); err != nil {
				return err
			}
		}
	}

	if err := m.pool.connectAll(); err != nil {
		return err
	}

	return nil
}

func (m *Module) Start(ctx contracts.AppContext) error {
	return nil
}

func (m *Module) Stop(_ contracts.AppContext) error {
	return m.pool.closeAll()
}

func (m *Module) CliCommands(ctx contracts.AppContext) ([]contracts.CliCommand, error) {
	registry := ctx.AppRegistry()
	for _, module := range registry.All() {
		if provider, ok := module.(contracts.MigrationsProvider); ok {
			migrations := provider.Migrations()
			for _, m := range migrations {
				registerMigration(m)
			}
		}
	}

	container := ctx.Container()
	configRaw, err := container.Resolve(contracts.ConfigModuleName)
	if err != nil {
		return nil, err
	}
	config, ok := configRaw.(contracts.Config)
	if !ok {
		return nil, fmt.Errorf("invalid config instance")
	}

	loggerRaw, err := container.Resolve(contracts.LoggerModuleName)
	if err != nil {
		return nil, err
	}
	logger, ok := loggerRaw.(contracts.Logger)
	if !ok {
		return nil, fmt.Errorf("invalid logger instance")
	}

	return []contracts.CliCommand{
		newMigrationUpCommand(m.pool, config, logger),
		newMigrationDownCommand(m.pool, config, logger),
		newMigrationStatusCommand(m.pool, config, logger),
	}, nil
}

func (m *Module) createConnection(name string, config map[string]interface{}) (contracts.Database, error) {
	driver, ok := config["driver"].(string)
	if !ok {
		return nil, ErrDriverNotSpecified.WithDetail("name", name)
	}

	dsn, ok := config["dsn"].(string)
	if !ok {
		return nil, ErrDSNNotSpecified.WithDetail("name", name)
	}

	sqlDriver := m.getSQLDriver(driver)

	options := m.getConnectionOptions(config)

	db := NewDatabase(sqlDriver, dsn, options...)

	return db, nil
}

func (m *Module) getSQLDriver(driver string) string {
	switch strings.ToLower(driver) {
	case "mysql":
		return "mysql"
	case "postgres", "postgresql":
		return "postgres"
	case "sqlite", "sqlite3":
		return "sqlite3"
	default:
		return driver
	}
}

func (m *Module) getConnectionOptions(config map[string]interface{}) []Option {
	var options []Option

	if poolConfig, ok := config["pool"].(map[string]interface{}); ok {
		maxOpen := m.getIntValue(poolConfig, "max_open_connections", 25)
		maxIdle := m.getIntValue(poolConfig, "max_idle_connections", 5)
		connMaxLifetime := m.getDurationValue(poolConfig, "conn_max_lifetime", time.Hour)
		connMaxIdleTime := m.getDurationValue(poolConfig, "conn_max_idle_time", 5*time.Minute)

		options = append(options,
			WithConnectionPool(maxOpen, maxIdle, connMaxLifetime),
			WithConnectionIdleTime(connMaxIdleTime),
		)
	}

	options = append(options, WithPingTimeout(5*time.Second))

	options = append(options, WithRetry(3, time.Second))

	return options
}

func (m *Module) getIntValue(config map[string]interface{}, key string, defaultValue int) int {
	if val, ok := config[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		case string:

			var result int
			if _, err := fmt.Sscanf(v, "%d", &result); err == nil {
				return result
			}
		}
	}
	return defaultValue
}

func (m *Module) getDurationValue(config map[string]interface{}, key string, defaultValue time.Duration) time.Duration {
	if val, ok := config[key]; ok {
		switch v := val.(type) {
		case string:
			if duration, err := time.ParseDuration(v); err == nil {
				return duration
			}
		case int:
			return time.Duration(v) * time.Second
		case int64:
			return time.Duration(v) * time.Second
		case float64:
			return time.Duration(v) * time.Second
		}
	}
	return defaultValue
}
