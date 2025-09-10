package database

import (
	"flag"
	"fmt"

	"github.com/shuldan/framework/pkg/contracts"
	"github.com/shuldan/framework/pkg/errors"
)

type migrationAbstractCommand struct {
	pool           *databasePool
	connectionName string
}

func (c *migrationAbstractCommand) Configure(flags *flag.FlagSet) {
	flags.StringVar(&c.connectionName, "connection", "", "Apply migrations only to this specific connection")
	flags.StringVar(&c.connectionName, "c", "", "Apply migrations only to this specific connection (shorthand)")
}

func (c *migrationAbstractCommand) processAllConnections(_ contracts.CliContext, op func(connName string, db contracts.Database) error) error {
	var errs []error
	connectionNames := getAllConnectionNames()

	for _, connectionName := range connectionNames {
		db, ok := c.pool.getDatabase(connectionName)
		if !ok {
			errs = append(errs, ErrConnectionNotFound.WithDetail("name", connectionName))
			continue
		}
		if err := op(connectionName, db); err != nil {
			errs = append(errs, fmt.Errorf("connection '%s': %w", connectionName, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
