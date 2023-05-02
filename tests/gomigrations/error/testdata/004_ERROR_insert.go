package gomigrations

import (
	"context"
	"fmt"

	"github.com/SergeiSkv/goose/v3"
	"github.com/jackc/pgx/v5"
)

func init() {
	goose.AddMigration(up004, nil)
}

func up004(tx pgx.Tx) error {
	for i := 1; i <= 100; i++ {
		// Simulate an error when no tx. We should have 50 rows
		// inserted in the DB.
		if i == 50 {
			return fmt.Errorf("simulate error: too many inserts")
		}
		q := "INSERT INTO foo VALUES ($1)"
		if _, err := tx.Exec(context.Background(), q); err != nil {
			return err
		}
	}
	return nil
}
