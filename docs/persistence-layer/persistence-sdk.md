# `@zengateglobal/persistence-sdk` — Tenant Persistence Client SDK

The `@zengateglobal/persistence-sdk` is the tenant-facing TypeScript client that turns Palmyra Pro's document‑oriented persistence layer into a simple, high‑level storage API for frontend apps.
Instead of dealing with raw HTTP calls, OpenAPI operations, low‑level schema plumbing, managing local storages or sync data, tenant UIs work with a small set of opinionated primitives that feel like in‑memory collections: read, write, and query immutable entity documents while the SDK quietly handles schema versions, validation boundaries, multi‑tenant isolation, offline storage ad sync.
Internally, it sits on top of the same backend persistence layer used by, for example, the admin tooling and REST API, but exposes a deliberately narrower, UX‑driven surface tailored to tenant scenarios (browsing, enriching, and consuming data) rather than low‑level persistence features.

## 1. Goals & Non‑Goals

> TBC: List the primary goals this SDK must achieve (e.g., safe document storage, schema-aware access, ergonomic tenant-facing API) and explicit non-goals (e.g., not an HTTP client, not an admin tooling SDK).

## 2. Architectural Overview

> TBC: Describe how the SDK sits between tenant UIs and the persistence layer, key design principles, and how it collaborates (or not) with `@zengateglobal/api-sdk` and other platform modules.

## 3. Core Responsibilities

> TBC: Enumerate the concrete responsibilities of the SDK (e.g., read/write entities, handle schema versions, enforce invariants, map persistence concepts into tenant-friendly abstractions) and which concerns stay in the backend.

## 4. Domain & Data Model

> TBC: Explain the high-level data model exposed to tenants: entities, documents, schema references, versioning concepts, and how these map (or intentionally do not map) to the underlying persistent layer tables.

## 5. Public API Surface

> TBC: Outline the main modules, entrypoints, and TypeScript types (e.g., clients, repositories, hooks, helpers). Include naming conventions, method patterns (get/save/list), and any generic abstractions that callers will rely on.

## 6. Configuration & Environment

> TBC: Describe required configuration (e.g., base URLs or adapters, auth tokens, environment flags), how configuration is passed (constructor options, providers), and how it aligns with Twelve‑Factor practices.

## 7. Authentication, Authorization & Multi‑Tenancy

> TBC: Clarify how auth flows into the SDK (JWT, Firebase, or session tokens), how tenant isolation is enforced at the SDK boundary, and which auth/tenant checks are delegated to the backend.

## 8. Error Handling & ProblemDetails Mapping

> TBC: Define the error model exposed to consumers, how backend ProblemDetails (RFC 7807) are represented, and patterns for distinguishing validation errors, auth errors, and unexpected failures.

## 9. Usage Patterns & Examples

> TBC: Provide canonical examples for common scenarios (reading entities, writing new versions, querying by filters, paginated lists) in both plain TypeScript and React usage (if applicable).

## 10. Performance, Caching & Limits

> TBC: Document any caching strategy, batching behavior, pagination defaults, and known limits (payload size, rate considerations) plus recommended usage patterns to avoid performance pitfalls.

## 11. Testing, Mocking & Tooling

> TBC: Describe how to test code that depends on this SDK (fixtures, mocks, MSW integration), and any utilities provided for end-to-end tests or storybook-style environments.

## 12. Versioning, Compatibility & Migration

> TBC: Explain how the SDK is versioned (SemVer), how breaking changes are handled, and how consumers should migrate between major versions or persistence-layer schema evolutions.

## 13. Open Questions & Future Work

> TBC: Track unresolved design decisions, potential extensions (offline support, advanced querying, richer type inference), and TODOs for future iterations of the SDK.
