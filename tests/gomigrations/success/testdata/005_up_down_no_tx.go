package gomigrations

import (
	"context"

	"github.com/SergeiSkv/goose/v3"
	"github.com/jackc/pgx/v5"
)

func init() {
	goose.AddMigrationNoTx(up005, down005)
}

func up005(db *pgx.Conn) error {
	q := "CREATE TABLE users (id INT, email TEXT)"
	_, err := db.Exec(context.Background(), q)
	return err
}

func down005(db *pgx.Conn) error {
	q := "DROP TABLE IF EXISTS users"
	_, err := db.Exec(context.Background(), q)
	return err
}
