# Multitenancy – Low‑Level Design (current implementation)

## Scope
This document captures the backend LLD for multi‑tenant routing and storage as implemented today. It focuses on tenant registry, tenant-space resolution, database routing, and domain adapters that are already wired (entities, users). Provisioning is currently stubbed; open items are listed at the end.

## Core concepts (implemented)
- **Tenant Space** (`platform/go/tenant.Space`): `{tenantId, slug, shortTenantId, schemaName, basePrefix}`.  
  - `schemaName = "tenant_" + snake_case(slug)`.  
  - `basePrefix = <envKey>/<slug>-<shortTenantId>/` (bucket comes from env, not stored per tenant).  
- **Admin schema**: derived from `ADMIN_TENANT_SLUG` (default `admin`) as `tenant_<slugSnake>`. The bootstrap CLI command initializes this schema and the base tables (see `apps/cli/cmd/bootstrap`).
- **Tenant registry**: immutable, versioned rows in `tenants` table (admin schema) defined in `database/schema/tenants.sql`. Active version enforced by partial index; slug uniqueness enforced across non-deleted rows.
- **Tenant middleware** (`platform/go/tenant/middleware/tenant_space.go`): after auth, extracts tenant claim, resolves via tenant service, enforces `basePrefix` envKey prefix, caches (TTL optional), and attaches `tenant.Space` to context; on failure emits ProblemDetails (401/403).

## Persistence routing
- **TenantDB** (`platform/go/persistence/tenant_db.go`): wraps `pgxpool`; `WithTenant(ctx, space, fn)` starts a tx, sets `search_path` to `<tenant schema>,<admin schema>`, executes `fn(tx)`, commits/rolls back. Admin schema passed via config; tenant schema comes from `tenant.Space`.

## Tenant registry persistence
- **Store** (`platform/go/persistence/tenant_repository.go`): append-only versions with fields `{tenant_id, tenant_version, slug, display_name, status, schema_name, base_prefix, short_tenant_id, created_at, created_by, db_ready, auth_ready, last_provisioned_at, last_error, is_active, is_soft_deleted}`.  
- **Service** (`domains/tenants/be/service`): CRUD over immutable versions; resolves `tenant.Space` for middleware; provisioning endpoints currently return `ErrNotImplemented` (see Open items).

## Tenant-scoped domains (implemented)
- **Entities**
  - Repo (`platform/go/persistence/entity_repository.go`): all methods take `tenant.Space` and run via `TenantDB.WithTenant`; DDL (entity table + indexes) executed lazily inside the tenant schema on first use. `search_path` keeps FK to admin `schema_repository`.
  - Domain repo (`domains/entities/be/repo`): extracts `tenant.Space` from context, forwards to persistence repo.
  - Isolation test (`platform/go/persistence/entity_repository_test.go`): two schemas, verifies writes and reads stay within their schema; cross-tenant get returns not-found.
- **Users**
  - Store (`platform/go/persistence/user_repository.go`): same pattern as entities; `users` table ensured lazily per tenant schema via `TenantDB.WithTenant`.
  - Domain repo (`domains/users/be/repo`): pulls `tenant.Space` from context; services/handlers unchanged.
  - Isolation test (`platform/go/persistence/user_repository_test.go`): two schemas, verifies separation and cross-tenant not-found.

## API wiring
- `apps/api/main.go` builds one `TenantDB` with `AdminSchema` derived from `ADMIN_TENANT_SLUG` and injects into entity and user repos.
- Middleware order: auth → request trace → tenant space. Tenants endpoints remain admin-only.

## Environment variables (used today)
- Core routing
  - `ENV_KEY` (required): leading segment for `basePrefix`; enforced by middleware and provisioning.
  - `ADMIN_TENANT_SLUG` (default `admin`): seeds the admin schema name `tenant_<slugSnake>`; the API derives the admin schema from this value.
- Database
  - `DATABASE_URL` (required), `TEST_DATABASE_URL` (for integration tests).
- Storage
  - `STORAGE_BACKEND` (`gcs`|`local`, default `gcs`).
  - `STORAGE_BUCKET` (required when backend=`gcs`; one bucket per environment class).
  - `STORAGE_LOCAL_DIR` (root path when backend=`local`; default `./.data/storage`).
  - (deprecated) `GCS_ASSETS_BUCKET` was the prior bucket env; use `STORAGE_BUCKET` instead.
- Auth
  - `AUTH_PROVIDER` (`firebase`|`dev`, default `firebase`).
  - `FIREBASE_CONFIG` (optional path to service account JSON; ADC used when absent).
  - `AUTH_TENANT_PREFIX` (optional override for external auth tenant names; defaults to `ENV_KEY` when empty).


