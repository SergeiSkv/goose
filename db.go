package goose

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// OpenDBWithDriver creates a connection to a database, and modifies goose
// internals to be compatible with the supplied driver by calling SetDialect.
func OpenDBWithDriver(dbstring string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dbstring)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(context.Background(), config)

}
