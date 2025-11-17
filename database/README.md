# Database Assets

This directory centralizes every SQL artifact that backs the Palmyra persistence layer so the schema can evolve outside of embedded Go strings.

```
database/
  schema/        # Canonical DDL snapshots executed when provisioning a fresh environment
  migrations/    # Incremental change sets (apply in timestamp order across envs)
  seeds/
    dev/         # Local-only fixtures that create sample data for developers
    staging/     # Sanitized records for shared staging environments
```

## Usage

- **Bootstrap:** Mount `database/schema` into `/docker-entrypoint-initdb.d` (see `docker-compose.yml`) or run the files manually against an empty database to create baseline tables/indexes.
- **Migrations:** Add forward-only SQL under `database/migrations` with a consistent prefix (e.g., `20251117T120000_add_status_to_users.sql`) when evolving the schema. The future migration runner should consume this folder.
- **Seeding:** Place deterministic seed scripts in `database/seeds/dev` or `database/seeds/staging`. These scripts are opt-in and should not be mounted automatically in production-like environments.

Keep the contracts in `/contracts` as the source of truth for APIs, and mirror structural changes here, regenerating code as needed per `docs/persistent-layer.md`.
