-- Tenant registry (immutable, versioned). Relies on search_path being set to the admin schema.

CREATE TABLE IF NOT EXISTS tenants (
    tenant_id UUID NOT NULL,
    tenant_version TEXT NOT NULL CHECK (tenant_version ~ '^[0-9]+\.[0-9]+\.[0-9]+$'),
    slug TEXT NOT NULL CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
    display_name TEXT NULL,
    status TEXT NOT NULL,
    schema_name TEXT NOT NULL CHECK (schema_name ~ '^[a-z][a-z0-9_]*$'),
    base_prefix TEXT NOT NULL,
    short_tenant_id TEXT NOT NULL CHECK (short_tenant_id ~ '^[0-9a-fA-F]{8}$'),
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    is_soft_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL,
    db_ready BOOLEAN NOT NULL DEFAULT FALSE,
    auth_ready BOOLEAN NOT NULL DEFAULT FALSE,
    last_provisioned_at TIMESTAMPTZ NULL,
    last_error TEXT NULL,
    PRIMARY KEY (tenant_id, tenant_version)
);

-- Only one active version per tenant.
CREATE UNIQUE INDEX IF NOT EXISTS tenants_active_one_per_id
    ON tenants (tenant_id) WHERE is_active = TRUE;

-- Prevent duplicate slugs among non-deleted tenants (across versions).
CREATE UNIQUE INDEX IF NOT EXISTS tenants_slug_unique_active
    ON tenants (slug) WHERE is_soft_deleted = FALSE;

CREATE INDEX IF NOT EXISTS tenants_slug_idx ON tenants (slug);
CREATE INDEX IF NOT EXISTS tenants_created_at_idx ON tenants (created_at DESC);

