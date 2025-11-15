# @zengateglobal/persistence-sdk (WIP)

Tenant-facing persistence client SDK that wraps Palmyra Pro’s document-oriented persistence layer.  
This package will eventually export the shared `PersistenceClient` interface plus concrete adapters
for online (API-backed) and offline (local-first + user-triggered sync) usage.

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
