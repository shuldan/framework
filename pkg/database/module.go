package database

import (
	"fmt"
	"strings"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
	"github.com/shuldan/framework/pkg/errors"
)

type Module struct {
	connections map[string]contracts.Database
}

func NewModule() contracts.AppModule {
	return &Module{
		connections: make(map[string]contracts.Database),
	}
}

func (m *Module) Name() string {
	return contracts.DatabaseModuleName
}

func (m *Module) Register(container contracts.DIContainer) error {
	return container.Factory("database", func(c contracts.DIContainer) (interface{}, error) {
		cfg, err := c.Resolve(contracts.ConfigModuleName)
		if err != nil {
			return nil, ErrResolveConfig.
				WithDetail("reason", err.Error()).
				WithCause(err)
		}

		config := cfg.(contracts.Config)
		dbConfig, ok := config.GetSub("database")
		if !ok {
			return nil, ErrConfigNotFound
		}

		defaultConnection := dbConfig.GetString("default", "primary")
		connectionsConfig, ok := dbConfig.GetSub("connections")
		if !ok {
			return nil, ErrConnectionsNotFound
		}

		connections := make(map[string]contracts.Database)
		allConnections := connectionsConfig.All()
		for name, connData := range allConnections {
			connMap, ok := connData.(map[string]interface{})
			if !ok {
				continue
			}
			db, err := m.createConnection(name, connMap)
			if err != nil {
				return nil, ErrCreateConnection.
					WithDetail("name", name).
					WithDetail("reason", err.Error()).
					WithCause(err)
			}
			connections[name] = db
			connKey := fmt.Sprintf("database.%s", name)
			if err := c.Instance(connKey, db); err != nil {
				return nil, ErrRegisterConnection.
					WithDetail("name", name).
					WithDetail("reason", err.Error()).
					WithCause(err)
			}
		}
		if defaultDB, ok := connections[defaultConnection]; ok {
			if err := c.Instance("database.default", defaultDB); err != nil {
				return nil, ErrRegisterConnection.
					WithDetail("name", "default").
					WithDetail("reason", err.Error()).
					WithCause(err)
			}
		}

		m.connections = connections
		return connections, nil
	})
}

func (m *Module) Start(ctx contracts.AppContext) error {
	dbs, err := ctx.Container().Resolve("database")
	if err != nil {
		return ErrResolveConnections.
			WithDetail("reason", err.Error()).
			WithCause(err)
	}

	connections := dbs.(map[string]contracts.Database)

	for name, db := range connections {
		if err := db.Connect(); err != nil {
			return ErrConnectDatabase.
				WithDetail("name", name).
				WithDetail("reason", err.Error()).
				WithCause(err)
		}
	}

	return nil
}

func (m *Module) Stop(_ contracts.AppContext) error {
	var errs []error

	for name, db := range m.connections {
		if err := db.Close(); err != nil {
			errs = append(errs, ErrCloseDatabase.
				WithDetail("name", name).
				WithDetail("reason", err.Error()).
				WithCause(err))
		}
	}

	if len(errs) > 0 {
		return ErrMultipleCloseErrors.WithCause(errors.Join(errs...))
	}

	return nil
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
