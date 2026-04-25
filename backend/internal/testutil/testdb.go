package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"backend/internal/platform/migrate"
	"backend/internal/testseed"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const defaultTestDSN = "postgres://postgres:postgres@localhost:5432/inventory_manager_test?sslmode=disable"
const embeddedDSN = "host=localhost port=15432 user=postgres password=postgres dbname=postgres sslmode=disable"
const embeddedPort = 15432

var (
	setupOnce  sync.Once
	globalDB   *sql.DB
	setupErr   error
	embeddedPG *embeddedpostgres.EmbeddedPostgres
)

func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	setupOnce.Do(func() {
		dsn := os.Getenv("TEST_DATABASE_URL")

		if dsn != "" {
			globalDB, setupErr = connectAndMigrate(dsn)
			return
		}

		globalDB, setupErr = connectAndMigrate(defaultTestDSN)
		if setupErr == nil {
			return
		}

		fmt.Println("[testutil] External PostgreSQL not reachable, starting embedded PostgreSQL...")
		embeddedPG = embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
			Port(embeddedPort).
			Username("postgres").
			Password("postgres").
			Database("postgres"))

		if err := embeddedPG.Start(); err != nil {
			setupErr = fmt.Errorf("start embedded postgres: %w", err)
			return
		}

		globalDB, setupErr = connectAndMigrate(embeddedDSN)
	})

	if setupErr != nil {
		t.Fatalf("test database setup failed: %v", setupErr)
	}

	if err := testseed.ResetDatabase(context.Background(), globalDB); err != nil {
		t.Fatalf("reset test database: %v", err)
	}

	return globalDB
}

func connectAndMigrate(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sql db: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db at %s: %w", dsn, err)
	}

	runner := migrate.NewRunner(db)
	if _, err := runner.Up(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	return db, nil
}

func StopEmbeddedPostgres() {
	if embeddedPG != nil {
		embeddedPG.Stop()
	}
}

func TruncateAll(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := testseed.ResetDatabase(context.Background(), db); err != nil {
		t.Fatalf("truncate all: %v", err)
	}
}

// SeedMasterData inserts the minimum master data needed for integration tests.
// It respects the scope hierarchy trigger from migration 000012:
//   1. Seed scope_systems
//   2. Create a device
//   3. Create system-type scope (no parent)
//   4. Create assembly-type scopes under the system scope
func SeedMasterData(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := testseed.SeedMasterData(context.Background(), db); err != nil {
		t.Fatalf("seed master data: %v", err)
	}
}

func MustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), query, args...); err != nil {
		t.Fatalf("MustExec failed: %s: %v", query, err)
	}
}

func MustQueryInt(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var result int
	if err := db.QueryRowContext(context.Background(), query, args...).Scan(&result); err != nil {
		t.Fatalf("MustQueryInt failed: %s: %v", query, err)
	}
	return result
}
