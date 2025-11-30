# Offline Demo App

Lightweight, Dexie-backed demo UI for local CRUD. No API required.

## Develop locally

1) Start the backend stack (if desired) via Compose, omitting the offline demo so the port stays free:
```bash
docker compose up postgres api web-platform-admin
```

2) Run the offline app dev server:
```bash
pnpm -C apps/demos/offline dev --host --port 4174
```
Then open http://localhost:4174.

## Build
```bash
pnpm -C apps/demos/offline build
```

## Docker-compose (bootstrap everything automatically)

We ship a dedicated compose stack that bootstraps the offline demo end-to-end: admin tenant, demo category, Persons schema, and a demo tenant.

Run it from the repo root:

```bash
docker compose -f docker-compose.offline.yml --env-file .env.dockercompose up --build
```

What it does (via `apps/demos/offline/tools/bootstrap-offline.sh`):
1. Wait for Postgres.
2. `cli-platform-admin bootstrap platform` (admin tenant).
3. Upsert **Demo Category** (ID `00000000-0000-4000-8000-000000000001`, slug `demo-category`).
4. Upsert **Persons** schema (ID `00000000-0000-4000-8000-000000000002`, version `1.0.0`, table `persons`, slug `persons`, definition at `apps/demos/offline/schemas/person.json`).
5. Create **demo tenant** (defaults: slug `demo`, admin email `demo@example.com`, admin name `Demo Admin`).

Env overrides (optional):
- Admin/bootstrap: `ENV_KEY`, `ADMIN_TENANT_SLUG`, `ADMIN_TENANT_NAME`, `ADMIN_EMAIL`, `ADMIN_FULL_NAME`
- Category: `DEMO_CATEGORY_ID`, `DEMO_CATEGORY_NAME`, `DEMO_CATEGORY_SLUG`
- Schema: `DEMO_SCHEMA_ID`, `DEMO_SCHEMA_VERSION`, `DEMO_SCHEMA_TABLE`, `DEMO_SCHEMA_SLUG`, `DEMO_SCHEMA_FILE`
- Tenant: `DEMO_TENANT_SLUG`, `DEMO_TENANT_NAME`, `DEMO_TENANT_ADMIN_EMAIL`, `DEMO_TENANT_ADMIN_FULL_NAME`

Notes:
- The bootstrap script is idempotent (safe to rerun).
- Persons JSON Schema lives at `apps/demos/offline/schemas/person.json`.

## E2E
```bash
docker compose -f apps/demos/offline/e2e/docker-compose.yml up --build --abort-on-container-exit --exit-code-from playwright
```
Runs a Playwright runner (Node 24) that installs browsers at runtime, starts the offline app via the Playwright `webServer` hook, and executes `pnpm -C apps/demos/offline test:e2e`.
