package goose

import (
	"context"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5"
)

// Run a migration specified in raw SQL.
//
// Sections of the script can be annotated with a special comment,
// starting with "-- +goose" to specify whether the section should
// be applied during an Up or Down migration
//
// All statements following an Up or Down annotation are grouped together
// until another direction annotation is found.
func runSQLMigration(
	ctx context.Context,
	db *pgx.Conn,
	statements []string,
	useTx bool,
	v int64,
	direction bool,
	noVersioning bool,
) error {
	if useTx {
		// TRANSACTION.

		verboseInfo("Begin transaction")

		tx, err := db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		for _, query := range statements {
			verboseInfo("Executing statement: %s\n", clearStatement(query))
			if _, err = tx.Exec(ctx, query); err != nil {
				verboseInfo("Rollback transaction")
				_ = tx.Rollback(ctx)
				return fmt.Errorf("failed to execute SQL query %q: %w", clearStatement(query), err)
			}
		}

		if !noVersioning {
			if direction {
				if err := store.InsertVersion(ctx, tx, v); err != nil {
					verboseInfo("Rollback transaction")
					_ = tx.Rollback(ctx)
					return fmt.Errorf("failed to insert new goose version: %w", err)
				}
			} else {
				if err := store.DeleteVersion(ctx, tx, v); err != nil {
					verboseInfo("Rollback transaction")
					_ = tx.Rollback(ctx)
					return fmt.Errorf("failed to delete goose version: %w", err)
				}
			}
		}

		verboseInfo("Commit transaction")
		if err = tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		return nil
	}

	// NO TRANSACTION.
	for _, query := range statements {
		verboseInfo("Executing statement: %s", clearStatement(query))
		if _, err := db.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to execute SQL query %q: %w", clearStatement(query), err)
		}
	}
	if !noVersioning {
		if direction {
			if err := store.InsertVersionNoTx(ctx, db, v); err != nil {
				return fmt.Errorf("failed to insert new goose version: %w", err)
			}
		} else {
			if err := store.DeleteVersionNoTx(ctx, db, v); err != nil {
				return fmt.Errorf("failed to delete goose version: %w", err)
			}
		}
	}

	return nil
}

const (
	grayColor  = "\033[90m"
	resetColor = "\033[00m"
)

func verboseInfo(s string, args ...interface{}) {
	if verbose {
		if noColor {
			log.Printf(s, args...)
		} else {
			log.Printf(grayColor+s+resetColor, args...)
		}
	}
}

var (
	matchSQLComments = regexp.MustCompile(`(?m)^--.*$[\r\n]*`)
	matchEmptyEOL    = regexp.MustCompile(`(?m)^$[\r\n]*`) // TODO: Duplicate
)

func clearStatement(s string) string {
	s = matchSQLComments.ReplaceAllString(s, ``)
	return matchEmptyEOL.ReplaceAllString(s, ``)
}
