package database

import "errors"

var (
	ErrNoConnections      = errors.New("database: no connections configured")
	ErrConnectionNotFound = errors.New("database: connection not found")
)
