#!/usr/bin/env bash
set -euo pipefail

log() { echo "[bootstrap-offline] $*"; }

DB_URL=${DATABASE_URL:-postgres://palmyra:palmyra@postgres:5432/palmyra?sslmode=disable}
ENV_KEY_VAL=${ENV_KEY:-dev}
ADMIN_SLUG=${ADMIN_TENANT_SLUG:-admin}

CATEGORY_ID=${DEMO_CATEGORY_ID:-00000000-0000-4000-8000-000000000001}
CATEGORY_NAME=${DEMO_CATEGORY_NAME:-Demo Category}
CATEGORY_SLUG=${DEMO_CATEGORY_SLUG:-demo-category}

SCHEMA_ID=${DEMO_SCHEMA_ID:-00000000-0000-4000-8000-000000000002}
SCHEMA_VERSION=${DEMO_SCHEMA_VERSION:-1.0.0}
SCHEMA_TABLE=${DEMO_SCHEMA_TABLE:-persons}
SCHEMA_SLUG=${DEMO_SCHEMA_SLUG:-persons}
SCHEMA_FILE=${DEMO_SCHEMA_FILE:-/schemas/person.json}

DEMO_TENANT_SLUG=${DEMO_TENANT_SLUG:-demo}
DEMO_TENANT_NAME=${DEMO_TENANT_NAME:-Demo}
DEMO_TENANT_ADMIN_EMAIL=${DEMO_TENANT_ADMIN_EMAIL:-demo@example.com}
DEMO_TENANT_ADMIN_FULL_NAME=${DEMO_TENANT_ADMIN_FULL_NAME:-Demo Admin}

log "waiting for postgres..."
/usr/local/bin/wait-for-postgres.sh postgres 5432

log "bootstrap platform (admin tenant: ${ADMIN_SLUG})"
/usr/local/bin/cli-platform-admin bootstrap platform \
  --database-url "${DB_URL}" \
  --env-key "${ENV_KEY_VAL}" \
  --admin-tenant-slug "${ADMIN_SLUG}" \
  --admin-tenant-name "${ADMIN_TENANT_NAME:-Admin}" \
  --admin-email "${ADMIN_EMAIL:-admin@example.com}" \
  --admin-full-name "${ADMIN_FULL_NAME:-Palmyra Admin}"

log "upsert demo category (${CATEGORY_ID})"
/usr/local/bin/cli-platform-admin schema categories upsert \
  --database-url "${DB_URL}" \
  --env-key "${ENV_KEY_VAL}" \
  --admin-tenant-slug "${ADMIN_SLUG}" \
  --id "${CATEGORY_ID}" \
  --name "${CATEGORY_NAME}" \
  --slug "${CATEGORY_SLUG}" \
  --description "Schemas for offline demo"

log "upsert persons schema (${SCHEMA_ID} v${SCHEMA_VERSION})"
/usr/local/bin/cli-platform-admin schema definitions upsert \
  --database-url "${DB_URL}" \
  --env-key "${ENV_KEY_VAL}" \
  --admin-tenant-slug "${ADMIN_SLUG}" \
  --schema-id "${SCHEMA_ID}" \
  --schema-version "${SCHEMA_VERSION}" \
  --table-name "${SCHEMA_TABLE}" \
  --slug "${SCHEMA_SLUG}" \
  --category-id "${CATEGORY_ID}" \
  --definition-file "${SCHEMA_FILE}"

log "create demo tenant (${DEMO_TENANT_SLUG})"
/usr/local/bin/cli-platform-admin tenant create \
  --database-url "${DB_URL}" \
  --env-key "${ENV_KEY_VAL}" \
  --tenant-slug "${DEMO_TENANT_SLUG}" \
  --tenant-name "${DEMO_TENANT_NAME}" \
  --admin-email "${DEMO_TENANT_ADMIN_EMAIL}" \
  --admin-full-name "${DEMO_TENANT_ADMIN_FULL_NAME}"

log "bootstrap complete"