## Auth alignment (current)
- JWT extractor now requires a tenant claim (`firebase.tenant` in dev/prod tokens). The API config builds a custom extractor that:
  - Accepts a UUID tenant claim directly, or
  - Resolves an external tenant key of the form `<envKey>-<slug>` via the tenant service, rejecting env-key mismatches or disabled/unknown tenants.
  - Writes the internal tenant UUID into `UserCredentials.TenantID`; tokens without a tenant are rejected.
- Tenant middleware still validates `basePrefix` envKey and returns ProblemDetails: 401 invalid tenant, 403 env mismatch/disabled/unknown.

## Bootstrapping & DDL
- Phase 1 (platform bootstrap) via `platform-cli bootstrap platform ...`:
  - Creates **admin schema only** and applies embedded DDL: `database/schema/tenant_space/users.sql`, `database/schema/platform/entity_schemas.sql`, `database/schema/platform/tenants.sql`.
  - Seeds admin tenant/user. No tenant roles are created here.
- Phase 2 (per-tenant bootstrap) is **only** in `DBProvisioner`:
  - Creates tenant NOLOGIN role, grants membership to the app role, creates tenant schema owned by that role.
  - Grants USAGE on admin schema plus SELECT on `schema_repository` and `schema_categories`.
  - Applies default privileges (tables/sequences) while scoped with `SET LOCAL ROLE`.
  - Creates tenant `users` table using the embedded asset (same as Phase 1) inside the tenant transaction so ownership stays with the tenant role.
  - Leaves entity tables lazy (created at runtime by repositories).

## Testing
- Integration (Testcontainers) for tenant registry, entities, users; skip when `testing.Short()` or `TEST_DATABASE_URL` unset.
- Unit tests for middleware cache and tenant helpers unchanged.

## Provisioning flow (implemented for DB, planned for auth/storage)
- Where we stand: tenant registry and tenant.Space middleware exist; `TenantDB.WithTenant` sets both `SET LOCAL ROLE roleName` and `search_path`. DB provisioner is active; auth/storage are still stubbed.
- Target behaviour (`POST /admin/tenants/{id}:provision`): end in `active` with `dbReady && authReady == true`, `lastProvisionedAt` set, `lastError` cleared; idempotent; accepts `pending|provisioning|active` (not `disabled`).
- Happy-path steps:
  1. Fetch + lock tenant; set status `provisioning`, clear `lastError`, bump version.
  2. Derive names: `schemaName=tenant_<slug_snake>`, `roleName=tenant_<slug_snake>_role`, `basePrefix=<ENV_KEY>/<slug>-<shortId>/`, `externalAuthTenant=<ENV_KEY>-<slug>`.
  3. Database (DBProvisioner only): ensure NOLOGIN `roleName`, grant it to app DB user; create schema owned by `roleName`; set default privileges in the same transaction with `SET LOCAL ROLE` + `search_path`; grant USAGE on admin schema and SELECT on `schema_repository`/`schema_categories`; create tenant `users` table from embedded DDL under tenant role; **entity tables remain runtime/lazy** and inherit ownership via default privs.
  4. Auth: ensure external auth tenant exists with envKey guard; mark `authReady`. (pending real impl)
  5. Storage: verify configured bucket/prefix (GCS/local); for GCS, write/delete sentinel under `basePrefix`. (pending real impl)
  6. Commit: if both ready → `status=active` else `provisioning`; set `lastProvisionedAt`, clear `lastError`, bump `tenant_version`.
- Failure handling: keep achieved flags, store `lastError`, status `pending` if nothing ready else `provisioning`; retries re-validate resources.
- Provision status (`GET ...:provision-status`): live-check role/grants/schema/base tables, auth tenant, GCS prefix; persist flag changes; promote to `active` when both ready.
- Runtime invariant: `TenantDB.WithTenant` must execute `SET LOCAL ROLE roleName` and `SET LOCAL search_path = schemaName,<admin_schema>` for every tenant-scoped txn; lazy ensure paths must run under the tenant role so new tables inherit ownership.
- The tenant registry stores `role_name`; runtime uses the stored value (no derivation). Keep DB and `tenant.Space` in sync; missing/empty `role_name` is an error.

## Current limitations / open items
- Provisioning workflow remains unimplemented; service returns `ErrNotImplemented` until wired to steps above.
- Entity/user tables are created lazily; decision pending on moving DDL into provisioning.
- No explicit test yet to assert FK to admin `schema_repository` under `search_path`.
- JWT tenant-claim mapping and envKey enforcement rely on existing middleware; keep aligned as auth evolves.

## Summary
Data isolation is enforced via per-request `search_path` and tenant-scoped repos for entities and users. Tenant registry is immutable/versioned in the admin schema, and middleware resolves tenant space with envKey guard. Provisioning is planned as above; endpoints intentionally return “not implemented” until implemented end-to-end.
