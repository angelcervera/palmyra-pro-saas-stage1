# Multitenancy Security Layers

This note captures the security controls that enforce tenant isolation across the stack. The two layers described here are:

- [OpenAPI / REST security](#openapi--rest-security-layer)
- [Database isolation](#database-isolation-per-tenant-roles)

Use this as the baseline when refining provisioning, runtime routing, and audits.

---

## OpenAPI / REST Security Layer

### AuthN
- Global `bearerAuth` (JWT) from `contracts/common/security.yaml` applies to all paths except signup/login.
- JWT verification provided by `platform/go/auth` with providers:
  - `firebase` (real signature checks) or `dev` (unsigned for local).

### Tenant resolution
- Middleware order: Auth → Tenant Space.
- Tenant middleware extracts tenant key from JWT (`firebase.tenant` or explicit tenant UUID), validates `envKey` prefix, looks up the tenant registry, and attaches `tenant.Space { tenantId, slug, shortTenantId, schemaName, basePrefix }` to context.
- Requests without a resolvable tenant receive ProblemDetails 401/403.

### AuthZ
- Domain handlers use `RequireRoles(admin|user_manager)` where specified in contracts. Tenant admin endpoints are admin-only.
- Tenant status guard: disabled tenants return 403 before hitting domain logic.

### Response shape
- Two-response policy per `docs/api.md`: success (200/201/204/202) + default `application/problem+json` using shared `ProblemDetails` schema.

### Interaction with DB isolation
- When the request context contains `tenant.Space`, `SpaceDB.WithSpace` must both set `search_path` and `SET LOCAL ROLE` to the tenant role derived from the space. This ties API-layer tenant resolution to DB-level privileges.

---

## Database Isolation (per-tenant roles)

### Goals
- Prevent cross-tenant data access even if application bugs attempt the wrong schema/table.
- Ensure tenant-scoped transactions run with the least privileges required to read/write that tenant’s data and read-only shared catalog tables.

### Role model
- For tenant slug `acme-co`, derive:
  - **Schema:** `tenant_acme_co`
  - **Runtime role:** `tenant_acme_co_role`
- Roles are **NOLOGIN**; the app pool user is granted membership and must `SET LOCAL ROLE` per request.

### Grants & ownership
- Tenant schema is owned by the tenant role, with:
  - `USAGE` on schema `tenant_<slug>`
  - `ALL` on existing tables/sequences in that schema
  - `ALTER DEFAULT PRIVILEGES IN SCHEMA tenant_<slug> GRANT ALL ON TABLES, SEQUENCES TO tenant_<slug>_role` (future tables inherit)
- Shared read-only access:
  - `GRANT USAGE ON SCHEMA <admin_schema> TO tenant_<slug>_role` (to reach catalog tables)
  - `GRANT SELECT ON <admin_schema>.schema_repository TO tenant_<slug>_role`
  - Extend with other read-only admin tables as they appear.

### Runtime enforcement
- Every tenant-scoped transaction must execute:
  - `SET LOCAL ROLE tenant_<slug>_role;`
  - `SET LOCAL search_path = tenant_<slug>,<admin_schema>;`
- With the role set, accidental references to other schemas fail with `permission denied`, adding a defense-in-depth layer beyond search_path.

---

## Diagram (request path)
1. **JWT auth** → credentials + tenant claim.  
2. **Tenant middleware** → resolves tenant, envKey guard, attaches `tenant.Space`.  
3. **SpaceDB.WithSpace** → `SET LOCAL ROLE tenant_<slug>_role; SET LOCAL search_path = tenant_<slug>,<admin_schema>;`.  
4. **Handlers/Repos** operate; cross-tenant access blocked by role grants.

---

## Open Questions / Decisions
- Do we also grant `SELECT` to other admin lookup tables (e.g., schema categories) for FE-driven reads? Decide per use case.
- How to surface “permission denied” vs. “not found” to callers? For now, surface standard ProblemDetails 403 when repository detects permission errors.
