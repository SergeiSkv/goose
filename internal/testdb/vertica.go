package testdb

import (
	"github.com/jackc/pgx/v5"
	_ "github.com/vertica/vertica-sql-go"
)

func newVertica(opts ...OptionsFunc) (*pgx.Conn, func(), error) {
	// TODO: delete
	return nil, nil, nil
}
