package gomigrations

import (
	"context"

	"github.com/SergeiSkv/goose/v3"
	"github.com/jackc/pgx/v5"
)

func init() {
	goose.AddMigrationNoTx(up001, nil)
}

func up001(db *pgx.Conn) error {
	q := "CREATE TABLE foo (id INT)"
	_, err := db.Exec(context.Background(), q)
	return err
}
