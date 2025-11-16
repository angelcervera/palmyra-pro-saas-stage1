---
id: adr-index
version: 1.0.0
lastUpdated: 2025-11-16
scope: decisions
appliesTo:
  - apps/*
  - domains/*
  - platform/*
relatedDocs:
  - docs/web-app.md
  - docs/api-server.md
  - docs/project-requirements-document.md
---

# Architecture Decision Records (ADR) — Index

This index lists accepted high‑level decisions with stable IDs. Each ADR includes context, decision, consequences, and alternatives.

- ADR-0001 — Adopt React 19 for frontend apps
  - File: `docs/adr/0001-frontend-framework-react19.md`
  - Status: Accepted — 2025-11-02

- ADR-0002 — Use shadcn/ui + Tailwind (Radix) for UI primitives
  - File: `docs/adr/0002-ui-library-shadcn-tailwind.md`
  - Status: Accepted — 2025-11-02

- ADR-0003 — Use VitePWA (Workbox) for PWA offline, caching, and updates
  - File: `docs/adr/0003-pwa-vitepwa-workbox.md`
  - Status: Accepted — 2025-11-02

- ADR-0004 — Contract-first backend with Chi + oapi-codegen
  - File: `docs/adr/0004-backend-chi-oapi-codegen.md`
  - Status: Accepted — 2025-11-02

- ADR-0005 — Consider Astro as frontend shell
  - File: `docs/adr/0005-frontend-shell-astro.md`
  - Status: Rejected (Admin/PWA) — 2025-11-02

- ADR-0006 — Publish a reusable API TS SDK package
  - File: `docs/adr/0006-publishable-api-ts-sdk.md`
  - Status: Accepted — 2025-11-03

- ADR-0007 — Biome formatting plus minimal ESLint for React hooks
  - File: `docs/adr/0007-frontend-tooling-biome-eslint-hooks.md`
  - Status: Accepted — 2025-11-16
