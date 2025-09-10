package database

import (
	"sync"

	"github.com/shuldan/framework/pkg/contracts"
	"github.com/shuldan/framework/pkg/errors"
)

type databasePool struct {
	mu          sync.RWMutex
	connections map[string]contracts.Database
}

func newDatabasePool() *databasePool {
	return &databasePool{
		connections: make(map[string]contracts.Database),
	}
}

func (p *databasePool) registerDatabase(name string, db contracts.Database) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.connections[name]; exists {
		return ErrRegisterConnection.WithDetail("name", name).WithDetail("reason", "connection already exists")
	}

	p.connections[name] = db
	return nil
}

func (p *databasePool) getDatabase(name string) (contracts.Database, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	db, exists := p.connections[name]
	return db, exists
}

func (p *databasePool) connectAll() error {
	var errs []error
	for _, db := range p.connections {
		if err := db.Connect(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return ErrMultipleFailedToOpenDatabase.WithCause(errors.Join(errs...))
	}

	return nil
}

func (p *databasePool) closeAll() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for name, db := range p.connections {
		if err := db.Close(); err != nil {
			errs = append(errs, ErrCloseDatabase.WithDetail("name", name).WithDetail("reason", err.Error()).WithCause(err))
		}
	}

	if len(errs) > 0 {
		return ErrMultipleCloseErrors.WithCause(errors.Join(errs...))
	}
	return nil
}
