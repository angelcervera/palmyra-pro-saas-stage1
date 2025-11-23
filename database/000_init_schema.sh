#!/bin/bash
set -euo pipefail

PSQL=(psql -v ON_ERROR_STOP=1 -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-postgres}")

ADMIN_TENANT_SLUG="${ADMIN_TENANT_SLUG:-admin}"
ADMIN_SCHEMA="tenant_${ADMIN_TENANT_SLUG//-/_}"

# Ensure admin schema exists and becomes default search_path for this database.
"${PSQL[@]}" -c "CREATE SCHEMA IF NOT EXISTS \"${ADMIN_SCHEMA}\"" 
"${PSQL[@]}" -c "ALTER DATABASE \"${POSTGRES_DB:-postgres}\" SET search_path TO \"${ADMIN_SCHEMA}\""

SCHEMA_DIR="/docker-entrypoint-initdb.d/schema"

run_sql_dir() {
  local dir="$1"
  local label="$2"
  if [[ ! -d "$dir" ]]; then
    echo "[db-init] Skipping missing directory $dir"
    return
  fi

  mapfile -t files < <(find "$dir" -maxdepth 1 -type f -name '*.sql' | sort)
  if [[ ${#files[@]} -eq 0 ]]; then
    echo "[db-init] No SQL files found under $dir"
    return
  fi

  echo "[db-init] Applying ${label} files from $dir"
  for file in "${files[@]}"; do
    echo "[db-init] -> $(basename "$file")"
    "${PSQL[@]}" -f "$file"
  done
}

run_sql_dir "$SCHEMA_DIR" "schema"
