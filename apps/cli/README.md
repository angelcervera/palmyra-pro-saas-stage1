# Palmyra Admin CLI

Command-line utilities for local/dev administration tasks (auth, bootstrap, tenants, users).

## Installation (local dev)

Build a binary first (recommended):

```bash
go build -o bin/platform-cli ./apps/cli
```

Then run commands via the binary:

```bash
bin/platform-cli --help
```

As an alternative, run directly from source:
```bash
go run ./apps/cli --help
```


## Commands

### bootstrap
Bootstrap platform resources. Creates the **admin** tenant space and an initial admin user (Phase 1).

```bash
bin/platform-cli bootstrap platform \
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
bin/platform-cli tenant create \
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
bin/platform-cli auth devtoken \
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
bin/platform-cli auth devtoken \
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
bin/platform-cli schema categories list \
  --database-url "postgres://palmyra:palmyra@localhost:5432/palmyra?sslmode=disable" \
  --env-key dev \
  --admin-tenant-slug admin \
  --include-deleted
```

Create or update (upsert) a category:
```bash
# create
bin/platform-cli schema categories upsert \
  --database-url "postgres://palmyra:palmyra@localhost:5432/palmyra?sslmode=disable" \
  --env-key dev \
  --admin-tenant-slug admin \
  --name "Payments" \
  --slug payments \
  --description "Schemas for payment flows"

# update (pass the category id)
bin/platform-cli schema categories upsert \
  --database-url "postgres://palmyra:palmyra@localhost:5432/palmyra?sslmode=disable" \
  --env-key dev \
  --admin-tenant-slug admin \
  --id 11111111-1111-1111-1111-111111111111 \
  --name "Payments & Billing"
```

Soft delete by id:
```bash
bin/platform-cli schema categories delete \
  --database-url "postgres://palmyra:palmyra@localhost:5432/palmyra?sslmode=disable" \
  --env-key dev \
  --admin-tenant-slug admin \
  --id 11111111-1111-1111-1111-111111111111
```
