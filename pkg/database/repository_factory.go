package database

import (
	"database/sql"
	"github.com/shuldan/framework/pkg/contracts"
)

type RepositoryFactory struct {
	db *sql.DB
}

func NewRepositoryFactory(db *sql.DB) *RepositoryFactory {
	return &RepositoryFactory{db: db}
}

func (f *RepositoryFactory) GetDatabase() *sql.DB {
	return f.db
}

func CreateSimpleRepository[T contracts.Aggregate, I contracts.ID, M contracts.Memento](
	db *sql.DB,
	mapper AggregateMapper[T, M],
) contracts.TransactionalRepository[T, I] {
	return NewSimpleRepository[T, I, M](db, mapper)
}

func CreateStrategyRepository[T contracts.Aggregate, I contracts.ID, M contracts.Memento](
	db *sql.DB,
	mapper StrategyMapper[T, M],
	defaultStrategy contracts.LoadingStrategy,
) contracts.StrategyRepository[T, I] {
	return NewStrategyRepository[T, I, M](db, mapper, defaultStrategy)
}
