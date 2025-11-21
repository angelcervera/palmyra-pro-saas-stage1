# Environment Classes & Deployments

Palmyra Pro is typically deployed into a small number of long‑lived
**environment classes** (for example, `prod`, `stg`, `dev`). Within each
class, there may be multiple concrete **deployment environments**, such as:

- the main dev stack (`dev`),
- a shared staging environment (`stg`), and
- ephemeral environments for pull requests (for example, `pr-1234`).

This document explains how storage and identity are organised across these
environments so that multi-tenancy remains consistent and operationally
manageable.

## GCS Buckets and Prefixes

- Each environment class uses one dedicated bucket, for example:
  - `palmyra-prod-assets` for production,
  - `palmyra-stg-assets` for staging,
  - `palmyra-dev-assets` for development and PR environments.
- Within a bucket, every deployment environment is given a distinct
  **environment key** (for example, `prod`, `stg`, `dev`, `pr-1234`), which
  becomes the leading segment of all tenant prefixes:
  - `prod/<tenantSlug>-<shortTenantId>/...`
  - `stg/<tenantSlug>-<shortTenantId>/...`
  - `dev/<tenantSlug>-<shortTenantId>/...`
  - `pr-1234/<tenantSlug>-<shortTenantId>/...`
- Each tenant’s **base prefix** is therefore:
  - `<envKey>/<tenantSlug>-<shortTenantId>/`
  where:
  - `envKey` identifies the deployment environment,
  - `tenantSlug` is the human‑readable slug, and
  - `shortTenantId` is a deterministic fragment of the Palmyra `tenantId`
    (for example, the first 8 hex characters of the UUID).

This layout:

- keeps paths debuggable (tenant slug is visible),
- guarantees uniqueness (short `tenantId` fragment), and
- makes it easy to:
  - locate all objects for a particular deployment, and
  - cleanly delete a PR environment’s data by removing a single top-level
    prefix (for example, all paths under `pr-1234/`).

## Identity Provider Tenants

While the multitenancy design remains authentication‑provider agnostic, the
current implementation with **Google Cloud Identity Platform / Firebase Auth**
follows the same environment naming pattern to keep identity and storage in
sync.

- Each external auth tenant identifier (for example, `firebase.tenant`) is
  constructed with an environment key and a logical tenant slug, such as:
  - `prod-acme-corp`,
  - `stg-acme-corp`,
  - `dev-acme-corp`,
  - `pr-1234-acme-corp`.
- The Palmyra backend:
  - treats this external tenant identifier as an **input key only**,
  - maps it into the internal `tenantId` and `slug` via the Admin Space
    tenants registry, and
  - enforces that each deployment only accepts tokens whose auth-tenant
    prefix matches its own environment key (preventing cross-environment
    leakage).

This two-level model (bucket per environment class; prefix per deployment
environment; consistent environment key in auth tenant IDs) preserves clear
separation between production, shared non-production, and ephemeral PR
environments while keeping configuration and operations manageable.

