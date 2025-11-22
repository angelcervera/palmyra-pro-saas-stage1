package persistence

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// mustTestPool creates a transient test database connection pool and applies core schema DDL.
// It reuses the helper already used by other persistence tests.
func mustTestPool(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, testDatabaseURL())
	if err != nil {
		t.Fatalf("create test pool: %v", err)
	}

	if err := applyCoreSchemaDDL(ctx, pool); err != nil {
		pool.Close()
		t.Fatalf("apply core schema: %v", err)
	}

	cleanup := func() {
		pool.Close()
	}

	return pool, cleanup
}

// testDatabaseURL reads TEST_DATABASE_URL or falls back to a local default.
// This mirrors other persistence tests' expectation of an external Postgres (e.g., Testcontainers).
func testDatabaseURL() string {
	if url, ok := os.LookupEnv("TEST_DATABASE_URL"); ok && url != "" {
		return url
	}
	return "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
}
