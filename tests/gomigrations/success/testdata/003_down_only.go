package gomigrations

import (
	"context"

	"github.com/SergeiSkv/goose/v3"
	"github.com/jackc/pgx/v5"
)

func init() {
	goose.AddMigration(nil, down003)
}

func down003(tx pgx.Tx) error {
	q := "TRUNCATE TABLE foo"
	_, err := tx.Exec(context.Background(), q)
	return err
}
