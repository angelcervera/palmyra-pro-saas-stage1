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

- **Bootstrap:** Mount the `database/` directory into `/docker-entrypoint-initdb.d` (see `docker-compose.yml`). The `000_init_schema_and_seeds.sh` script automatically applies every file in `schema/` and, when `PALMYRA_DB_SEED_MODE=dev`, the contents of `seeds/dev/`.
- **Migrations:** Add forward-only SQL under `database/migrations` with a consistent prefix (e.g., `20251117T120000_add_status_to_users.sql`) when evolving the schema. The future migration runner should consume this folder.
- **Seeding:** Place deterministic seed scripts in `database/seeds/dev` or `database/seeds/staging`. These scripts are opt-in and should not be mounted automatically in production-like environments.

Keep the contracts in `/contracts` as the source of truth for APIs, and mirror structural changes here, regenerating code as needed per `docs/persistent-layer.md`.
