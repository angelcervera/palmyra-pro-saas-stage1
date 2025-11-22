# Multitenancy – Low‑Level Design (current implementation)

## Scope
This document captures the backend LLD for multi‑tenant routing and storage as implemented today. It focuses on tenant registry, tenant-space resolution, database routing, and domain adapters that are already wired (entities, users). Provisioning is currently stubbed; open items are listed at the end.

## Core concepts (implemented)
- **Tenant Space** (`platform/go/tenant.Space`): `{tenantId, slug, shortTenantId, schemaName, basePrefix}`.  
  - `schemaName = "tenant_" + snake_case(slug)`.  
  - `basePrefix = <envKey>/<slug>-<shortTenantId>/` (bucket comes from env, not stored per tenant).  
- **Admin schema**: derived from `ADMIN_TENANT_SLUG` (default `admin`) as `tenant_<slugSnake>`. `database/000_init_schema_and_seeds.sh` sets DB `search_path` to this schema at bootstrap.
- **Tenant registry**: immutable, versioned rows in `tenants` table (admin schema) defined in `database/schema/002_tenants_schema.sql`. Active version enforced by partial index; slug uniqueness enforced across non-deleted rows.
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
- `apps/api/main.go` builds one `TenantDB` with `AdminSchema` from env (`TENANT_SCHEMA` / derived admin schema) and injects into entity and user repos.
- Middleware order: auth → request trace → tenant space. Tenants endpoints remain admin-only.

## Environment variables (used today)
- `ADMIN_TENANT_SLUG` (default `admin`): defines admin schema name (`tenant_<slugSnake>`).
- `TENANT_SCHEMA` (optional override for admin schema in API config; defaults to admin).
- `ENV_KEY` (required): leading segment for `basePrefix`; enforced by middleware.
- `GCS_ASSETS_BUCKET` (bucket per environment class); not stored per tenant.
- `DATABASE_URL`, `TEST_DATABASE_URL` (tests), `PLATFORM_DB_SEED_MODE` (init script seeds).

## Auth alignment (current)
- JWT extractor now requires a tenant claim (`firebase.tenant` in dev/prod tokens). The API config builds a custom extractor that:
  - Accepts a UUID tenant claim directly, or
  - Resolves an external tenant key of the form `<envKey>-<slug>` via the tenant service, rejecting env-key mismatches or disabled/unknown tenants.
  - Writes the internal tenant UUID into `UserCredentials.TenantID`; tokens without a tenant are rejected.
- Tenant middleware still validates `basePrefix` envKey and returns ProblemDetails: 401 invalid tenant, 403 env mismatch/disabled/unknown.

## Bootstrapping & DDL
- Base schemas/tables: `database/schema/001_core_schema.sql`.
- Tenant registry DDL: `database/schema/002_tenants_schema.sql`.
- Init script: `database/000_init_schema_and_seeds.sh` creates admin schema from `ADMIN_TENANT_SLUG`, sets database search_path, applies ordered schema SQL, optional dev seeds.

## Testing
- Integration (Testcontainers) for tenant registry, entities, users; skip when `testing.Short()` or `TEST_DATABASE_URL` unset.
- Unit tests for middleware cache and tenant helpers unchanged.

## Provisioning flow (planned, with tenant roles)
- Where we stand: tenant registry and tenant.Space middleware exist; `TenantDB.WithTenant` currently only sets `search_path`; provisioning endpoints are stubbed.
- Recent changes: tenant Space now carries `roleName`; `TenantDB.WithTenant` sets both `SET LOCAL ROLE roleName` and `search_path`; service wiring uses provisioners (DB roles/schema/grants, auth, storage) plus advisory lock.
- Target behaviour (`POST /admin/tenants/{id}:provision`): end in `active` with `dbReady && authReady == true`, `lastProvisionedAt` set, `lastError` cleared; idempotent; accepts `pending|provisioning|active` (not `disabled`).
- Happy-path steps:
  1. Fetch + lock tenant; set status `provisioning`, clear `lastError`, bump version.
  2. Derive names: `schemaName=tenant_<slug_snake>`, `roleName=tenant_<slug_snake>_role`, `basePrefix=<ENV_KEY>/<slug>-<shortId>/`, `externalAuthTenant=<ENV_KEY>-<slug>`.
  3. Database: ensure NOLOGIN `roleName`, grant it to app DB user; create schema owned by `roleName`; default privileges to `roleName`; read-only grants to admin schema (`USAGE`) and `schema_repository` (`SELECT`); run base DDL for shared tenant tables (e.g., `users`) under `SET ROLE roleName`; **entity tables remain runtime/lazy** because new schemas can be added later—default privileges ensure those tables will be owned by the tenant role when created.
  4. Auth: ensure external auth tenant exists with envKey guard; mark `authReady`.
  5. Storage: verify `GCS_ASSETS_BUCKET`, optional sentinel under `basePrefix`.
  6. Commit: if both ready → `status=active` else `provisioning`; set `lastProvisionedAt`, clear `lastError`, bump `tenant_version`.
- Failure handling: keep achieved flags, store `lastError`, status `pending` if nothing ready else `provisioning`; retries re-validate resources.
- Provision status (`GET ...:provision-status`): live-check role/grants/schema/base tables, auth tenant, GCS prefix; persist flag changes; promote to `active` when both ready.
- Runtime invariant: `TenantDB.WithTenant` must execute `SET LOCAL ROLE roleName` and `SET LOCAL search_path = schemaName,<admin_schema>` for every tenant-scoped txn; lazy ensure paths must run under the tenant role so new tables inherit ownership.
  - Because schemas can be added over time in the schema repository, entity tables are created on-demand at runtime; default privileges from provisioning ensure those tables land with the tenant role and stay isolated.

## Current limitations / open items
- Provisioning workflow remains unimplemented; service returns `ErrNotImplemented` until wired to steps above.
- Entity/user tables are created lazily; decision pending on moving DDL into provisioning.
- No explicit test yet to assert FK to admin `schema_repository` under `search_path`.
- JWT tenant-claim mapping and envKey enforcement rely on existing middleware; keep aligned as auth evolves.

## Summary
Data isolation is enforced via per-request `search_path` and tenant-scoped repos for entities and users. Tenant registry is immutable/versioned in the admin schema, and middleware resolves tenant space with envKey guard. Provisioning is planned as above; endpoints intentionally return “not implemented” until implemented end-to-end.
