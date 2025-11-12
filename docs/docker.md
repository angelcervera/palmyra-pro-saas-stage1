# Docker & Local Environment Guidelines

This repository ships with two Docker-related entry points:

1. `apps/api/Dockerfile` — multi-stage build for the Go API. This image is production-friendly and is intended to be deployed to Google Cloud Run (or any container platform). It compiles the API binary statically and copies the OpenAPI contracts needed at runtime.
2. `docker-compose.yml` — local development orchestration for the API + PostgreSQL. Use this only for development. It is **not** a production deployment template.

## When to use which

| Scenario                            | Use                              | Notes                                                                                                                                                                                   |
|-------------------------------------|----------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Local development (API + DB)        | `docker compose up --build`      | Reads configuration from `.env.dockercompose`. Spins up `postgres` and `api` containers. Auth middleware defaults to `AUTH_PROVIDER=dev`, which allows unsigned JWTs for local testing. |
| Deploying backend (Cloud Run, etc.) | Build from `apps/api/Dockerfile` | Supply real environment variables (e.g., `DATABASE_URL`, `AUTH_PROVIDER=firebase`, `FIREBASE_CONFIG`, `GCLOUD_PROJECT`). Do **not** use Docker Compose for production.                  |

## Local development workflow

1. Adjust `.env.dockercompose` if you need non-default credentials or want to point to Firebase service-account JSON.
2. Start the stack:
   ```bash
   docker compose up --build
   ```
3. API will listen on `http://localhost:3000`. PostgreSQL is exposed on `localhost:5432` with credentials from the env file (default `tcgdb` / `tcgdb`).
4. Run the admin UI in another terminal:
   ```bash
   pnpm dev -C apps/web-admin
   ```
   Set `VITE_API_BASE_URL=http://localhost:3000/api/v1` (default in `.env.example`).
5. Tear everything down when done:
   ```bash
   docker compose down
   ```

### Environment variables (.env.dockercompose)

- `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD` – database credentials.
- `DATABASE_URL` – connection string the API uses (defaults to the Postgres service).
- `AUTH_PROVIDER` – `dev` (unsigned tokens) for local testing; set to `firebase` in production.
- Optional: `FIREBASE_CONFIG` (path inside the container to service-account JSON), `GCLOUD_PROJECT`.

If a key is missing, the `docker-compose.yml` file provides safe defaults via `${VAR:-default}` syntax.

### Tips

- To inspect logs: `docker compose logs -f api` or `docker compose logs -f postgres`.
- To run database commands or connect via psql: `psql postgres://tcgdb:tcgdb@localhost:5432/tcgdb`.
- The Compose file is purposely minimal. Add volumes or tooling overrides in a local override file if needed (`docker-compose.override.yml`).

## Deploying to Cloud Run (or similar)

- This should be used from CI/CD pipelines.
- Build the production image from `apps/api/Dockerfile`:
  ```bash
  docker build -t gcr.io/<project>/tcgdb-api:latest -f apps/api/Dockerfile .
  ```
- Configure environment variables in the deployment platform (Cloud Run service configuration) — **do not** rely on `.env.dockercompose`.
- Set `AUTH_PROVIDER=firebase` and supply `FIREBASE_CONFIG` / `GCLOUD_PROJECT`. Ensure secrets are handled securely (Secret Manager or equivalent).

## Why Compose is dev-only

- No TLS termination, scaling, high availability (HA) or production hardening.
- Credentials are stored in a local env file for convenience.
- Authentication defaults to unsigned tokens (`dev` mode). Production must run with real Firebase verification.

Keep these goals in mind as you use the Docker assets: Compose is your local playground, while the Dockerfile is your deployable artifact.
