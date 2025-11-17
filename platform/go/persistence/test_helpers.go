package persistence

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
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

// applyCoreSchemaDDL loads and executes the base SQL schema files used across environments.
// Tests call this helper so they can bootstrap a clean database without relying on embedded DDL.
func applyCoreSchemaDDL(ctx context.Context, pool *pgxpool.Pool) error {
	coreSchemaOnce.Do(func() {
		root, err := repoRootFromFile()
		if err != nil {
			coreSchemaInitErr = err
			return
		}

		dirPath := filepath.Join(append([]string{root}, schemaDirParts...)...)
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			coreSchemaInitErr = fmt.Errorf("read schema dir (%s): %w", dirPath, err)
			return
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})

		var statements []string
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
				continue
			}

			content, readErr := os.ReadFile(filepath.Join(dirPath, entry.Name()))
			if readErr != nil {
				coreSchemaInitErr = fmt.Errorf("read schema file %s: %w", entry.Name(), readErr)
				return
			}

			statements = append(statements, splitStatements(string(content))...)
		}

		if len(statements) == 0 {
			coreSchemaInitErr = fmt.Errorf("schema directory %s contained no SQL statements", dirPath)
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
