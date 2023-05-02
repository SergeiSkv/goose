package main

import (
	"context"

	"github.com/SergeiSkv/goose/v3"
	"github.com/jackc/pgx/v5"
)

func init() {
	goose.AddMigration(Up00002, Down00002)
}

func Up00002(tx pgx.Tx) error {
	_, err := tx.Exec(context.Background(), "UPDATE users SET username='admin' WHERE username='root';")
	if err != nil {
		return err
	}
	return nil
}

func Down00002(tx pgx.Tx) error {
	_, err := tx.Exec(context.Background(), "UPDATE users SET username='root' WHERE username='admin';")
	if err != nil {
		return err
	}
	return nil
}
