---
adr: 0005
id: adr-0005-astro-shell
title: Consider Astro as Frontend Shell
status: Rejected (Admin/PWA)
date: 2025-11-02
deciders: [core, frontend]
tags: [frontend, architecture, spa, pwa, astro]
relatedDocs:
  - docs/web-app.md
  - docs/adr/index.md
---

## Context
We evaluated Astro as a potential shell for the frontend. Astro excels for content-heavy, SEO-first sites and hybrid islands, but our primary apps are an Admin SPA and a PWA with offline-first behavior and role-gated CRUD flows. These prioritize client routing, background sync, and tight SPA ergonomics over SSR/SSG.

## Decision
Do not use Astro as the application shell for Admin or PWA. Retain React 19 + Vite SPA shells (`apps/web-admin`, `apps/web-pwa`). Astro remains a candidate for a separate marketing/docs site if needed.

## Consequences
- Simpler routing and service worker lifecycle for offline-first PWA.
- Fewer moving parts (no Astro layer) and consistent SPA DevX across apps.
- If we later need SEO/SSG for public pages, we can introduce a dedicated Astro (or similar) marketing app without impacting Admin/PWA.

## Alternatives Considered
- Use Astro for all web apps: adds complexity to client routing and SW flows; limited upside without SEO requirements.
- Keep pure SPA shells (chosen): best fit for Admin and PWA requirements.

## Revisit Criteria
- A public marketing/docs site with strong SEO needs.
- A future requirement for SSR/edge rendering in a public-facing app (not Admin/PWA).

