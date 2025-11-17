# ZenGate Global — Palmyra Pro SaaS

A contract‑first, domain‑oriented platform for managing versioned JSON Schemas and the entity data that depends on them. This repository is the monorepo that hosts the OpenAPI contracts, generated SDKs, backend (Go/Chi), and frontend (React 19 + Vite PWA) for ZenGate Global’s Palmyra Pro data platform.

This README consolidates the essentials to understand the project, navigate the repository, and contribute effectively.

## Detailed Documentation

- [Project Requirements](docs/project-requirements-document.md) — product scope, domains, architecture, and tech stack.
- [Branching & Release Policy](docs/branching-policy.md) — trunk‑based workflow, PR gates, SemVer releases.
- [ADR Index](docs/adr/index.md) — architecture decisions with status and dates.
- [Agent Docs Index](docs/agent-index.yaml) — canonical index for agent‑readable docs and commands.
- [API Conventions](docs/api/api.md) — naming, methods, status codes, pagination, ProblemDetails.
- [API Server Guideline](docs/api/api-server.md) — backend rulebook: contract‑first, routing, auth, testing, codegen.
- [Persistence Layer](docs/persistence-layer/persistent-layer.md) — document‑oriented persistence on PostgreSQL with schema versioning.
- [Web App Guide](docs/web-app.md) — React 19 admin shell, build steps, SDK usage.
- [Auth Testing Playbook](docs/auth-testing.md) — practical steps for exercising dev (unsigned) and Firebase auth flows.


## Table of Contents

