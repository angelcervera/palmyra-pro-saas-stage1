# E2E stack (Playwright)

This compose stack spins up everything needed for Playwright E2E runs against the admin web app and API:

- `postgres`: Postgres 16 with local volume `./.data/e2e/postgres`
- `api`: Go API built from `apps/api/Dockerfile` with dev auth and local storage
- `admin-web`: Nginx + built SPA from `apps/web-platform-admin/Dockerfile` (proxies `/api/` to the API service)
- `playwright`: Official Playwright image that mounts the repo and runs the test suite

## Usage

```bash
# Build and start core services
docker compose -f docker-compose.e2e.yml up -d postgres api admin-web

# Run Playwright inside the dedicated container
docker compose -f docker-compose.e2e.yml run --rm playwright

# Tear down
docker compose -f docker-compose.e2e.yml down
```

Notes:
- Config lives in `.env.e2e`; defaults use `AUTH_PROVIDER=dev` and local storage.
- The admin web proxies API calls to `http://api:3000/api/` inside the compose network; externally it listens on `4173`.
- The Playwright service mounts the workspace and runs `pnpm -C apps/demos/offline test:e2e`. Adjust the command if you add more suites.
