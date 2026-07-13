package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func Open(ctx context.Context, path string) (*sql.DB, error) {
	if err := osMkdir(filepath.Dir(path)); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	for _, q := range []string{"PRAGMA journal_mode=WAL", "PRAGMA foreign_keys=ON", "PRAGMA busy_timeout=5000"} {
		if _, err = db.ExecContext(ctx, q); err != nil {
			db.Close()
			return nil, err
		}
	}
	if err = Migrate(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
func osMkdir(dir string) error {
	if dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0755)
}
func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)"); err != nil {
		return err
	}
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, e := range entries {
		var version int
		if _, err := fmt.Sscanf(e.Name(), "%d_", &version); err != nil {
			return err
		}
		var exists int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version=?", version).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		b, err := migrationFiles.ReadFile("migrations/" + e.Name())
		if err != nil {
			return err
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, string(b)); err == nil {
			_, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations(version) VALUES(?)", version)
		}
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s: %w", e.Name(), err)
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
