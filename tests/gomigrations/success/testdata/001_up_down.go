package gomigrations

import (
	"context"

	"github.com/SergeiSkv/goose/v3"
	"github.com/jackc/pgx/v5"
)

func init() {
	goose.AddMigration(up001, down001)
}

func up001(tx pgx.Tx) error {
	q := "CREATE TABLE foo (id INT, subid INT, name TEXT)"
	_, err := tx.Exec(context.Background(), q)
	return err
}

func down001(tx pgx.Tx) error {
	q := "DROP TABLE IF EXISTS foo"
	_, err := tx.Exec(context.Background(), q)
	return err
}
