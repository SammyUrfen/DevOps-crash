package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"sort"
)

// Migrate applies every .sql file that this database has not seen, in file
// name order. Each file runs inside a transaction with its ledger row, so a
// failed migration leaves no half-applied schema.
//
// This replaces a migration tool on purpose. The tool earns its place when a
// down migration or a checksum check is needed. Neither is needed yet.
func Migrate(ctx context.Context, db *sql.DB, fsys fs.FS) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name       TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}

	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && path.Ext(e.Name()) == ".sql" {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		applied, err := apply(ctx, db, fsys, name)
		if err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if applied {
			slog.Info("migration applied", "name", name)
		}
	}
	return nil
}

func apply(ctx context.Context, db *sql.DB, fsys fs.FS, name string) (bool, error) {
	body, err := fs.ReadFile(fsys, name)
	if err != nil {
		return false, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	// ON CONFLICT DO NOTHING plus RowsAffected tells us whether this migration
	// is new, and locks the row against a second pod starting at the same time.
	res, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (name) VALUES ($1) ON CONFLICT DO NOTHING`, name)
	if err != nil {
		return false, err
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, string(body)); err != nil {
		return false, err
	}
	return true, tx.Commit()
}
