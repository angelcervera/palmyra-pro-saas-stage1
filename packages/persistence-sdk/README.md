# @zengateglobal/persistence-sdk (WIP)

Tenant-facing persistence client SDK that wraps Palmyra Pro’s document-oriented persistence layer.  
This package will eventually export the shared `PersistenceClient` interface plus concrete adapters
for online (API-backed) and offline (local-first + user-triggered sync) usage.

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

Additional build config files (tsconfig, package.json, etc.) will be added once implementation begins.
