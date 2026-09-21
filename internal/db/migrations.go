package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"

	migrationfiles "github.com/grapinou/LazyMarking/db/migrations"
	"github.com/pressly/goose/v3"
)

// MigrationReport describes the forward-only schema preparation performed
// before the server starts serving requests.
type MigrationReport struct {
	FromVersion int64
	ToVersion   int64
	Applied     []int64
}

// OpenMigratedDB opens a database and applies every pending embedded migration.
// It returns no usable connection if migration preparation fails.
func OpenMigratedDB(ctx context.Context, dbPath string) (*sql.DB, MigrationReport, error) {
	return openMigratedDB(ctx, dbPath, migrationfiles.Files)
}

func openMigratedDB(ctx context.Context, dbPath string, migrations fs.FS) (*sql.DB, MigrationReport, error) {
	conn, err := InitDB(dbPath)
	if err != nil {
		return nil, MigrationReport{}, err
	}
	report, err := applyMigrations(ctx, conn, migrations)
	if err != nil {
		_ = conn.Close()
		return nil, report, err
	}
	return conn, report, nil
}

func applyMigrations(ctx context.Context, conn *sql.DB, migrations fs.FS) (MigrationReport, error) {
	provider, err := newMigrationProvider(conn, migrations)
	if err != nil {
		return MigrationReport{}, fmt.Errorf("prepare embedded database migrations: %w", err)
	}
	from, target, err := provider.GetVersions(ctx)
	if err != nil {
		return MigrationReport{}, fmt.Errorf("read database migration version: %w", err)
	}
	report := MigrationReport{FromVersion: from, ToVersion: target}
	if from > target {
		return report, fmt.Errorf("database schema version %d is newer than supported version %d", from, target)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return report, fmt.Errorf("apply embedded database migrations from version %d to %d: %w", from, target, err)
	}
	for _, result := range results {
		report.Applied = append(report.Applied, result.Source.Version)
	}
	current, target, err := provider.GetVersions(ctx)
	if err != nil {
		return report, fmt.Errorf("verify database migration version: %w", err)
	}
	report.ToVersion = target
	if current != target {
		return report, fmt.Errorf("database migration incomplete: current version %d, expected %d", current, target)
	}
	return report, nil
}

func newMigrationProvider(conn *sql.DB, migrations fs.FS) (*goose.Provider, error) {
	return goose.NewProvider(
		goose.DialectSQLite3,
		conn,
		migrations,
		goose.WithDisableGlobalRegistry(true),
	)
}
