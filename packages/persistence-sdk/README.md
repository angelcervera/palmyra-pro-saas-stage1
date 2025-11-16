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
