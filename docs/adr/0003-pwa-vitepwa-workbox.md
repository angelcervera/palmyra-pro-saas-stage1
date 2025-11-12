---
adr: 0003
id: adr-0003-vitepwa-workbox
title: Use VitePWA (Workbox) for PWA Offline, Caching, and Updates
status: Accepted
date: 2025-11-02
deciders: [core, frontend]
tags: [pwa, workbox, offline, service-worker]
relatedDocs:
  - docs/web-app.md
---

## Context
We will ship a PWA app with offline‑first capabilities, background sync, and update flow. We need straightforward integration with Vite and modern SW tooling.

## Decision
Use `vite-plugin-pwa` (Workbox under the hood) with `registerType: 'autoUpdate'`, runtime caching for GET `/api/v1/*`, offline fallback, and Background Sync for queued mutations.

## Consequences
- Installable app with offline cache of shell/assets; runtime caching of data.
- Simple update UX (show banner when a new SW is waiting; skipWaiting/controllerchange).
- Requires careful token handling for SW tasks; never store refresh tokens in SW.

## Alternatives Considered
- Custom SW: more control, higher maintenance.
- Workbox CLI without Vite plugin: workable but less integrated DX.

## Links
- docs/web-app.md

