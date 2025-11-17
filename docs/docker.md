# Docker & Local Environment Guidelines

This repository ships with three Docker-related entry points:

1. `apps/api/Dockerfile` — multi-stage build for the Go API. This image is production-friendly and is intended to be deployed to Google Cloud Run (or any container platform). It compiles the API binary statically and copies the OpenAPI contracts needed at runtime.
2. `apps/web-admin/Dockerfile` — multi-stage build for the React admin UI. It installs the monorepo with pnpm, builds the Vite bundle, and publishes the compiled assets behind an Nginx proxy that forwards `/api/*` calls to the Go API.
3. `docker-compose.yml` — local development orchestration for PostgreSQL, the API, and the admin UI. Use this only for development or demos. It is **not** a production deployment template.

## When to use which

| Scenario                                   | Use                               | Notes                                                                                                                                                                                                |
|--------------------------------------------|-----------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Local development (API + DB + Admin UI)    | `docker compose up --build`       | Reads configuration from `.env.dockercompose`. Spins up `postgres`, `api`, and `web-admin` containers. Auth middleware defaults to `AUTH_PROVIDER=dev`, and the `web-admin` service proxies `/api/*` to the API. |
| Deploying backend (Cloud Run, etc.)        | Build from `apps/api/Dockerfile`  | Supply real environment variables (e.g., `DATABASE_URL`, `AUTH_PROVIDER=firebase`, `FIREBASE_CONFIG`, `GCLOUD_PROJECT`). Do **not** use Docker Compose for production.                                 |
| Deploying static admin UI (Cloud storage)  | Build from `apps/web-admin/Dockerfile` | Publishes a ready-to-serve bundle. Provide a reverse proxy (or CDN) that forwards `/api/*` to the backend service, matching what the Nginx stage does locally.                                          |

## Local development workflow

1. Adjust `.env.dockercompose` if you need non-default credentials or want to point to Firebase service-account JSON.
2. Start the stack:
   ```bash
   docker compose up --build
   ```
3. Once the stack is up:
   - Admin UI: `http://localhost:4173` (served by the `web-admin` container). Requests to `/api/*` are transparently proxied to the Go API service.
   - API: `http://localhost:3000`.
   - PostgreSQL: `localhost:5432` with credentials from the env file (default `palmyra` / `palmyra`).
4. Tear everything down when done:
   ```bash
   docker compose down
   ```
5. Optional: if you still prefer hot-reload development, you can run `pnpm dev -C apps/web-admin` locally and skip the `web-admin` container. Compose continues to provide Postgres + API for that workflow.

### Environment variables (.env.dockercompose)

- `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD` – database credentials.
- `DATABASE_URL` – connection string the API uses (defaults to the Postgres service).
- `AUTH_PROVIDER` – `dev` (unsigned tokens) for local testing; set to `firebase` in production.
- `VITE_API_BASE_URL` – compile-time base URL injected into the admin UI image. Defaults to `/api/v1`, which works with the built-in proxy. Override with a fully qualified URL if you are not routing traffic through the Nginx proxy.
- `VITE_ENV` – optional environment label surfaced inside the UI (e.g., `local`, `staging`).
- Optional: `FIREBASE_CONFIG` (path inside the container to service-account JSON), `GCLOUD_PROJECT`.

If a key is missing, the `docker-compose.yml` file provides safe defaults via `${VAR:-default}` syntax.

### Tips

- To inspect logs: `docker compose logs -f api`, `docker compose logs -f web-admin`, or `docker compose logs -f postgres`.
- To run database commands or connect via psql: `psql postgres://palmyra:palmyra@localhost:5432/palmyra`.
- The Compose file is purposely minimal. Add volumes or tooling overrides in a local override file if needed (`docker-compose.override.yml`).

## Deploying to Cloud Run (or similar)

- This should be used from CI/CD pipelines.
- Build the production image from `apps/api/Dockerfile`:
  ```bash
  docker build -t gcr.io/<project>/palmyra-api:latest -f apps/api/Dockerfile .
  ```
- Configure environment variables in the deployment platform (Cloud Run service configuration) — **do not** rely on `.env.dockercompose`.
- Set `AUTH_PROVIDER=firebase` and supply `FIREBASE_CONFIG` / `GCLOUD_PROJECT`. Ensure secrets are handled securely (Secret Manager or equivalent).

## Why Compose is dev-only

- No TLS termination, scaling, high availability (HA) or production hardening.
- Credentials are stored in a local env file for convenience.
- Authentication defaults to unsigned tokens (`dev` mode). Production must run with real Firebase verification.

Keep these goals in mind as you use the Docker assets: Compose is your local playground, while the Dockerfile is your deployable artifact.
