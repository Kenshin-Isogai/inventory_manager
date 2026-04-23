package migrate

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Runner struct {
	db *sql.DB
}

type StatusRow struct {
	Name    string
	Applied bool
}

func NewRunner(db *sql.DB) *Runner {
	return &Runner{db: db}
}

func (r *Runner) Up(ctx context.Context) (int, error) {
	if err := r.ensureSchemaTable(ctx); err != nil {
		return 0, err
	}

	files, err := upFiles()
	if err != nil {
		return 0, err
	}

	applied := 0
	for _, name := range files {
		done, err := r.isApplied(ctx, name)
		if err != nil {
			return applied, err
		}
		if done {
			continue
		}

		payload, err := migrationFiles.ReadFile("migrations/" + name)
		if err != nil {
			return applied, fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return applied, fmt.Errorf("begin tx %s: %w", name, err)
		}

		if _, err := tx.ExecContext(ctx, string(payload)); err != nil {
			tx.Rollback()
			return applied, fmt.Errorf("execute migration %s: %w", name, err)
		}

		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, name); err != nil {
			tx.Rollback()
			return applied, fmt.Errorf("record migration %s: %w", name, err)
		}

		if err := tx.Commit(); err != nil {
			return applied, fmt.Errorf("commit migration %s: %w", name, err)
		}
		applied++
	}

	return applied, nil
}

func (r *Runner) Status(ctx context.Context) ([]StatusRow, error) {
	if err := r.ensureSchemaTable(ctx); err != nil {
		return nil, err
	}

	files, err := upFiles()
	if err != nil {
		return nil, err
	}

	rows := make([]StatusRow, 0, len(files))
	for _, name := range files {
		done, err := r.isApplied(ctx, name)
		if err != nil {
			return nil, err
		}
		rows = append(rows, StatusRow{Name: name, Applied: done})
	}

	return rows, nil
}

func (r *Runner) ensureSchemaTable(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}
	return nil
}

func (r *Runner) isApplied(ctx context.Context, name string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name = $1)`, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("query applied migration %s: %w", name, err)
	}
	return exists, nil
}

func upFiles() ([]string, error) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}
