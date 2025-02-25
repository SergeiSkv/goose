package goose

import (
	"fmt"

	"github.com/SergeiSkv/goose/v3/internal/dialect"
)

func init() {
	store, _ = dialect.NewStore(dialect.Postgres, TableName())
}

var store dialect.Store

// SetDialect sets the dialect to use for the goose package.
func SetDialect(s string) error {
	var d dialect.Dialect
	switch s {
	case "postgres", "pgx":
		d = dialect.Postgres
	case "mysql":
		d = dialect.Mysql
	case "sqlite3", "sqlite":
		d = dialect.Sqlite3
	case "mssql", "azuresql":
		d = dialect.Sqlserver
	case "redshift":
		d = dialect.Redshift
	case "tidb":
		d = dialect.Tidb
	case "clickhouse":
		d = dialect.Clickhouse
	case "vertica":
		d = dialect.Vertica
	default:
		return fmt.Errorf("%q: unknown dialect", s)
	}
	var err error
	store, err = dialect.NewStore(d, TableName())
	return err
}
