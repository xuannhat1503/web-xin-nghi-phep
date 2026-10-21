package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	migratemysql "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/g-asiapacific/elm/backend/db/migrations"
	"github.com/g-asiapacific/elm/backend/internal/config"
)

var migrationFile = regexp.MustCompile(`^(\d+)_.+\.up\.sql$`)

// LatestMigration returns the highest migration version embedded in the binary.
func LatestMigration() (uint, error) {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return 0, err
	}
	var versions []uint
	for _, e := range entries {
		if m := migrationFile.FindStringSubmatch(e.Name()); m != nil {
			v, err := strconv.ParseUint(m[1], 10, 32)
			if err != nil {
				return 0, err
			}
			versions = append(versions, uint(v))
		}
	}
	if len(versions) == 0 {
		return 0, errors.New("no embedded migrations")
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })
	return versions[len(versions)-1], nil
}

// Migrator wraps golang-migrate with the embedded migrations. Migration files need multi-statement support,
// so it uses its own connection instead of the application pool.
type Migrator struct {
	m  *migrate.Migrate
	db *sql.DB
}

func NewMigrator(c config.DB) (*Migrator, error) {
	db, err := sql.Open("mysql", DSN(c, true))
	if err != nil {
		return nil, err
	}
	if err := CheckServer(context.Background(), db); err != nil {
		db.Close()
		return nil, err
	}
	driver, err := migratemysql.WithInstance(db, &migratemysql.Config{})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate driver: %w", err)
	}
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		db.Close()
		return nil, err
	}
	m, err := migrate.NewWithInstance("iofs", src, "mysql", driver)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate init: %w", err)
	}
	return &Migrator{m: m, db: db}, nil
}

// Up applies all pending migrations. Returns true if anything was applied.
func (mg *Migrator) Up() (bool, error) {
	err := mg.m.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		return false, nil
	}
	return err == nil, err
}

// Down rolls back exactly n migrations.
func (mg *Migrator) Down(n int) error {
	if n < 1 {
		return errors.New("steps must be >= 1")
	}
	return mg.m.Steps(-n)
}

// Version returns the current version; version 0 means no migration has been applied.
func (mg *Migrator) Version() (uint, bool, error) {
	v, dirty, err := mg.m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	return v, dirty, err
}

func (mg *Migrator) Close() error {
	srcErr, dbErr := mg.m.Close()
	return errors.Join(srcErr, dbErr)
}

// SchemaVersion reads the applied version from the application pool (used by readiness checks).
func SchemaVersion(ctx context.Context, db *sql.DB) (version uint, dirty bool, err error) {
	err = db.QueryRowContext(ctx, "SELECT version, dirty FROM schema_migrations LIMIT 1").Scan(&version, &dirty)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return version, dirty, err
}
