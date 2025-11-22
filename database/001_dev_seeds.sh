#!/bin/bash
set -euo pipefail

PSQL=(psql -v ON_ERROR_STOP=1 -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-postgres}")

ADMIN_TENANT_SLUG="${ADMIN_TENANT_SLUG:-admin}"
ADMIN_SCHEMA="tenant_${ADMIN_TENANT_SLUG//-/_}"

run_sql_dir() {
  local dir="$1"
  local label="$2"
  if [[ ! -d "$dir" ]]; then
    echo "[db-init] Skipping missing directory $dir"
    return
  }

  mapfile -t files < <(find "$dir" -maxdepth 1 -type f -name '*.sql' | sort)
  if [[ ${#files[@]} -eq 0 ]]; then
    echo "[db-init] No SQL files found under $dir"
    return
  }

  echo "[db-init] Applying ${label} files from $dir"
  for file in "${files[@]}"; do
    echo "[db-init] -> $(basename "$file")"
    "${PSQL[@]}" -f "$file"
  done
}

seed_mode="${PLATFORM_DB_SEED_MODE:-dev}"
if [[ "$seed_mode" == "dev" ]]; then
  # search_path already set by 000_init_schema.sh at database level; this is a no-op guard if run standalone.
  "${PSQL[@]}" -c "SET search_path TO ${ADMIN_SCHEMA};" >/dev/null
  run_sql_dir "/docker-entrypoint-initdb.d/seeds/dev" "dev seeds"
else
  echo "[db-init] Dev seeds skipped (PLATFORM_DB_SEED_MODE=$seed_mode)"
fi
