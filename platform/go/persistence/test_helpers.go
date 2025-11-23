package persistence

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	coreSchemaOnce       sync.Once
	coreSchemaStatements []string
	coreSchemaInitErr    error
	schemaDirParts       = []string{"database", "schema"}
)

// applyCoreSchemaDDL bootstraps the admin schema for tests using the shared helper.
// It remains unexported for backward compatibility in persistence tests.
func applyCoreSchemaDDL(ctx context.Context, pool *pgxpool.Pool) error {
	return BootstrapAdminSchema(ctx, pool, "tenant_admin")
}

func repoRootFromFile() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("determine schema helper path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..")), nil
}

func splitStatements(sql string) []string {
	rawStatements := strings.Split(sql, ";")
	statements := make([]string, 0, len(rawStatements))
	for _, raw := range rawStatements {
		stmt := strings.TrimSpace(raw)
		if stmt == "" {
			continue
		}
		statements = append(statements, stmt)
	}
	return statements
}

// applyDDLToSchema applies a specific schema SQL file into the given schema using search_path.
func applyDDLToSchema(ctx context.Context, pool *pgxpool.Pool, schema, filename string) error {
	if _, err := pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS `+schema); err != nil {
		return err
	}

	root, err := repoRootFromFile()
	if err != nil {
		return err
	}
	path := filepath.Join(root, "database", "schema", filename)
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	stmts := splitStatements(string(content))

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err = tx.Exec(ctx, `SELECT set_config('search_path', $1, false)`, schema); err != nil {
		return err
	}

	for _, stmt := range stmts {
		if _, err := tx.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
