---
adr: 0007
id: adr-0007-biome-plus-eslint-hooks
title: Use Biome for Formatting/Linting and Keep ESLint for React Hooks Rules
status: Accepted
date: 2025-11-16
deciders: [core, frontend]
tags: [frontend, tooling, linting, react]
relatedDocs:
  - docs/web-app.md
  - docs/project-requirements-document.md
  - AGENTS.md
---

## Context
Frontend repositories currently rely on both Prettier and ESLint (with several plugins) to enforce formatting and lint rules. This combination causes slow feedback on large React/TypeScript projects (Admin app, SDKs) and requires ongoing migration work to the flat ESLint config. Biome provides a single, opinionated tool that covers formatting, linting, import sorting, and JSON/CSS checks with excellent performance. However, Biome does not yet implement the two React Hooks invariants that have guarded our React 19 code: `react-hooks/rules-of-hooks` and `react-hooks/exhaustive-deps`. Dropping those checks would introduce regressions that are hard to detect with tests alone.

## Decision
Adopt **Biome** as the default formatter/linter/checker for all TypeScript and React code. Retain **ESLint** only to run the `eslint-plugin-react-hooks` rules so React Hooks usage stays safe. ESLint executes with a minimal config (just the plugin, no stylistic or TypeScript rules) and only targets `.tsx` files.

## Consequences
- Faster local and CI feedback thanks to Biome's Rust engine and single-tool workflow.
- Consistent formatting/lint behavior across `apps/web-admin`, `packages/api-sdk`, `packages/persistence-sdk`, and future frontend packages.
- Editors can rely on Biome for auto-formatting and safe fixes, reducing Prettier/ESLint conflicts.
- A lightweight ESLint pass remains in place for Hooks correctness, so engineers still run two commands (Biome + ESLint) but with minimal overhead.
- Once Biome ships React Hooks analysis, removing ESLint will require only deleting the slim config.

## Alternatives Considered
### Keep only ESLint + Prettier
- High configuration overhead, plugin drift, and slower execution; does not simplify the toolchain.

### Switch entirely to Biome right now
- Fastest workflow but loses Hooks rules, allowing subtle bugs (incorrect dependency arrays, conditional hooks) to reach production.

### Wait for Biome Hooks support before adopting it
- Delays DX wins and keeps the current, slower setup for an unknown timeframe.

## Implementation Notes
- Keep `@biomejs/biome` pinned at the monorepo root with a shared config consumed by frontend packages.
- Ensure each frontend workspace (`apps/web-admin`, `packages/api-sdk`, `packages/persistence-sdk`, future FE packages) exposes `pnpm run format|lint|check` scripts that call Biome.
- Create a minimal `eslint.config.js` in `apps/web-admin` (and any other React package) that enables only `react-hooks/rules-of-hooks` (error) and `react-hooks/exhaustive-deps` (warn) for `.tsx` sources.
- Update CI and AGENT guidelines: run Biome (format → lint → check) plus ESLint hooks when touching React code.
- Revisit this ADR once Biome natively supports Hooks rules.
