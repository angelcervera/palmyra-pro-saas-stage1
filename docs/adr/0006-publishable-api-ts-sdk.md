---
adr: 0006
id: adr-0006-api-ts-sdk
title: Publish a Reusable API TypeScript SDK Package
status: Accepted
date: 2025-11-03
deciders: [core, frontend]
tags: [frontend, sdk, openapi, packages, npm]
relatedDocs:
  - docs/web-app.md
  - docs/api.md
  - docs/agent-index.yaml
---

## Context
Multiple apps (Admin, PWA, potential external apps) must consume a consistent, versioned, type‑safe client for our contract‑first API. We will generate per‑domain clients directly inside the SDK package under `packages/api-sdk/src/generated/<domain>`, avoiding duplication and manual sync steps.

## Decision
Create a publishable, scoped SDK package `@tcglanddev/api-sdk` under `/packages/api-sdk` that composes the generated clients and exports a stable surface (types and services). Apps in the monorepo, and external apps, will depend on this package rather than importing from `generated/ts/*`.

Key properties:
- Source of truth: OpenAPI contracts in `/contracts/*` → codegen into `packages/api-sdk/src/generated/<domain>` (read‑only within SDK)
- Packaging: ESM output with TypeScript types; uses Fetch client; ProblemDetails modeled per contract
- Scope: `@tcglanddev` (see `.npmrc`); published to the private/company registry
- Versioning: Semantic Versioning; breaking changes only with a major bump
- Consumption: internal via `workspace:*`, external via registry

## Consequences
- Apps import from a single package; no direct dependency on generated internals or codegen layout
- Clear release flow and versioning gates for API changes
- Slight indirection: SDK must be rebuilt/published when contracts change (expected)

## Implementation Notes
- Directory: `/packages/api-sdk`
- Build: compiles SDK sources (including `src/generated`) and wraps with `/platform/ts/http` for auth and error handling when appropriate
- Publish: `pnpm -F @tcglanddev/api-sdk publish` (registry scope configured in `.npmrc`)

## Alternatives Considered
- Generate to a separate top-level `generated/ts/*`: adds copy/sync steps and increases drift risk
- Per‑domain SDK packages: more granular but increases package sprawl and release coordination
