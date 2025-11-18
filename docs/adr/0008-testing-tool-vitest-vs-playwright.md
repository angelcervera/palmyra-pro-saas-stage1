---
adr: 0008
id: adr-0008-vitest-vs-playwright-offline-sqlite
title: Prefer Vitest (Node sqlite-wasm) Over Playwright For Offline SQLite Proof Tests
status: Accepted
date: 2025-11-18
deciders: [core, platform]
tags: [testing, persistence, tooling]
relatedDocs:
  - docs/persistence-layer/offline-sqlite-provider.md
  - packages/persistence-sdk/src/providers/offline-sqlite/offline-sqlite.integration.test.ts
---

## Context

We needed an integration test proving the new `offline-sqlite` persistence provider works end-to-end against sqlite-wasm. Two candidate approaches were evaluated:

1. **Playwright + Vitest browser mode**: launch headless Chromium, run the provider in a PWA-like environment, and interact with OPFS via Web Workers. This mirrors the production browser stack.
2. **Pure Vitest (Node)**: run sqlite-wasm directly inside Node, patching the worker promiser with a Node-specific adapter that loads the WASM bundle from disk.

The Playwright path required downloading large browser binaries, adding COOP/COEP headers, and still failed because the bundled Chromium headless shell lacks the OPFS + `SharedArrayBuffer` combination that sqlite-wasm’s worker needs (tests consistently timed out before `sqlite3Worker1Promiser` resolved). That setup also introduces CI friction (browser caching, GPU/Atomics constraints) for a proof test that only needs to verify persistence logic.

## Decision

Adopt **Vitest running entirely in Node** for the offline SQLite proof. We dynamically import sqlite-wasm’s `sqlite3.mjs`, patch `fetch` to serve the local `sqlite3.wasm`, and provide a custom `promiserFactory` to the provider so its normal codepath is exercised without Web Workers. The opt-in script `pnpm -C packages/persistence-sdk run test:offline-sqlite` now executes quickly and deterministically, with no external browser dependencies.

## Consequences

- ✅ Fast, reproducible test (~0.6s) that validates schema seeding, CRUD, and provider reopen flows without downloading browsers or setting COOP/COEP headers.
- ✅ Works in any CI agent that can run Node 24; no headless browser sandboxing issues.
- ⚠️ The test does **not** exercise real OPFS/Worker semantics; it proves persistence logic but not browser storage APIs. Browser-level coverage still relies on future end-to-end tests in the PWA shell.
- 🚧 Documentation now directs engineers to run the Node-based proof when they need confidence in sqlite-wasm wiring.

## Alternatives Considered

### Playwright + Vitest browser mode (Rejected)
- Required 300+ MB browser downloads and additional CI setup.
- Headless shell lacked the OPFS features sqlite-wasm expects; tests hung before the worker became ready despite COOP/COEP headers.
- Debugging was difficult (limited console visibility, screenshots not helpful for initialization failures).

### Skip an integration test entirely (Rejected)
- Would leave the new provider untested, relying only on unit tests or future app code to catch regressions.
- The proof test was a stakeholder requirement before wiring the provider into real offline flows.

## Implementation Notes

- `OfflineSqliteProvider` now accepts an optional `promiserFactory`, enabling tests (or future environments) to bypass Web Workers while reusing the same provider logic.
- `tests/node-sqlite-promiser.ts` patches `fetch` to serve `sqlite3.wasm` with `application/wasm` from disk, then reuses sqlite-wasm’s OO1 API to back the promiser.
- `pnpm run test:offline-sqlite` runs only this proof and is opt-in to keep regular CI fast.
- If we later need real-browser coverage, add a separate Playwright suite behind another opt-in script without replacing this fast proof.
