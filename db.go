package goose

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// OpenDBWithDriver creates a connection to a database, and modifies goose
// internals to be compatible with the supplied driver by calling SetDialect.
func OpenDBWithDriver(driver string, dbstring string) (*pgx.Conn, error) {
	if err := SetDialect(driver); err != nil {
		return nil, err
	}

	switch driver {
	case "mssql":
		driver = "sqlserver"
	case "redshift":
		driver = "postgres"
	case "tidb":
		driver = "mysql"
	}

	switch driver {
	case "postgres", "pgx":
		conConfig, err := pgx.ParseConfig(dbstring)
		if err != nil {
			return nil, err
		}
		conConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
		return pgx.ConnectConfig(context.Background(), conConfig)
	default:
		return nil, fmt.Errorf("unsupported driver %s", driver)
	}
}
