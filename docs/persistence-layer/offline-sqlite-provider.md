# Offline SQLite Persistence Provider

This document describes the `offline-sqlite` implementation that now ships inside `@zengateglobal/persistence-sdk`.
It explains why we selected SQLite WASM + the SyncAccessHandle Pool VFS, how the provider is structured, and how
frontends can seed metadata or rotate the underlying database when a user logs in or out.

## 1. Technology choices

- **Engine:** [`@sqlite.org/sqlite-wasm`](https://www.npmjs.com/package/@sqlite.org/sqlite-wasm) gives us a fully
  supported WebAssembly build of SQLite plus a Promise-friendly Worker interface. We interact with it through the
  official `sqlite3Worker1Promiser` wrapper so queries always execute on a dedicated Worker thread.
- **Persistence layer:** Browsers expose several SQLite-friendly storage adapters. We intentionally target the
  [`opfs-sahpool` SyncAccessHandle VFS](https://sqlite.org/wasm/doc/trunk/persistence.md#vfs-opfs-sahpool) because it:
  - works in modern Chromium, Firefox, and **Safari 16.4+** without the COOP/COEP headers that the regular OPFS VFS
    requires;
  - uses pooled `FileSystemSyncAccessHandle` instances to keep I/O deterministic, which matches our “one DB per session”
    requirement;
  - does not rely on `SharedArrayBuffer`, avoiding Safari’s <17 storage bug that affects the default OPFS VFS.
- **Worker bootstrap:** A minimal module worker (`packages/persistence-sdk/src/providers/offline-sqlite/sqlite-worker.ts`)
  imports `sqlite3InitModule`, installs the SAH pool VFS, and then calls `sqlite3.initWorker1API()`. That keeps the
  execution model identical to the documentation’s [demo-123 walkthrough](https://sqlite.org/wasm/doc/trunk/demo-123.md)
  and the [Worker1 API contract](https://sqlite.org/wasm/doc/trunk/api-worker1.md).

## 2. Provider capabilities

- **PersistenceProvider contract:** The exported `OfflineSqliteProvider` implements all CRUD methods from
  `PersistenceProvider`, including pagination, batch writes, and soft deletes.
- **Schema metadata:** Metadata is stored locally in two tables (`schema_metadata`, `schema_versions`). The provider
  exposes `replaceMetadata(snapshot)` so callers can seed or refresh schemas after an online sync. Metadata lookups are
  cached in-memory per database.
- **Document storage:** Entity payloads live in the `entities` table as JSON strings. Writes are immutable—each write
  generates a new local `entityVersion` and timestamps the record. Soft deletes simply flip the `is_deleted` flag so the
  UI can continue to query historical data if needed.
- **Database rotation:** Because the offline UI only needs one database per session, the provider surfaces
  `setActiveDatabase(name)` and `close()` helpers. Logouts can call `close()` to release OPFS handles, and the next login
  can point the provider to a tenant-specific filename (for example, `/offline/user-123.db`). Internally the Worker is
  reused; we just close the current DB and open a new one via the same promiser.

## 3. Safari compatibility & fallbacks

- Safari <17 cannot safely use the regular OPFS VFS due to WebKit bug 255458. By using `opfs-sahpool` we stay within the
  set of browsers called out as compatible in the persistence guide and avoid the COOP/COEP headers that Safari-based
  PWAs typically cannot set.
- If an environment lacks Worker/OPFS support (for example, SSR), the provider throws early so applications can fall
  back to another storage option (e.g., the existing `offline` stub or an IndexedDB-backed adapter).

## 4. Usage summary

```ts
import { createOfflineSqliteProvider } from "@zengateglobal/persistence-sdk";

const provider = createOfflineSqliteProvider({
  databaseName: "/offline/default.db",
});

await provider.replaceMetadata(latestSnapshot); // seed schemas after an online sync
const items = await provider.queryEntities({ tableName: "entity_documents" });

await provider.setActiveDatabase(`/offline/${tenantId}.db`);
await provider.close(); // when logging out
```

Frontends should call `replaceMetadata` whenever they pull down new schemas from the online API and switch databases
when the active tenant changes. All CRUD operations then behave like the online provider but run entirely on the local
SQLite file.

## 5. Change journal (FIFO queue)

- Every mutation creates an immutable entity row and appends a corresponding entry in `entity_journal`.
- The journal acts as a FIFO queue: call `listJournalEntries()` to read all pending rows (ordered by `change_id`), push
  them to the server, then call `clearJournalEntries()` to wipe the table and reset the autoincrement sequence. After a
  successful sync the queue is empty and ready for the next batch.
- Because the queue is local and keyed by an autoincrement `change_id`, it is unaffected by client timezones or clock
  drift.

## 6. Integration test (optional proof)

To exercise the provider against a real sqlite-wasm runtime without spinning up a browser, we ship a Vitest proof that
swaps the worker-backed promiser with a Node-based sqlite-wasm promiser. The test seeds metadata, performs CRUD, closes
the provider, reopens a new provider instance, and confirms the data is still available within the same sqlite-wasm
process (satisfying the “one DB per session” requirement).

```bash
pnpm -C packages/persistence-sdk run test:offline-sqlite
```

The command runs a single Vitest file (`src/providers/offline-sqlite/offline-sqlite.integration.test.ts`) and is opt-in so day-to-day CI jobs
remain fast. Because this uses sqlite-wasm’s Node build, it validates the provider wiring while avoiding browser
automation dependencies.
