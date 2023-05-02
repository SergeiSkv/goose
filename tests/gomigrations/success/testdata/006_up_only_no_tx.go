package gomigrations

import (
	"context"

	"github.com/SergeiSkv/goose/v3"
	"github.com/jackc/pgx/v5"
)

func init() {
	goose.AddMigrationNoTx(up006, nil)
}

func up006(db *pgx.Conn) error {
	q := "INSERT INTO users VALUES (1, 'admin@example.com')"
	_, err := db.Exec(context.Background(), q)
	return err
}
