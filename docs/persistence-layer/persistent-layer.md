# Persistent Layer

All data access operations are routed through the **Persistent Layer**, which provides a unified and controlled
abstraction over data storage.
This layer implements a **document-oriented persistence model** while maintaining strict **schema governance** through
versioned JSON Schema definitions that describe the structure and validation rules for each entity.

Although the model follows a document-database paradigm, **PostgreSQL** is adopted as the underlying storage engine to
leverage transactional integrity, scalability, and SQL-based querying capabilities.

This architecture provides a **controlled evolution path** for data definitions and content, enabling **immutable
storage**, **auditable change tracking**, and **safe migration** across schema versions.
It balances the flexibility of a document store with the robustness and consistency guarantees of a relational database.

## Repository Location

Follow the monorepo convention that concentrates shared backend infrastructure under `platform/go`. The persistence
layer should therefore live in a dedicated package such as `platform/go/persistence`, making it available to every
domain service without duplicating storage concerns within `domains/<domain>/be`.

## Schema Repository

The **Schema Repository** acts as the central authority for defining and managing data structures.
It maintains a **complete version history** of every schema associated with the system's entities, supporting **schema
evolution** and **controlled backward compatibility**.

Each schema version follows **semantic versioning** (`major.minor.patch`) and is treated as **immutable**, so
modifications to existing schemas always result in the creation of a new version rather than altering prior definitions.
This ensures **auditability**, **traceability**, and a consistent **data lineage** across time.

The repository enables **data migration processes** between schema versions when evolution is required, ensuring
long-term compatibility of stored data and facilitating automated validation and transformation.

Each schema entry contains:

* `schema_id`: A unique schema identifier.
* `schema_version`: A semantic version number (`major.minor.patch`).
* `schema_definition`: A `JSONB` field containing the formal JSON Schema definition specifying structure and constraints.
* `created_at`: A creation timestamp.
* `updated_at`: A last-modified timestamp.
* `deleted_at`: A soft-delete timestamp for logical removal.
* `is_active`: A flag indicating whether it is the currently active version.

## Entity Tables

For every entity defined in the Schema Repository, the system automatically provisions a **corresponding entity table**
within PostgreSQL.
This table serves as the immutable ledger of all document instances associated with that entity type.

The Persistent Layer is responsible for mediating between **JSON-based document representations** and **relational
storage structures**, ensuring schema conformity, referential integrity, and version alignment.

Entity records are **immutable by design**. Updates do not overwrite existing data but instead create new document
versions, preserving historical state and enabling **temporal (time-travel) queries** and **audit tracking**.

Each entity table currently stores the columns below (see `EntityRepository.ensureEntityTable` for the authoritative DDL):

* `entity_id UUID`: Primary key (together with `entity_version`). During design we plan to allow client-provided IDs,
  but the implementation still stores UUIDs today, generating them automatically when the caller omits an ID. Once the
  new identifier format lands, this column will move to text with the regex documented above.
* `entity_version TEXT`: Semantic version string (`major.minor.patch`) representing each immutable revision.
* `schema_id UUID`: Foreign key to `schema_repository.schema_id`.
* `schema_version TEXT`: Foreign key to `schema_repository.schema_version`; paired with `schema_id` to pin the schema.
* `slug TEXT`: Search-friendly slug constrained to `^[a-z0-9]+(?:-[a-z0-9]+)*$`; unique among active records.
* `payload JSONB`: The validated document body.
* `created_at TIMESTAMPTZ`: Insert timestamp captured by Postgres.
* `is_active BOOLEAN`: Indicates the latest version for a given `entity_id` (enforced via partial unique index).
* `is_soft_deleted BOOLEAN`: Marks versions hidden from default queries; soft deletes toggle this flag and clear
  `is_active` for the entity.

There are no `updated_at`/`deleted_at` timestamps because entity versions are immutable and only track creation time.
