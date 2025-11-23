-- Base schema initialization for every tenant space, including the one used as administratio space.

-- Users table for admin/approval workflows.
CREATE TABLE IF NOT EXISTS users (
    user_id UUID PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    full_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS users_created_at_idx ON users(created_at DESC);
