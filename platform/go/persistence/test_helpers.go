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
	coreSchemaOnce        sync.Once
	coreSchemaStatements  []string
	coreSchemaInitErr     error
	coreSchemaRelativeDir = []string{"docker", "initdb", "000_core_schema.sql"}
)

// applyCoreSchemaDDL loads and executes the base SQL schema used by the Docker init scripts.
// Tests call this helper so they can bootstrap a clean database without relying on embedded DDL.
func applyCoreSchemaDDL(ctx context.Context, pool *pgxpool.Pool) error {
	coreSchemaOnce.Do(func() {
		_, filename, _, ok := runtime.Caller(0)
		if !ok {
			coreSchemaInitErr = fmt.Errorf("determine schema helper path")
			return
		}

		root := filepath.Join(filepath.Dir(filename), "..", "..", "..")
		path := filepath.Join(append([]string{root}, coreSchemaRelativeDir...)...)

		contents, err := os.ReadFile(path)
		if err != nil {
			coreSchemaInitErr = fmt.Errorf("read core schema ddl (%s): %w", path, err)
			return
		}

		rawStatements := strings.Split(string(contents), ";")
		statements := make([]string, 0, len(rawStatements))
		for _, raw := range rawStatements {
			stmt := strings.TrimSpace(raw)
			if stmt == "" {
				continue
			}
			statements = append(statements, stmt)
		}

		if len(statements) == 0 {
			coreSchemaInitErr = fmt.Errorf("core schema ddl file is empty")
			return
		}

		coreSchemaStatements = statements
	})

	if coreSchemaInitErr != nil {
		return coreSchemaInitErr
	}

	for _, stmt := range coreSchemaStatements {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("apply core schema ddl: %w", err)
		}
	}

	return nil
}
