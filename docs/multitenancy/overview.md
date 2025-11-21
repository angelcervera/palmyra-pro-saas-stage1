# Multitenancy Overview – Tenant Spaces

This document summarizes the initial multitenancy design for Palmyra Pro.
It captures the high-level model and terminology so that future iterations
can refine contracts, middleware, and operational details.

## Goals

- Provide strong data isolation per tenant while keeping a single monolithic
  backend and shared infrastructure.
- Preserve a central, governed catalog of schemas and categories for the
  whole platform.
- Align with existing conventions: OpenAPI-first, `pgxpool` for PostgreSQL,
  `envconfig` for configuration, and Google Cloud Storage (GCS) for binary
  assets.

## Key Concepts

- **Slug**  
  A stable, name‑ and URL‑friendly identifier in `kebab-case`, matching the
  pattern `^[a-z0-9]+(?:-[a-z0-9]+)*$`. Slugs are used for human‑readable
  identifiers (for example, tenant slugs or schema slugs) and can be
  transformed into other forms (such as `snake_case`) when needed for
  infrastructure names (schemas, queues, etc.).

- **Tenant**  
  A logical customer / organization using Palmyra Pro.

- **tenantId (Palmyra Tenant ID)**  
  A platform-generated, stable identifier for a tenant. In the current
  context, this is a UUID created by the Palmyra Pro Tenant Admin when a
  tenant is provisioned. It is independent of any identifiers used by the
  underlying authentication provider (for example, Firebase/Identity Platform
  tenant IDs) and acts as the primary key for:
  - locating the Tenant Space (PostgreSQL schema + GCS namespace),
  - deriving the tenant’s base GCS prefix together with the slug
    (see below), and
  - referencing the tenant from other parts of the platform.  
  
  Each tenantId is associated with additional metadata such as a human-facing
  `slug`, display name, contact email, and status in the tenants registry.  
  
  A short, deterministic fragment of `tenantId` (for example, the first 8 hex
  characters of the UUID without dashes) is always used in GCS prefixes to
  guarantee uniqueness even if slugs are similar or change over time, and to allow even distribution on sharding.

- **Tenant Space**  
  The unit of isolation for a tenant. It combines:
  - One PostgreSQL schema dedicated to that tenant.
  - A logical namespace inside a shared GCS bucket, identified by a
    tenant-specific base prefix.
  - Additional configuration and flags as needed.

- **Admin Space**  
  A special, platform-owned Tenant Space whose tenant metadata (including
  `slug`) is configured via environment variable (for example,
  `ADMIN_TENANT_SLUG`, defaulting to `"admin"`). It uses:
  - The same PostgreSQL schema naming convention as any other tenant
    (`tenant_<slugSnake>`), where the admin schema is derived from the
    configured admin slug.
  - The shared assets bucket for the environment (from `GCS_ASSETS_BUCKET`),
    using the same `<tenantSlug>-<shortTenantId>/` prefix pattern as other
    Tenant Spaces; for the Admin Space this means a base prefix of
    `<adminSlug>-<shortAdminTenantId>/`.
  - Additional control-plane tables and configuration that apply across all
    tenants.

## High-Level Architecture

At a high level, the system distinguishes between:

- **Platform-global data (Admin Space)**  
  - Schema definitions and categories.  
  - Tenant registry and configuration (mapping tenant → DB schema, tenant
    base prefix in GCS, status, etc.; the physical bucket is shared per
    environment).  
  - Other administrative / cross-tenant metadata.

- **Tenant-local data (Tenant Spaces)**  
  - Tenant-specific entity tables (document storage) in the tenant’s own
    PostgreSQL schema.  
  - Tenant-specific users and other per-tenant tables.  
  - Tenant-specific binary objects stored in the tenant’s GCS namespace
    (including the Admin Space).

## Tenant Space – PostgreSQL

- Each tenant is assigned a dedicated **PostgreSQL schema**.
- Tenant schema names follow the pattern `tenant_<slug>`, where `<slug>` is
  derived from the tenant’s canonical slug by normalizing to lowercase
  `snake_case` (alphanumeric plus underscore only), for example
  `tenant_acme_corp`.
- Within that schema the persistent layer provisions:
  - The tenant’s **entity tables**, following the existing document-oriented
    model (immutable versions, JSONB payload, hashes, timestamps, etc.).
  - A per-tenant **users** table.
  - Any additional tenant-local tables.
- Entity tables continue to reference the platform-global Schema Repository via
  `schema_id` and `schema_version`, but now live in the tenant’s schema
  instead of a shared schema.
- The shared persistent layer in `platform/go/persistence` is responsible for
  routing operations to the correct PostgreSQL schema based on the current
  Tenant Space.

## Tenant Space – Binary Storage (GCS)

Binary assets (pictures, documents, attachments) are stored in Google Cloud
Storage using a **single shared bucket per environment with one dedicated
prefix per tenant**:

