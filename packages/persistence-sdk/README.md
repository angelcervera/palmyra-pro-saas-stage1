# @zengateglobal/persistence-sdk (WIP)

Tenant-facing persistence client SDK that wraps Palmyra Pro’s document-oriented persistence layer.  
This package will eventually export the shared `PersistenceClient` interface plus concrete adapters
for online (API-backed) and offline (local-first + user-triggered sync) usage.

## Rich payload support

- Callers can pass their own domain models (including `Date` instances or other plain objects) to `saveEntity`/
  `batchWrites`.
- The SDK takes care of normalizing those values into JSON-safe wire payloads (dates → ISO strings, nested
  objects/arrays → JSON) before
  sending requests to the backend or persisting offline.
- Metadata fields such as `createdAt`, `updatedAt`, and `deletedAt` are surfaced as real `Date` objects on every
  entity/metadata record; adapters
  convert the backend ISO strings for you.
- Responses come back as whatever type parameter the caller requested; helpers in `src/shared` provide the JSON ↔
  app-model conversion.

## Status

- [x] Core interface + shared types defined under `src/core`.
- [x] Package wiring + build config (`tsc`) ready.
- [ ] Online adapter wired to real `@zengateglobal/api-sdk` calls (currently placeholder responses).
- [ ] Offline adapter storage engine + manual sync pipeline.
- [ ] Unit tests (Vitest) for critical flows.

## Sync contract (WIP)

`PersistenceClient.sync(request)` is stubbed for now; use the request shape below when wiring providers:
- `sourceProviderId`: provider that holds the journal to push (e.g., offline Dexie).
- `targetProviderId`: provider that will receive outgoing changes and is used to pull fresh data.

Planned sync flow (implementation to come):
1) If the source provider has journal entries, push them to the target via `batchWrites`. If this fails, stop and surface the error.
2) On success, clear the source provider journal.
3) Pull schemas from the target via `getMetadata` and apply them to the source via `setMetadata`.
4) Clear all entity tables in the source provider (WIP in provider contract).
5) For each schema/table from the target, `queryEntities` on the target and `batchWrites` the results into the source.

Note: This MVP flow is intentionally simple and downloads all data, so it’s not performance-optimal. It exists to ship the first iteration quickly; later versions will optimize the sync strategy.

## Folder layout

```
packages/persistence-sdk/
├── README.md
└── src/
    ├── core/                # Interface contracts, domain types, factories
    ├── adapters/
    │   ├── online/          # Implementation that proxies @zengateglobal/api-sdk
    │   └── offline/         # Implementation that stores locally + manual sync
    └── shared/              # Cross-cutting utilities (error mapping, JWT helpers, etc.)
```

Additional build config files (tsconfig, package.json, etc.) are already in place so this package
can be built and consumed locally via `pnpm run build -C packages/persistence-sdk`.

## Providers.

- [X] In-memory offline provider
- [X] Dexie offline provider
- [X] api-sdk online provider
- [ ] Pure indexedDB offline provider
- [ ] CouchDB offline provider
- [ ] SQLite offline provider
- [ ] PgLite offline provider
