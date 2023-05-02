package gomigrations

import (
	"context"

	"github.com/SergeiSkv/goose/v3"
	"github.com/jackc/pgx/v5"
)

func init() {
	goose.AddMigrationNoTx(nil, down007)
}

func down007(db *pgx.Conn) error {
	q := "TRUNCATE TABLE users"
	_, err := db.Exec(context.Background(), q)
	return err
}
