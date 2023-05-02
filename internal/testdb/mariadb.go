package testdb

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
)

func newMariaDB(opts ...OptionsFunc) (*pgx.Conn, func(), error) {
	// TODO: delete
	return nil, nil, nil
}
