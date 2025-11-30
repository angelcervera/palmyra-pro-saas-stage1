#!/usr/bin/env bash

set -euo pipefail

DB_URL=${1:-}

if [[ -z "${DB_URL}" ]]; then
  echo "Usage: $0 <database-url>" >&2
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BIN_PATH="${ROOT_DIR}/bin/cli-platform-admin"
ENV_KEY="dev"
ADMIN_SLUG="admin"

echo "[1] Building CLI binary -> ${BIN_PATH}" >&2
go build -o "${BIN_PATH}" ./apps/cli-platform-admin

run() {
  echo "\n>>>>>> $*" >&2
  "$@"
}

new_category() {
  local name slug attempts=0
  while :; do
    name="cli-cat-$(date +%s%N)"
    slug="${name}"
    if run "${BIN_PATH}" schema categories upsert \
      --database-url "${DB_URL}" \
      --env-key "${ENV_KEY}" \
      --admin-tenant-slug "${ADMIN_SLUG}" \
      --name "${name}" \
      --slug "${slug}"; then
      break
    fi
    attempts=$((attempts + 1))
    if [[ ${attempts} -ge 3 ]]; then
      echo "failed to create category after ${attempts} attempts" >&2
      exit 1
    fi
    sleep 1
  done
}

first_category_id() {
  "${BIN_PATH}" schema categories list \
    --database-url "${DB_URL}" \
    --env-key "${ENV_KEY}" \
    --admin-tenant-slug "${ADMIN_SLUG}" \
    | awk 'NR==2 {print $1}'
}

list_categories() {
  run "${BIN_PATH}" schema categories list \
    --database-url "${DB_URL}" \
    --env-key "${ENV_KEY}" \
    --admin-tenant-slug "${ADMIN_SLUG}"
}

update_category() {
  local id=$1
  run "${BIN_PATH}" schema categories upsert \
    --database-url "${DB_URL}" \
    --env-key "${ENV_KEY}" \
    --admin-tenant-slug "${ADMIN_SLUG}" \
    --id "${id}" \
    --name "${2:-updated-${id}}"
}

delete_category() {
  local id=$1
  run "${BIN_PATH}" schema categories delete \
    --database-url "${DB_URL}" \
    --env-key "${ENV_KEY}" \
    --admin-tenant-slug "${ADMIN_SLUG}" \
    --id "${id}"
}

schema_list_all() {
  run "${BIN_PATH}" schema definitions list \
    --database-url "${DB_URL}" \
    --env-key "${ENV_KEY}" \
    --admin-tenant-slug "${ADMIN_SLUG}"
}

schema_list_all_full() {
  run "${BIN_PATH}" schema definitions list \
    --database-url "${DB_URL}" \
    --env-key "${ENV_KEY}" \
    --admin-tenant-slug "${ADMIN_SLUG}" \
    --include-inactive \
    --include-deleted
}

schema_upsert() {
  local category_id=$1
  local schema_file=$2
  local schema_id=${3:-}
  local version_flag=()
  local id_flag=()
  if [[ -n "${schema_id}" ]]; then
    id_flag=(--schema-id "${schema_id}")
  fi
  if [[ -n "${4:-}" ]]; then
    version_flag=(--schema-version "${4}")
  fi

  run "${BIN_PATH}" schema definitions upsert \
    --database-url "${DB_URL}" \
    --env-key "${ENV_KEY}" \
    --admin-tenant-slug "${ADMIN_SLUG}" \
    --table-name "cli_entities" \
    --slug "cli-schema" \
    --category-id "${category_id}" \
    --definition-file "${schema_file}" \
    "${id_flag[@]}" "${version_flag[@]}"
}

schema_delete() {
  local schema_id=$1 version=$2
  run "${BIN_PATH}" schema definitions delete \
    --database-url "${DB_URL}" \
    --env-key "${ENV_KEY}" \
    --admin-tenant-slug "${ADMIN_SLUG}" \
    --schema-id "${schema_id}" \
    --schema-version "${version}"
}

temp_schema() {
  local file
  file=$(mktemp)
  cat >"${file}" <<'EOF'
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "name": {"type": "string"}
  },
  "required": ["name"],
  "additionalProperties": false
}
EOF
  echo "${file}"
}

temp_schema_v2() {
  local file
  file=$(mktemp)
  cat >"${file}" <<'EOF'
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "name": {"type": "string"},
    "description": {"type": "string"}
  },
  "required": ["name"],
  "additionalProperties": false
}
EOF
  echo "${file}"
}

echo "[2] Create category & list" >&2
new_category
list_categories

echo "[3] Update first category & list" >&2
cat_id=$(first_category_id)
update_category "${cat_id}" "updated-${cat_id}"
list_categories

echo "[4] Delete first category & list" >&2
delete_category "${cat_id}"
list_categories

echo "[5] Repeat create/update" >&2
new_category
list_categories
cat_id=$(first_category_id)
update_category "${cat_id}" "updated-${cat_id}-2"
list_categories

echo "[6] Schema lifecycle using latest category" >&2
latest_category=$(first_category_id)
schema_v1=$(temp_schema)
schema_v2=$(temp_schema_v2)

create_output=$(schema_upsert "${latest_category}" "${schema_v1}")
echo "${create_output}"
schema_list_all

schema_id=$(echo "${create_output}" | awk '/Upserted schema definition/ {print $4}')
version1=$(echo "${create_output}" | awk '/Upserted schema definition/ {print $6}')

update_output=$(schema_upsert "${latest_category}" "${schema_v2}" "${schema_id}" "1.0.1")
echo "${update_output}"
schema_list_all

schema_delete "${schema_id}" "1.0.1"
schema_list_all_full

# reactivate prior version so we end with an active schema
schema_upsert "${latest_category}" "${schema_v1}" "${schema_id}" "${version1}" >/dev/null
schema_list_all

echo "Done." >&2
