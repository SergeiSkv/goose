package goose

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"path"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	// ErrNoCurrentVersion when a current migration version is not found.
	ErrNoCurrentVersion = errors.New("no current version found")
	// ErrNoNextVersion when the next migration version is not found.
	ErrNoNextVersion = errors.New("no next version found")
	// MaxVersion is the maximum allowed version.
	MaxVersion int64 = math.MaxInt64

	registeredGoMigrations = map[int64]*Migration{}
)

// Migrations slice.
type Migrations []*Migration

// helpers so we can use pkg sort
func (ms Migrations) Len() int      { return len(ms) }
func (ms Migrations) Swap(i, j int) { ms[i], ms[j] = ms[j], ms[i] }
func (ms Migrations) Less(i, j int) bool {
	if ms[i].Version == ms[j].Version {
		panic(fmt.Sprintf("goose: duplicate version %v detected:\n%v\n%v", ms[i].Version, ms[i].Source, ms[j].Source))
	}
	return ms[i].Version < ms[j].Version
}

// Current gets the current migration.
func (ms Migrations) Current(current int64) (*Migration, error) {
	for i, migration := range ms {
		if migration.Version == current {
			return ms[i], nil
		}
	}

	return nil, ErrNoCurrentVersion
}

// Next gets the next migration.
func (ms Migrations) Next(current int64) (*Migration, error) {
	for i, migration := range ms {
		if migration.Version > current {
			return ms[i], nil
		}
	}

	return nil, ErrNoNextVersion
}

// Previous : Get the previous migration.
func (ms Migrations) Previous(current int64) (*Migration, error) {
	for i := len(ms) - 1; i >= 0; i-- {
		if ms[i].Version < current {
			return ms[i], nil
		}
	}

	return nil, ErrNoNextVersion
}

// Last gets the last migration.
func (ms Migrations) Last() (*Migration, error) {
	if len(ms) == 0 {
		return nil, ErrNoNextVersion
	}

	return ms[len(ms)-1], nil
}

// Versioned gets versioned migrations.
func (ms Migrations) versioned() (Migrations, error) {
	var migrations Migrations

	// assume that the user will never have more than 19700101000000 migrations
	for _, m := range ms {
		// parse version as timestamp
		versionTime, err := time.Parse(timestampFormat, fmt.Sprintf("%d", m.Version))

		if versionTime.Before(time.Unix(0, 0)) || err != nil {
			migrations = append(migrations, m)
		}
	}

	return migrations, nil
}

// Timestamped gets the timestamped migrations.
func (ms Migrations) timestamped() (Migrations, error) {
	var migrations Migrations

	// assume that the user will never have more than 19700101000000 migrations
	for _, m := range ms {
		// parse version as timestamp
		versionTime, err := time.Parse(timestampFormat, fmt.Sprintf("%d", m.Version))
		if err != nil {
			// probably not a timestamp
			continue
		}

		if versionTime.After(time.Unix(0, 0)) {
			migrations = append(migrations, m)
		}
	}
	return migrations, nil
}

func (ms Migrations) String() string {
	str := ""
	for _, m := range ms {
		str += fmt.Sprintln(m)
	}
	return str
}

// GoMigration is a Go migration func that is run within a transaction.
type GoMigration func(tx pgx.Tx) error

// GoMigrationNoTx is a Go migration func that is run outside a transaction.
type GoMigrationNoTx func(db *pgx.Conn) error

// AddMigration adds Go migrations.
func AddMigration(up, down GoMigration) {
	_, filename, _, _ := runtime.Caller(1)
	AddNamedMigration(filename, up, down)
}

// AddNamedMigration adds named Go migrations.
func AddNamedMigration(filename string, up, down GoMigration) {
	if err := register(filename, true, up, down, nil, nil); err != nil {
		panic(err)
	}
}

// AddMigrationNoTx adds Go migrations that will be run outside transaction.
func AddMigrationNoTx(up, down GoMigrationNoTx) {
	_, filename, _, _ := runtime.Caller(1)
	AddNamedMigrationNoTx(filename, up, down)
}

// AddNamedMigrationNoTx adds named Go migrations that will be run outside transaction.
func AddNamedMigrationNoTx(filename string, up, down GoMigrationNoTx) {
	if err := register(filename, false, nil, nil, up, down); err != nil {
		panic(err)
	}
}

func register(
	filename string,
	useTx bool,
	up, down GoMigration,
	upNoTx, downNoTx GoMigrationNoTx,
) error {
	// Sanity check caller did not mix tx and non-tx based functions.
	if (up != nil || down != nil) && (upNoTx != nil || downNoTx != nil) {
		return fmt.Errorf("cannot mix tx and non-tx based go migrations functions")
	}
	v, _ := NumericComponent(filename)
	if existing, ok := registeredGoMigrations[v]; ok {
		return fmt.Errorf("failed to add migration %q: version %d conflicts with %q",
			filename,
			v,
			existing.Source,
		)
	}
	// Add to global as a registered migration.
	registeredGoMigrations[v] = &Migration{
		Version:    v,
		Next:       -1,
		Previous:   -1,
		Registered: true,
		Source:     filename,
		UseTx:      useTx,
		UpFn:       up,
		DownFn:     down,
		UpFnNoTx:   upNoTx,
		DownFnNoTx: downNoTx,
	}
	return nil
}

