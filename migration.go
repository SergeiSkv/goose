package goose

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/SergeiSkv/goose/v3/internal/sqlparser"
	"github.com/jackc/pgx/v5"
)

// MigrationRecord struct.
type MigrationRecord struct {
	VersionID int64
	TStamp    time.Time
	IsApplied bool // was this a result of up() or down()
}

// Migration struct.
type Migration struct {
	Version              int64
	Next                 int64  // next version, or -1 if none
	Previous             int64  // previous version, -1 if none
	Source               string // path to .sql script or go file
	Registered           bool
	UseTx                bool
	UpFn, DownFn         GoMigration
	UpFnNoTx, DownFnNoTx GoMigrationNoTx
	noVersioning         bool
}

func (m *Migration) String() string {
	return fmt.Sprint(m.Source)
}

// Up runs an up migration.
func (m *Migration) Up(db *pgx.Conn) error {
	ctx := context.Background()
	if err := m.run(ctx, db, true); err != nil {
		return err
	}
	return nil
}

// Down runs a down migration.
func (m *Migration) Down(db *pgx.Conn) error {
	ctx := context.Background()
	if err := m.run(ctx, db, false); err != nil {
		return err
	}
	return nil
}

func (m *Migration) run(ctx context.Context, db *pgx.Conn, direction bool) error {
	switch filepath.Ext(m.Source) {
	case ".sql":
		f, err := baseFS.Open(m.Source)
		if err != nil {
			return fmt.Errorf("ERROR %v: failed to open SQL migration file: %w", filepath.Base(m.Source), err)
		}
		defer f.Close()

		statements, useTx, err := sqlparser.ParseSQLMigration(f, sqlparser.FromBool(direction), verbose)
		if err != nil {
			return fmt.Errorf("ERROR %v: failed to parse SQL migration file: %w", filepath.Base(m.Source), err)
		}

		start := time.Now()
		if err := runSQLMigration(ctx, db, statements, useTx, m.Version, direction, m.noVersioning); err != nil {
			return fmt.Errorf("ERROR %v: failed to run SQL migration: %w", filepath.Base(m.Source), err)
		}
		finish := truncateDuration(time.Since(start))

		if len(statements) > 0 {
			log.Printf("OK   %s (%s)\n", filepath.Base(m.Source), finish)
		} else {
			log.Printf("EMPTY %s (%s)\n", filepath.Base(m.Source), finish)
		}

	case ".go":
		if !m.Registered {
			return fmt.Errorf("ERROR %v: failed to run Go migration: Go functions must be registered and built into a custom binary (see https://github.com/SergeiSkv/goose/v3/tree/master/examples/go-migrations)", m.Source)
		}
		start := time.Now()
		var empty bool
		if m.UseTx {
			// Run go-based migration inside a tx.
			fn := m.DownFn
			if direction {
				fn = m.UpFn
			}
			empty = (fn == nil)
			if err := runGoMigration(
				ctx,
				db,
				fn,
				m.Version,
				direction,
				!m.noVersioning,
			); err != nil {
				return fmt.Errorf("ERROR go migration: %q: %w", filepath.Base(m.Source), err)
			}
		} else {
			// Run go-based migration outside a tx.
			fn := m.DownFnNoTx
			if direction {
				fn = m.UpFnNoTx
			}
			empty = (fn == nil)
			if err := runGoMigrationNoTx(
				ctx,
				db,
				fn,
				m.Version,
				direction,
				!m.noVersioning,
			); err != nil {
				return fmt.Errorf("ERROR go migration no tx: %q: %w", filepath.Base(m.Source), err)
			}
		}
		finish := truncateDuration(time.Since(start))
		if !empty {
			log.Printf("OK   %s (%s)\n", filepath.Base(m.Source), finish)
		} else {
			log.Printf("EMPTY %s (%s)\n", filepath.Base(m.Source), finish)
		}
	}
	return nil
}

func runGoMigrationNoTx(
	ctx context.Context,
	db *pgx.Conn,
	fn GoMigrationNoTx,
	version int64,
	direction bool,
	recordVersion bool,
) error {
	if fn != nil {
		// Run go migration function.
		if err := fn(db); err != nil {
			return fmt.Errorf("failed to run go migration: %w", err)
		}
	}
	if recordVersion {
		return insertOrDeleteVersionNoTx(ctx, db, version, direction)
	}
	return nil
}

func runGoMigration(
	ctx context.Context,
	db *pgx.Conn,
	fn GoMigration,
	version int64,
	direction bool,
	recordVersion bool,
) error {
	if fn == nil && !recordVersion {
		return nil
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	if fn != nil {
		// Run go migration function.
		if err := fn(tx); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to run go migration: %w", err)
		}
	}
	if recordVersion {
		if err := insertOrDeleteVersion(ctx, tx, version, direction); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to update version: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func insertOrDeleteVersion(ctx context.Context, tx pgx.Tx, version int64, direction bool) error {
	if direction {
		return store.InsertVersion(ctx, tx, version)
	}
	return store.DeleteVersion(ctx, tx, version)
}

func insertOrDeleteVersionNoTx(ctx context.Context, db *pgx.Conn, version int64, direction bool) error {
	if direction {
		return store.InsertVersionNoTx(ctx, db, version)
	}
	return store.DeleteVersionNoTx(ctx, db, version)
}

// NumericComponent looks for migration scripts with names in the form:
// XXX_descriptivename.ext where XXX specifies the version number
// and ext specifies the type of migration
func NumericComponent(name string) (int64, error) {
	base := filepath.Base(name)

	if ext := filepath.Ext(base); ext != ".go" && ext != ".sql" {
		return 0, errors.New("not a recognized migration file type")
	}

	idx := strings.Index(base, "_")
	if idx < 0 {
		return 0, errors.New("no filename separator '_' found")
	}

	n, e := strconv.ParseInt(base[:idx], 10, 64)
	if e == nil && n <= 0 {
		return 0, errors.New("migration IDs must be greater than zero")
	}

	return n, e
}

func truncateDuration(d time.Duration) time.Duration {
	for _, v := range []time.Duration{
		time.Second,
		time.Millisecond,
		time.Microsecond,
	} {
		if d > v {
			return d.Round(v / time.Duration(100))
		}
	}
	return d
}
