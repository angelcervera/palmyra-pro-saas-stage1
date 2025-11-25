# Palmyra Admin CLI

Command-line utilities for local/dev administration tasks (auth, bootstrap, tenants, users).

## Installation (local dev)

Build a binary first (recommended):

```bash
go build -o bin/cli-platform-admin ./apps/cli
```

Then run commands via the binary:

```bash
bin/cli-platform-admin --help
```

As an alternative, run directly from source:
```bash
go run ./apps/cli --help
```


## Commands

### bootstrap
Bootstrap platform resources. Creates the **admin** tenant space and an initial admin user (Phase 1).

```bash
bin/cli-platform-admin bootstrap platform \
  --database-url "postgres://palmyra:palmyra@localhost:5432/palmyra?sslmode=disable" \
  --env-key dev \
  --tenant-slug admin \
  --tenant-name "Admin" \
  --admin-email admin@example.com \
  --admin-full-name "Palmyra Admin"
```

Flags:
- `--database-url` (required): Postgres connection string.
- `--env-key`: environment key prefix (default `dev`).
- `--tenant-slug`: admin tenant slug (default `admin`).
- `--tenant-name`: display name for admin tenant (default `Admin`).
- `--admin-email` (required): initial admin user email.
- `--admin-full-name` (required): initial admin user full name.
- `--created-by`: UUID for audit (optional; defaults to random).

### tenant
Tenant utilities.

#### tenant create
Create and fully bootstrap a tenant space (role, schema, grants, base tables, tenant admin user).

```bash
bin/cli-platform-admin tenant create \
  --database-url "postgres://palmyra:palmyra@localhost:5432/palmyra?sslmode=disable" \
  --env-key dev \
  --tenant-slug acme \
  --tenant-name "Acme Corp" \
  --admin-email admin@acme.com \
  --admin-full-name "Acme Admin"
```

Flags:
- `--database-url` (required): Postgres connection string.
- `--env-key`: environment key prefix (default `dev`).
- `--tenant-slug` (required): tenant slug.
- `--tenant-name`: display name for tenant.
- `--admin-email` (required): tenant admin user email.
- `--admin-full-name` (required): tenant admin user full name.

### auth
Group for authentication helpers.

#### auth devtoken
Generate an unsigned Firebase-compatible JWT for local/CI use (works with `AUTH_PROVIDER=dev`).

Required flags:
- `--project-id` — Firebase project ID (aud/iss)
- `--tenant` — `firebase.tenant` claim
- `--user-id` — user_id/sub/uid claim
- `--email` — email claim

Optional flags:
- `--name` — display name
- `--email-verified` (default: true)
- `--admin` — set `isAdmin=true`
- `--palmyra-roles` — comma-separated roles (e.g. `admin,user_manager`)
- `--tenant-roles` — comma-separated tenant roles
- `--sign-in-provider` — firebase.sign_in_provider (default: `password`)
- `--expires-in` — token lifetime (e.g. `30m`, `2h`, default `1h`)
- `--audience` — override `aud` (defaults to project-id)
- `--issuer` — override `iss` (defaults to `https://securetoken.google.com/<project-id>`)

Examples:

```bash
# Admin token for dev tenant
bin/cli-platform-admin auth devtoken \
  --project-id local-palmyra \
  --tenant tenant-dev \
  --user-id admin-123 \
  --email admin@example.com \
  --name "Dev Admin" \
  --admin \
  --palmyra-roles admin \
  --tenant-roles admin \
  --expires-in 2h

# Non-admin user
bin/cli-platform-admin auth devtoken \
  --project-id local-palmyra \
  --tenant tenant-dev \
  --user-id user-001 \
  --email user@example.com \
  --name "Viewer User"
```

Output is written to stdout; pipe or copy into `Authorization: Bearer <token>` headers or `sessionStorage.setItem('jwt', token)` for the web app.

## Roadmap
- `auth create-user`
- `auth create-tenant`
- `bootstrap` helpers for first-run setup

### schema categories
Admin CRUD helpers for schema categories (backed by the shared persistence layer).

List categories (optionally include soft-deleted):
```bash
bin/cli-platform-admin schema categories list \
  --database-url "postgres://palmyra:palmyra@localhost:5432/palmyra?sslmode=disable" \
  --env-key dev \
  --admin-tenant-slug admin \
  --include-deleted
```

Create or update (upsert) a category:
```bash
# create
bin/cli-platform-admin schema categories upsert \
  --database-url "postgres://palmyra:palmyra@localhost:5432/palmyra?sslmode=disable" \
  --env-key dev \
  --admin-tenant-slug admin \
  --name "Payments" \
  --slug payments \
  --description "Schemas for payment flows"

# update (pass the category id)
bin/cli-platform-admin schema categories upsert \
  --database-url "postgres://palmyra:palmyra@localhost:5432/palmyra?sslmode=disable" \
  --env-key dev \
  --admin-tenant-slug admin \
  --id 11111111-1111-1111-1111-111111111111 \
  --name "Payments & Billing"
```

Soft delete by id:
```bash
bin/cli-platform-admin schema categories delete \
  --database-url "postgres://palmyra:palmyra@localhost:5432/palmyra?sslmode=disable" \
  --env-key dev \
  --admin-tenant-slug admin \
  --id 11111111-1111-1111-1111-111111111111
```

### schema definitions
Manage schema repository definitions/versions.

List all schemas (optionally include inactive):
```bash
bin/cli-platform-admin schema definitions list \
  --database-url "postgres://palmyra:palmyra@localhost:5432/palmyra?sslmode=disable" \
  --env-key dev \
  --admin-tenant-slug admin \
  --include-inactive
```

List versions for a specific schema (optionally include soft-deleted):
```bash
bin/cli-platform-admin schema definitions list \
  --database-url "postgres://palmyra:palmyra@localhost:5432/palmyra?sslmode=disable" \
  --env-key dev \
  --admin-tenant-slug admin \
  --schema-id 11111111-1111-1111-1111-111111111111 \
  --include-deleted
```

Create or update (upsert) a schema definition version:
```bash
bin/cli-platform-admin schema definitions upsert \
  --database-url "postgres://palmyra:palmyra@localhost:5432/palmyra?sslmode=disable" \
  --env-key dev \
  --admin-tenant-slug admin \
  --table-name cards_entities \
  --slug cards-schema \
  --category-id 22222222-2222-2222-2222-222222222222 \
  --definition-file ./schemas/cards.json
  # optionally provide --schema-id <uuid> and/or --schema-version 1.2.3
```

Soft delete a schema version:
```bash
bin/cli-platform-admin schema definitions delete \
  --database-url "postgres://palmyra:palmyra@localhost:5432/palmyra?sslmode=disable" \
  --env-key dev \
  --admin-tenant-slug admin \
  --schema-id 11111111-1111-1111-1111-111111111111 \
  --schema-version 1.0.0
```