func collectMigrationsFS(fsys fs.FS, dirpath string, current, target int64) (Migrations, error) {
	if _, err := fs.Stat(fsys, dirpath); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%s directory does not exist", dirpath)
	}

	var migrations Migrations

	// SQL migration files.
	sqlMigrationFiles, err := fs.Glob(fsys, path.Join(dirpath, "*.sql"))
	if err != nil {
		return nil, err
	}
	for _, file := range sqlMigrationFiles {
		v, err := NumericComponent(file)
		if err != nil {
			return nil, fmt.Errorf("could not parse SQL migration file %q: %w", file, err)
		}
		if versionFilter(v, current, target) {
			migration := &Migration{Version: v, Next: -1, Previous: -1, Source: file}
			migrations = append(migrations, migration)
		}
	}

	// Go migrations registered via goose.AddMigration().
	for _, migration := range registeredGoMigrations {
		v, err := NumericComponent(migration.Source)
		if err != nil {
			return nil, fmt.Errorf("could not parse go migration file %q: %w", migration.Source, err)
		}
		if versionFilter(v, current, target) {
			migrations = append(migrations, migration)
		}
	}

	// Go migration files
	goMigrationFiles, err := fs.Glob(fsys, path.Join(dirpath, "*.go"))
	if err != nil {
		return nil, err
	}
	for _, file := range goMigrationFiles {
		v, err := NumericComponent(file)
		if err != nil {
			continue // Skip any files that don't have version prefix.
		}

		if strings.HasSuffix(file, "_test.go") {
			continue // Skip Go test files.
		}

		// Skip migrations already existing migrations registered via goose.AddMigration().
		if _, ok := registeredGoMigrations[v]; ok {
			continue
		}

		if versionFilter(v, current, target) {
			migration := &Migration{Version: v, Next: -1, Previous: -1, Source: file, Registered: false}
			migrations = append(migrations, migration)
		}
	}

	migrations = sortAndConnectMigrations(migrations)

	return migrations, nil
}

// CollectMigrations returns all the valid looking migration scripts in the
// migrations folder and go func registry, and key them by version.
func CollectMigrations(dirpath string, current, target int64) (Migrations, error) {
	return collectMigrationsFS(baseFS, dirpath, current, target)
}

func sortAndConnectMigrations(migrations Migrations) Migrations {
	sort.Sort(migrations)

	// now that we're sorted in the appropriate direction,
	// populate next and previous for each migration
	for i, m := range migrations {
		prev := int64(-1)
		if i > 0 {
			prev = migrations[i-1].Version
			migrations[i-1].Next = m.Version
		}
		migrations[i].Previous = prev
	}

	return migrations
}

func versionFilter(v, current, target int64) bool {

	if target > current {
		return v > current && v <= target
	}

	if target < current {
		return v <= current && v > target
	}

	return false
}

// EnsureDBVersion retrieves the current version for this DB.
// Create and initialize the DB version table if it doesn't exist.
func EnsureDBVersion(db *pgx.Conn) (int64, error) {
	ctx := context.Background()
	dbMigrations, err := store.ListMigrations(ctx, db)
	if err != nil {
		return 0, createVersionTable(ctx, db)
	}
	// The most recent record for each migration specifies
	// whether it has been applied or rolled back.
	// The first version we find that has been applied is the current version.
	//
	// TODO(mf): for historic reasons, we continue to use the is_applied column,
	// but at some point we need to deprecate this logic and ideally remove
	// this column.
	//
	// For context, see:
	// https://github.com/SergeiSkv/goose/v3/pull/131#pullrequestreview-178409168
	//
	// The dbMigrations list is expected to be ordered by descending ID. But
	// in the future we should be able to query the last record only.
	skipLookup := make(map[int64]struct{})
	for _, m := range dbMigrations {
		// Have we already marked this version to be skipped?
		if _, ok := skipLookup[m.VersionID]; ok {
			continue
		}
		// If version has been applied we are done.
		if m.IsApplied {
			return m.VersionID, nil
		}
		// Latest version of migration has not been applied.
		skipLookup[m.VersionID] = struct{}{}
	}
	return 0, ErrNoNextVersion
}

// createVersionTable creates the db version table and inserts the
// initial 0 value into it.
func createVersionTable(ctx context.Context, db *pgx.Conn) error {
	txn, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	if err = store.CreateVersionTable(ctx, txn); err != nil {
		_ = txn.Rollback(ctx)
		return err
	}
	if err = store.InsertVersion(ctx, txn, 0); err != nil {
		_ = txn.Rollback(ctx)
		return err
	}
	return txn.Commit(ctx)
}

// GetDBVersion is an alias for EnsureDBVersion, but returns -1 in error.
func GetDBVersion(db *pgx.Conn) (int64, error) {
	version, err := EnsureDBVersion(db)
	if err != nil {
		return -1, err
	}

	return version, nil
}
