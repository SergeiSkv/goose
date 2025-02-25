package goose

import (
	"context"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
)

// Reset rolls back all migrations
func Reset(db *pgx.Conn, dir string, opts ...OptionsFunc) error {
	ctx := context.Background()
	option := &options{}
	for _, f := range opts {
		f(option)
	}
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return fmt.Errorf("failed to collect migrations: %w", err)
	}
	if option.noVersioning {
		return DownTo(db, dir, minVersion, opts...)
	}

	statuses, err := dbMigrationsStatus(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to get status of migrations: %w", err)
	}
	sort.Sort(sort.Reverse(migrations))

	for _, migration := range migrations {
		if !statuses[migration.Version] {
			continue
		}
		if err = migration.Down(db); err != nil {
			return fmt.Errorf("failed to db-down: %w", err)
		}
	}

	return nil
}

func dbMigrationsStatus(ctx context.Context, db *pgx.Conn) (map[int64]bool, error) {
	dbMigrations, err := store.ListMigrations(ctx, db)
	if err != nil {
		return nil, err
	}
	// The most recent record for each migration specifies
	// whether it has been applied or rolled back.
	results := make(map[int64]bool)

	for _, m := range dbMigrations {
		if _, ok := results[m.VersionID]; ok {
			continue
		}
		results[m.VersionID] = m.IsApplied
	}
	return results, nil
}
