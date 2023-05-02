package gomigrations

import (
	"database/sql"
)

func init() {
	goose.AddMigrationNoTx(nil, down007)
}

func down007(db *sql.DB) error {
	q := "TRUNCATE TABLE users"
	_, err := db.Exec(q)
	return err
}
