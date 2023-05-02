package gomigrations

import (
	"database/sql"
)

func init() {
	goose.AddMigrationNoTx(up006, nil)
}

func up006(db *sql.DB) error {
	q := "INSERT INTO users VALUES (1, 'admin@example.com')"
	_, err := db.Exec(q)
	return err
}
