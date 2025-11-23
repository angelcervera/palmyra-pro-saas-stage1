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
