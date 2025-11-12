---
adr: 0004
id: adr-0004-chi-oapi
title: Contract-first Backend with Chi + oapi-codegen
status: Accepted
date: 2025-11-02
deciders: [core, backend]
tags: [backend, go, openapi, chi]
relatedDocs:
  - docs/api-server.md
  - docs/api.md
---

## Context
We are contract-first. We need a Go HTTP stack with strong OpenAPI integration and generated server interfaces that our domain code can implement cleanly.

## Decision
Use Chi for HTTP routing and `oapi-codegen` to generate models and Chi server interfaces under `/generated/go/<domain>`. Domain handlers implement these interfaces; generated code remains read-only.

## Consequences
- Clear separation between generated contracts and handwritten domain code.
- Consistent mapping to ProblemDetails and pagination schemas from `contracts/common`.
- Requires regeneration on contract changes; CI must enforce no manual edits in `/generated`.

## Alternatives Considered
- net/http + manual handlers: more boilerplate, easier to drift from contracts.
- Other generators/frameworks: acceptable, but `oapi-codegen` + Chi best fits our stack and conventions.

## Links
- docs/api-server.md

