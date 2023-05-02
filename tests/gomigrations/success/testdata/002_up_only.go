package gomigrations

import (
	"database/sql"
)

func init() {
	goose.AddMigration(up002, nil)
}

func up002(tx *sql.Tx) error {
	q := "INSERT INTO foo VALUES (1, 1, 'Alice')"
	_, err := tx.Exec(q)
	return err
}