- [Overview & Goals](#overview--goals)
- [Core Features & Scope](#core-features--scope)
- [Architecture at a Glance](#architecture-at-a-glance)
- [Repository Structure](#repository-structure)
- [Tech Stack](#tech-stack)
  - [OpenAPI](#openapi)
  - [Frontend](#frontend)
  - [Backend](#backend)
- [Design Guidelines (UI/UX)](#design-guidelines-uiux)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Current status](#current-status)
  - [Running the API server](#running-the-api-server)
- [Development Workflow (contract‑first)](#development-workflow-contract-first)
- [Roadmap](#roadmap)
- [Contributing](#contributing)


## Overview & Goals

ZenGate Global operates several data‑intensive workflows that require accurate, well‑governed structured data across products and teams.

Today, critical datasets are fragmented across systems, with inconsistent schemas, limited validation, and no single source of truth for how entities are defined or stored.

Palmyra Pro provides a contract‑first platform centered on versioned JSON Schemas and the entities that conform to them, giving the organization a governed, continuously updated data backbone.

High‑level goals:

- Own the full data lifecycle (schema definition, ingestion, validation, CRUD, and curation).
- Be contract‑first to keep frontend and backend in sync with zero drift.
- Provide admin tooling for users, schema categories, schema definitions, and entity documents with robust filtering and pagination.


## Core Features & Scope

In scope for the current phase:

- Authentication (Firebase Auth + JWT)
- Authorization with roles: admin, user_manager
- User sign‑up and approval flow (pending → approve/reject; enable/disable)
- Admin user management including CRUD, filtering, and pagination
- Admin CRUD for Schema Categories (organize schemas into a hierarchy)
- Schema Repository management for versioned JSON Schemas
- CRUD for JSON entity documents (filtering + pagination) backed by the persistence layer

Out of scope: Everything else not listed above for this phase. See docs/Project Requirements Document.md for the complete narrative.


## Architecture at a Glance

Key principles:

- OpenAPI‑driven / contract‑first: OpenAPI is the single source of truth between frontend and backend.
- Monolithic, domain‑based: Each domain (auth, users, schema-categories, schema-repository, entities) contains its own FE/BE code where applicable.
- Monorepo: One place for contracts, generated code, apps, and platform‑wide utilities.

Benefits:

- Strong modularity and clear boundaries by domain
- Hermetic, incremental builds (future Bazel setup)
- Independent development per domain using generated SDKs


## Repository Structure

The repository is organized vertically by domain and separates generated artifacts from source:

```
/contracts/                      # OpenAPI definitions (one per domain)
  auth.yaml                      # signup/login/refresh/logout + securitySchemes
  users.yaml                     # user CRUD/listing + admin actions
  schema-categories.yaml         # admin CRUD for schema categories hierarchy
  schema-repository.yaml         # schema definitions and versions
  entities.yaml                  # JSON entity documents per schema/table
  common/                        # Shared components (Pagination, ProblemDetails, IAM, primitives)

/domains/
  auth/
    be/                          # Go handlers, services, repos, tests
  users/
    be/
  schema-categories/
    fe/                          # Tree UI for categories (create/update/delete)
    be/                          # Handler/Service/Repo/tests for /api/v1/schema-categories
  schema-repository/
    be/                          # Handler/Service/Repo/tests for schema repository endpoints
  entities/
    be/                          # Handler/Service/Repo/tests for entity documents

/generated/                      # Codegen output (read‑only)
  go/
    auth/
    users/
    schema-categories/
    schema-repository/
    entities/

/packages/api-sdk/src/generated/ # TypeScript SDK clients and types
  auth/
  users/
  schema-categories/
  schema-repository/
  entities/

/platform/
  ts/                            # Shared frontend utilities, UI, form & auth helpers
  go/
    logging/
    middleware/
    auth/
    persistence/                 # Shared persistence (pgxpool, schema repository, schema categories store)

/apps/
  web-admin/                     # React 19 + Vite admin application shell
  api/                           # Go API entrypoint (Chi router) mounting domain routes

/docs/
  api.md                         # API conventions
  project-requirements-document.md
```

Note: Some paths above are planned as the project evolves; not all may be present yet in the repository.


## Tech Stack

- Monorepo and build system: Bazel 8 (bzlmod) — planned; enables hermetic, cached, reproducible builds
- Contracts: OpenAPI 3.0.4+
- Frontend: React 19 + Vite, TypeScript 5, pnpm, ShadCN UI (Tailwind), Radix UI, React Hook Form, Zod, React Router, i18n (react‑i18next), date‑fns v4; testing with Vitest, RTL, Cypress
- Backend: Go 1.25+, Chi router

See docs/Project Requirements Document.md for rationale and deeper details.


### OpenAPI

- TS client codegen: run `pnpm openapi:ts` (uses `tools/codegen/openapi/ts/openapi-ts.config.ts`). For details, see the “Contracts → TypeScript Codegen” section in `docs/web-app.md`.

- The OpenAPI files are modular by domain and reuse shared components from `/contracts/common/`.
- Generated Go stubs go under `/generated/go/<domain>`. TypeScript clients are emitted inside the SDK at `packages/api-sdk/src/generated/<domain>` and consumed via the `@zengateglobal/api-sdk` package.
- Error responses use ProblemDetails (RFC 7807). Collection endpoints use a standardized Pagination model.

New domain highlights:
- `contracts/schema-categories.yaml` defines `/api/v1/schema-categories` for admin CRUD on the categories tree.
- Generated Go server interfaces under `/generated/go/schema-categories` are implemented by `domains/schema-categories/be`.

API conventions and patterns: docs/API.md


### Frontend

- React 19 SPA/PWA with Vite. Offline‑first using VitePWA (Workbox) for runtime caching, background sync, and auto‑updates.
- UI via ShadCN + Tailwind; forms with React Hook Form + Zod; state with Nanostores; routing via React Router.


### Backend

- Go 1.25+, monolithic by domain with Chi for routing; generated models and handlers from OpenAPI via oapi-codegen.
- Shared persistence in `platform/go/persistence` provides:
  - `schema_repository` management (versioned JSON Schemas for entities)
  - `schema_categories` store (hierarchical categories linked from schema repository)


## Design Guidelines (UI/UX)

Follow ShadCN dashboard patterns and Tailwind conventions. Ensure accessibility (a11y) and consistent variants, focus states, and responsive behavior (collapsible sidebar, mobile‑first layouts). See the “Design Guidelines” section in docs/Project Requirements Document.md for specifics.


## Getting Started

### Prerequisites

- Node.js 24 and pnpm 10.x (>=10.20.0 <11) (for frontend and tooling)
- Go 1.25+ (for backend)
- Bazel 8 (planned; not required until build rules are introduced)

### Current status

This repository currently emphasizes documentation and contract‑first planning. As domains and contracts are added, code generation and app scaffolding will be introduced alongside Bazel build targets.

### Running the API server

The scaffolded Chi server lives under `apps/api`. It relies on a few environment variables (loaded via `envconfig`) before starting:

| Variable           | Default    | Purpose                                                                    |
|--------------------|------------|----------------------------------------------------------------------------|
| `PORT`             | `3000`     | TCP port where the HTTP server listens                                     |
| `SHUTDOWN_TIMEOUT` | `10s`      | Graceful shutdown timeout when SIGINT is received                          |
| `REQUEST_TIMEOUT`  | `15s`      | Per-request timeout enforced by Chi’s timeout middleware                   |
| `FIREBASE_CONFIG`  | _empty_    | Optional path to a Firebase service-account JSON for local development     |
| `GCLOUD_PROJECT`   | _empty_    | Optional Firebase/GCP project ID (required if not embedded in credentials) |
| `LOG_LEVEL`        | `info`     | Minimum zap severity (`debug`, `info`, `warn`, `error`)                    |
| `DATABASE_URL`     | _none_     | PostgreSQL connection string used by the persistence layer                 |
| `AUTH_PROVIDER`    | `firebase` | Auth backend (`firebase` or `dev`)                                         |

If `FIREBASE_CONFIG` is omitted, the Firebase SDK will fall back to default credentials (e.g., ADC on GCP).

Example (Linux/macOS) for running the server on port 8080 with an explicit Firebase credential:

```bash
PORT=8080 \
REQUEST_TIMEOUT=20s \
SHUTDOWN_TIMEOUT=15s \
DATABASE_URL=postgres://palmyra:palmyra@localhost:5432/palmyra?sslmode=disable \
FIREBASE_CONFIG=/home/angelcc/.ssh/palmyra-dev-firebase-adminsdk.json \
go run ./apps/api
```

### Docker (API + Postgres)

You can boot the backend stack locally with Docker Compose (edit `.env.dockercompose` if you need different credentials or API settings):

```bash
docker compose up --build
```

Services:

- **postgres** – `postgres:16` listening on port `5432`. Credentials default to `palmyra`/`palmyra` with database `palmyra`. Data persists in the `postgres-data` volume.
- **api** – Go server on port `3000`, automatically pointing to the Postgres container via `DATABASE_URL`.

Environment defaults live in `.env.dockercompose`; adjust values there before running `docker compose` if you need different ports or credentials.

Optional Firebase integration:

- Mount your service-account JSON into the container (e.g. `./firebase/service-account.json:/app/firebase/service-account.json:ro`) and set `FIREBASE_CONFIG=/app/firebase/service-account.json` in `docker-compose.yml` (see commented examples).
- Provide `GCLOUD_PROJECT` if your credentials require a project id.

When running with `AUTH_PROVIDER=dev`, supply a bearer token containing unsigned JWT claims. The middleware does **not** verify signatures; instead it copies fields such as `email`, `name`, and `isAdmin` from the token payload. Set `isAdmin: true` in the JWT to simulate admin access.

With the containers running, the admin frontend can target the API by setting `VITE_API_BASE_URL=http://localhost:3000/api/v1` and running `pnpm dev -C apps/web-admin`.

See also [`docs/docker.md`](docs/docker.md) for a deeper explanation of the Docker setup and deployment considerations.


## Development Workflow (contract‑first)

- Start from the OpenAPI contract. The contract is the source of truth.
- Generate SDKs and stubs per domain.
- Implement backend services and frontend pages against the generated types.
- Keep contracts and implementations in lock‑step; regenerate on contract changes.

For schema categories specifically:
- Update `contracts/schema-categories.yaml` first, then run codegen (`go generate ./tools/codegen/openapi/go`).
- Implement or adjust handlers/services in `domains/schema-categories/be` and wire routes in `apps/api`.

See docs/API.md for naming, versioning, status codes, pagination, filtering, and security conventions.


## Roadmap

- Initialize contracts and shared components (Pagination, ProblemDetails)
- Introduce codegen pipelines for Go and TypeScript
- Scaffold domains: auth, users, schema-categories, schema-repository, entities
- Implement authentication + user approval flows
- CRUD for schema categories, schemas, and entity documents with filtering/pagination
- Add Bazel build and CI caching
- Ship PWA shell with domain pages and standardized UI


## Contributing

- Propose changes to contracts first; discuss breaking changes early.
- Keep domain boundaries clear; prefer shared utilities under `/platform` when cross‑cutting.
- Follow API conventions in docs/API.md and UI guidelines in docs/Project Requirements Document.md.
