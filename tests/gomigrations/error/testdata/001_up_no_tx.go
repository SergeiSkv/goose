package gomigrations

import (
	"database/sql"
)

func init() {
	goose.AddMigrationNoTx(up001, nil)
}

func up001(db *sql.DB) error {
	q := "CREATE TABLE foo (id INT)"
	_, err := db.Exec(q)
	return err
}
