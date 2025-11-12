---
adr: 0001
id: adr-0001-react19
title: Adopt React 19 for Frontend Apps
status: Accepted
date: 2025-11-02
deciders: [core, frontend]
tags: [frontend, react, node24]
relatedDocs:
  - docs/web-app.md
  - docs/project-requirements-document.md
---

## Context
We require multiple SPAs: an Admin app and a PWA app, running on Node.js 24 with Vite and TypeScript. We need modern React features, wide ecosystem support, and no SSR requirement.

## Decision
Adopt React 19 for all frontend applications (Admin and PWA). Do not use Next.js. Use Vite for build/dev.

## Consequences
- Unlocks Suspense and new React 19 ergonomics; simpler data fetching integration.
- Ecosystem alignment with current libraries (React Router, TanStack Query, RHF, shadcn).
- No SSR/SSG out of the box (acceptable per scope); hydration strictly client‑side.

## Alternatives Considered
- React 18: stable, but misses 19 improvements we plan to leverage.
- Next.js 14/15: unnecessary complexity (routing/SSR) for our SPA needs.

## Links
- docs/web-app.md
