-- Base schema initialization for Palmyra Pro persistence layer.
-- This file is executed automatically by the Postgres container on first startup.

-- Schema Categories capture the taxonomy for schemas.
CREATE TABLE IF NOT EXISTS schema_categories (
    category_id UUID PRIMARY KEY,
    parent_category_id UUID REFERENCES schema_categories(category_id),
    name TEXT NOT NULL,
    slug TEXT NOT NULL CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS schema_categories_name_idx
    ON schema_categories(name)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS schema_categories_slug_idx
    ON schema_categories(slug)
    WHERE deleted_at IS NULL;

-- Schema Repository stores every JSON schema definition and lifecycle flags.
CREATE TABLE IF NOT EXISTS schema_repository (
    schema_id UUID NOT NULL,
    schema_version TEXT NOT NULL CHECK (schema_version ~ '^[0-9]+\.[0-9]+\.[0-9]+$'),
    schema_definition JSONB NOT NULL,
    table_name TEXT NOT NULL CHECK (table_name ~ '^[a-z][a-z0-9_]*$'),
    slug TEXT NOT NULL CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
    category_id UUID NOT NULL REFERENCES schema_categories(category_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_soft_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (schema_id, schema_version)
);

CREATE UNIQUE INDEX IF NOT EXISTS schema_repository_active_schema_idx
    ON schema_repository(schema_id)
    WHERE is_active AND NOT is_soft_deleted;

CREATE UNIQUE INDEX IF NOT EXISTS schema_repository_table_name_idx
    ON schema_repository(table_name)
    WHERE NOT is_soft_deleted AND is_active;

CREATE UNIQUE INDEX IF NOT EXISTS schema_repository_slug_idx
    ON schema_repository(slug)
    WHERE NOT is_soft_deleted AND is_active;

CREATE INDEX IF NOT EXISTS schema_repository_category_idx
    ON schema_repository(category_id)
    WHERE NOT is_soft_deleted;

-- Users table for admin/approval workflows.
CREATE TABLE IF NOT EXISTS users (
    user_id UUID PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    full_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS users_created_at_idx ON users(created_at DESC);
