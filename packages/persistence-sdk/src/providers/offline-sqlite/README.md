# Offline SQLite Provider (current state)

This document snapshots the capabilities and gaps of the `offline-sqlite` persistence provider.

## What works
- Local storage via `sqlite-wasm` in a worker (default DB `file:/palmyra-offline.db?vfs=opfs-sahpool`; optional `kvvfs`).
- Metadata bootstrap and caching: `schema_metadata` + `schema_versions` tables; `replaceMetadata(initialSnapshot)` seeds tables and pre-creates entity tables.
- Entity storage model per table: immutable versions (`entity_id`, `entity_version`, `schema_version`, `payload` JSON string, `ts`, `is_deleted`, `is_active`) with unique index enforcing one active row per entity.
- CRUD primitives: `getEntity`, paginated `queryEntities` (active + non-deleted, newest first), `saveEntity` (always new version; auto `entityVersion`; updates mark old version inactive), `deleteEntity` (soft-delete current active row).
- Batch writes: transactional save/delete loop with `BatchWriteError` carrying table/entity context.
- Change journal: `entity_journal` captures create/update/delete entries; helpers to list and clear it.
- Database lifecycle: lazy open, `setActiveDatabase` to swap files, `close` to release handle; injectable worker/promiser for SSR/Node tests.

## Limitations
- No sync orchestration between offline and online providers; `PersistenceClient.sync` is unimplemented and there is no merge/conflict policy. Sync implementation is for target providers only. At the moment, offline clients are expected to be only source provider.

## Options overview
- `databaseName`, `vfs`, `workerFactory`, `promiserFactory`, `initialMetadata`, `logger` (debug/error hooks); `enableJournal` is defined but unused.

## TODO
- [ ] Enforce schema validation using stored definitions. At the moment, no JSON Schema validation of payloads despite stored schema definitions; metadata only supplies `activeVersion`.
- [ ] Implement richer queries (filters/sorts) and honor soft-delete defaults in `getEntity` when desired.. At the moment Filtering/sorting beyond recency pagination is absent; `getEntity` returns soft-deleted rows (caller must inspect `isDeleted`).
- [ ] Error handling is coarse; provider wraps errors but does not classify retryable vs fatal cases.
- [ ] Expand tests: unit coverage for edge cases (batch failures, soft-delete read, metadata reload) plus browser-targeted runs. At the moment, only one integration test (`offline-sqlite.integration.test.ts`) covers basic persist/reopen; broader unit coverage is missing.
