# Offline Demo App

Lightweight, Dexie-backed demo UI for local CRUD. No API required.

## Develop locally

1) Start the backend stack (if desired) via Compose, omitting the offline demo so the port stays free:
```bash
docker compose up postgres api web-platform-admin
```

2) Run the offline app dev server:
```bash
export VITE_API_BASE_URL=${VITE_API_BASE_URL:-/api/v1}   # point to your API (e.g. http://localhost:3000/api/v1)
export VITE_API_TOKEN=${VITE_API_TOKEN:-<paste-your-generated-token>} # unsigned dev JWT (tenant slug \"demo\"), override as needed
pnpm -C examples/offline dev --host --port 4174
```
Then open http://localhost:4174.

## Build
```bash
VITE_API_BASE_URL=${VITE_API_BASE_URL:-/api/v1} \
VITE_API_TOKEN=${VITE_API_TOKEN:-<paste-your-generated-token>} \
pnpm -C examples/offline build
```

### Generate a dev JWT (tenant \"demo\") and set it

Run this and copy the output token:
```bash
node - <<'JS'
function b64(o){return Buffer.from(JSON.stringify(o)).toString('base64url')}
const now=Math.floor(Date.now()/1000);
const header={alg:'none',typ:'JWT'};
const payload={
  iss:'https://securetoken.google.com/local-palmyra',
  aud:'local-palmyra',
  auth_time:now,
  user_id:'demo-admin',
  sub:'demo-admin',
  iat:now,
  exp:now+3600,
  email:'demo@example.com',
  email_verified:true,
  name:'Demo Admin',
  isAdmin:true,
  palmyraRoles:['admin'],
  tenantRoles:['admin'],
  tenantSlug:'demo',
  firebase:{
    identities:{email:['demo@example.com']},
    sign_in_provider:'password',
    tenant:'demo'
  }
};
console.log(`${b64(header)}.${b64(payload)}`);
JS
```
Set `VITE_API_TOKEN` to that value (env export, `.env`, or compose arg).

### Docker Compose with your token

```bash
cp examples/offline/.env.example examples/offline/.env
# edit examples/offline/.env to set VITE_API_TOKEN to your generated token
docker compose -f examples/offline/docker-compose.yml \
  --env-file examples/offline/.env \
  up --build
```

## Docker-compose (bootstrap everything automatically)

We ship a dedicated compose stack that bootstraps the offline demo end-to-end: admin tenant, demo category, Persons schema, and a demo tenant.

Run it from the repo root:

```bash
docker compose -f examples/offline/docker-compose.yml up --build
```

What it does (via `examples/offline/tools/bootstrap-offline.sh`):
1. Wait for Postgres.
2. `cli-platform-admin bootstrap platform` (admin tenant).
3. Upsert **Demo Category** (ID `00000000-0000-4000-8000-000000000001`, slug `demo-category`).
4. Upsert **Persons** schema (ID `00000000-0000-4000-8000-000000000002`, version `1.0.0`, table `persons`, slug `persons`, definition at `examples/offline/schemas/person.json`).
5. Create **demo tenant** (defaults: slug `demo`, admin email `demo@example.com`, admin name `Demo Admin`).

Env overrides (optional):
- Admin/bootstrap: `ENV_KEY`, `ADMIN_TENANT_SLUG`, `ADMIN_TENANT_NAME`, `ADMIN_EMAIL`, `ADMIN_FULL_NAME`
- Category: `DEMO_CATEGORY_ID`, `DEMO_CATEGORY_NAME`, `DEMO_CATEGORY_SLUG`
- Schema: `DEMO_SCHEMA_ID`, `DEMO_SCHEMA_VERSION`, `DEMO_SCHEMA_TABLE`, `DEMO_SCHEMA_SLUG`, `DEMO_SCHEMA_FILE`
- Tenant: `DEMO_TENANT_SLUG`, `DEMO_TENANT_NAME`, `DEMO_TENANT_ADMIN_EMAIL`, `DEMO_TENANT_ADMIN_FULL_NAME`

Notes:
- The bootstrap script is idempotent (safe to rerun).
- Persons JSON Schema lives at `examples/offline/schemas/person.json`.

## E2E
```bash
docker compose -f examples/offline/e2e/docker-compose.yml up --build --abort-on-container-exit --exit-code-from playwright
```
Runs a Playwright runner (Node 24) that installs browsers at runtime, starts the offline app via the Playwright `webServer` hook, and executes `pnpm -C examples/offline test:e2e`.
