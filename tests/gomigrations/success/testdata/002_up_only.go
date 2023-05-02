package gomigrations

import (
	"context"

	"github.com/SergeiSkv/goose/v3"
	"github.com/jackc/pgx/v5"
)

func init() {
	goose.AddMigration(up002, nil)
}

func up002(tx pgx.Tx) error {
	q := "INSERT INTO foo VALUES (1, 1, 'Alice')"
	_, err := tx.Exec(context.Background(), q)
	return err
}