- For each tenant (including the Admin Space):
  - Use the environment’s GCS bucket configured via the `GCS_ASSETS_BUCKET`
    environment variable (one bucket per environment, for example
    `palmyra-dev-assets`, `palmyra-prod-assets`). Different environments may
    use different GCP projects as needed, but in some cases environments may
    share a project (for example, temporary PR deployments).
  - Assign a **tenant base prefix** derived from the tenant’s `slug` and
    stable `tenantId`, in the form `<tenantSlug>-<shortTenantId>/`. Using the
    immutable `tenantId` fragment alongside the human‑readable slug ensures
    collision-free prefixes, keeps paths debuggable, and remains stable even
    if display names change.
- Inside that tenant base prefix, logical subpaths can group content by purpose,
  for example:
  - `entities/<entityId>/<attachmentId>`
  - `avatars/<userId>`
  - `documents/...`
- The tenant’s GCS configuration (bucket name and base prefix) is stored in
  the admin `tenants` table and exposed via the Tenant Space abstraction.

Only a logical key (e.g. `entities/<entityId>/<attachmentId>`) should be
referenced from domain code. The combination of Tenant Space + logical key is
what determines the actual GCS location `(bucket, tenantBasePrefix + key)`.

## TenantSpace Abstraction

Tenant Space is represented conceptually as a small, runtime object resolved
once per request and reused throughout the stack. It includes:

- Tenant identity (internal `tenantId` plus associated `slug`).
- PostgreSQL schema name for that tenant.
- GCS bucket name (typically shared across tenants) and the tenant’s base
  prefix within that bucket.
- Optional feature flags, limits, and other metadata.

### Resolution Flow (Conceptual)

1. Authentication middleware validates the JWT and extracts a tenant
   identifier (for example, from a dedicated claim).
2. A lookup in the Admin Schema’s tenant registry maps that identifier to a
   Tenant Space (schema name, bucket, prefix, status).
3. The Tenant Space is attached to the request context.
4. Domain services and persistence layer functions read Tenant Space from the
   context instead of manually choosing schema names or buckets.

## Request Handling and Data Routing

- Handlers remain thin and contract-driven, using the existing patterns:
  - Use generated OpenAPI types.
  - Rely on shared middleware for auth and Tenant Space resolution.
- The persistent layer:
  - Uses Tenant Space to select the effective PostgreSQL schema for each
    operation (e.g. via search_path or explicit schema-qualified names).
  - Provides APIs to domain repos that are already scoped to “the current
    tenant space”.
- A shared blob-storage abstraction in `platform/go` handles GCS operations,
  always parameterized by Tenant Space plus a logical key.

## Tenant Lifecycle (Conceptual)

### Provisioning a Tenant Space

When a new tenant is created, the system:

1. Inserts a tenant record into the Admin Space’s tenant registry (id/slug,
   initial status, etc.). The Admin Space itself is represented by a reserved
   admin-tenant record whose `slug` is provided by configuration (for example,
   `ADMIN_TENANT_SLUG`, default `"admin"`); its `tenantId` is a regular
   Palmyra Tenant ID stored in the same registry.
2. Creates a dedicated PostgreSQL schema for that tenant and provisions base
   tables/indexes.
3. Ensures the shared GCS bucket exists and assigns a tenant base prefix
   within that bucket.
4. Writes the resulting schema name, bucket name, and tenant base prefix into the tenant
   record.

### Runtime Usage

- For every tenant-scoped request:
  - Tenant Space is resolved from JWT claims and the tenant registry.
  - Database operations execute in the tenant’s schema.
  - Blob operations target the shared bucket using the tenant’s base prefix.

### Deactivation / Archival (Later Iteration)

- Mark tenant as disabled in the tenant registry and reject new requests.
- Apply retention policies or lifecycle rules to objects under the tenant’s
  base prefix in the shared GCS bucket.
- Archive or drop the tenant’s PostgreSQL schema depending on business and
  compliance requirements.

## Separation of Concerns

- **Admin Schema & Tenant Registry**
  - Owns platform-global concepts: schemas, categories, and tenants.
  - Provides a single source of truth for mapping tenant IDs to Tenant Spaces.

- **Tenant Space (Runtime)**
  - Encapsulates all tenant-specific routing data (DB schema, GCS bucket,
    prefix, flags).
  - Is resolved once per request and attached to the context.

- **Persistent Layer & Storage Abstractions**
  - Hide multitenancy plumbing from domain handlers and services.
  - Provide tenant-aware APIs for both PostgreSQL and GCS.

## Next Steps and Open Questions

This document is intentionally high-level. Future iterations will define:

- Exact JWT claims and mapping logic used to resolve Tenant Space.
- Detailed DDL patterns for per-tenant schemas and tables.
- How and when per-tenant migrations or table provisioning occur
  (provision-time vs. lazy).
- IAM, encryption, and logging strategies for the shared GCS bucket and the
  per-tenant prefixes within it.
- Quotas, rate limits, and monitoring per tenant.
- APIs and UI flows for onboarding, suspending, and deleting tenants.
