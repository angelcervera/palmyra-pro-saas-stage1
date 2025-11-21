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

- **Tenant**  
  A logical customer / organization using Palmyra Pro.

- **Slug**  
  A stable, name‑ and URL‑friendly identifier in `kebab-case`, matching the
  pattern `^[a-z0-9]+(?:-[a-z0-9]+)*$`. Slugs are used for human‑readable
  identifiers (for example, tenant slugs or schema slugs) and can be
  transformed into other forms (such as `snake_case`) when needed for
  infrastructure names (schemas, queues, etc.).

- **Tenant Space**  
  The unit of isolation for a tenant. It combines:
  - One PostgreSQL schema dedicated to that tenant.
  - A logical namespace inside a shared GCS bucket, identified by a
    tenant-specific base prefix.
  - Additional configuration and flags as needed.

- **Admin Schema**  
  A configurable PostgreSQL schema that stores platform-global data such as:
  - Schema Repository (versioned JSON Schemas).
  - Schema Categories.
  - Tenant registry and configuration.
  - Other shared/admin tables as the platform evolves.

## High-Level Architecture

At a high level, the system distinguishes between:

- **Platform-global data (Admin Schema)**  
  - Schema definitions and categories.  
  - Tenant registry and configuration (mapping tenant → DB schema, tenant
    base prefix in GCS, status, etc.; the physical bucket is typically shared
    across tenants).  
  - Other administrative / cross-tenant metadata.

- **Tenant-local data (Tenant Spaces)**  
  - Tenant-specific entity tables (document storage) in the tenant’s own
    PostgreSQL schema.  
  - Tenant-specific users and other per-tenant tables.  
  - Tenant-specific binary objects stored in the tenant’s GCS bucket/prefix.

The Admin Schema name is configurable via the `DB_ADMIN_SCHEMA` environment
variable (default suggestion: `palmyra_admin`) and loaded via `envconfig`, in
line with the existing backend configuration approach.

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

- For each tenant:
  - Use the environment’s GCS bucket configured via the `GCS_ASSETS_BUCKET`
    environment variable (one bucket per environment, for example
    `palmyra-dev-assets`, `palmyra-prod-assets`). Different environments may
    use different GCP projects as needed, but sometimes, we will share the same
    GCP project between different environments (like temporal PRs deployments).
  - Assign a **tenant base prefix** derived from the stable `tenantId`,
    exactly `<tenantId>/`. Using the immutable `tenantId` ensures
    collision-free prefixes, supports even distribution/sharding of objects,
    and remains stable even if tenant slugs or display names change.
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

- Tenant identity (id/slug).
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

1. Inserts a tenant record into the Admin Schema’s tenant registry (id/slug,
   initial status, etc.).
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
