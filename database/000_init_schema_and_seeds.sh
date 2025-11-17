#!/bin/bash
set -euo pipefail

PSQL=(psql -v ON_ERROR_STOP=1 -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-postgres}")

declare -r SCHEMA_DIR="/docker-entrypoint-initdb.d/schema"
declare -r DEV_SEED_DIR="/docker-entrypoint-initdb.d/seeds/dev"

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

seed_mode="${PALMYRA_DB_SEED_MODE:-dev}"
if [[ "$seed_mode" == "dev" ]]; then
  run_sql_dir "$DEV_SEED_DIR" "dev seeds"
else
  echo "[db-init] Dev seeds skipped (PALMYRA_DB_SEED_MODE=$seed_mode)"
fi
