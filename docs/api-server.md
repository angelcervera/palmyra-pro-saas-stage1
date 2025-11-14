---
id: api-server
version: 1.1.0
lastUpdated: 2025-11-02
appliesTo:
  - apps/api
  - domains/*/be
  - generated/go/*
  - platform/go/*
relatedDocs:
  - docs/api.md
  - docs/persistent-layer.md
  - docs/project-requirements-document.md
  - docs/adr/index.md
commandsUsed:
  - go generate ./tools/codegen/openapi/go
  - go test ./...
  - go fmt ./...
  - go build ./apps/api
---

# Backend Common Guideline

## 0) Golden Rules (Normative)

- BE-CON-001 (MUST) Contract-first: Start from `/contracts/*.yaml` (per domain) and `/contracts/common/*` (shared). Never invent fields or endpoints not defined there.
- BE-GEN-002 (MUST) Generated code is read-only: Everything under `/generated/` is a build artifact (Go server stubs/models via `oapi-codegen`). Never edit by hand; import from domain packages.
- BE-HTTP-003 (MUST) Two-response policy: Each path returns a single success code (200/201/204) and default `application/problem+json` (RFC7807) using shared `ProblemDetails`.
- BE-AUTH-004 (MUST) JWT everywhere (except `POST /auth/signup`, `POST /auth/login`). Enforce role-based authorization with middleware (`admin`, `user_manager`).
- BE-JSON-005 (MUST) JSON uses camelCase, ISO-8601 timestamps, and UUID ids; align with shared primitives.
- BE-ROUTE-006 (SHOULD) Serve under `/api/v1/...`; use plural, kebab-case paths; limit nested routes to strong relationships.

---

## 1) Monorepo Layout (Do Not Deviate)

```
/contracts/                # OpenAPI per domain + shared
/contracts/common/         # ProblemDetails, Pagination, security, primitives, iam, etc.
/domains/<domain>/be       # Handwritten Go code: services, handlers, repos, tests
/apps/api                  # API entrypoint, global middleware, wiring
/generated/go/<domain>     # oapi-codegen outputs for Chi stubs & models
/platform/go               # shared libs: config, logging, middleware, auth, http, errors
```

* Import models & server interfaces from `/generated/go/<domain>`.
* Place cross-cutting code only in `/platform/go`.

---

## 2) Project Baseline (scaffold once, reuse)

### 2.1 Dependencies

* `github.com/go-chi/chi/v5`
* `github.com/go-chi/chi/v5/middleware`
* `github.com/oapi-codegen/oapi-codegen/v2` (chi server gen)
* `github.com/google/uuid`
* `go.uber.org/zap` (logging)
* `github.com/caarlos0/env/v11` (config) or stdlib `flag` if simpler
* `github.com/stretchr/testify` (tests)

### 2.2 `/apps/api` main.go pattern

* Build a `chi.Router` with global middleware (request ID, real IP, recoverer, timeout).
* Attach **CORS**, **gzip**, **request logging**, **JWT auth**.
* Mount domain routers by calling domain `RegisterRoutes(router, deps)` functions.

**Example (abbrev.)**

TBC

---

### 2.3 Backend Build Gate (must run after each BE change)

After any change under `apps/api`, `domains/*/be`, or shared backend libraries in `platform/go/*`, run both commands locally before proceeding:

```bash
go fmt ./...
go build ./apps/api
```

Fix formatting and compilation errors immediately. Treat this gate like the frontend build gate in `docs/web-app.md`.

---

## 3) OpenAPI → Codegen Rules

* The AI agent must **regenerate** Go stubs whenever a `/contracts/*.yaml` changes via `oapi-codegen` generation file `generate`.
* Use `oapi-codegen` with Chi server interfaces:
    * models: `/generated/go/<domain>/models.gen.go`
    * server: `/generated/go/<domain>/server.gen.go` (interfaces to implement)
* Domain code implements generated interfaces (handlers) and converts to/from domain services.
* Always reference shared components via `$ref: "../common/<file>.yaml"` (Pagination, ProblemDetails, security, primitives, iam).

**Minimal config (example):**

```yaml
# oapi-codegen.yaml
package: <domain>
generate:
  chi-server: true
  models: true
output: ./generated/go/<domain>/server.gen.go
```

---

## 4) HTTP Conventions (must follow)

* Paths: `/api/v1/...`; plural nouns; kebab-case; nested only when relationship is strong. Use `:action` suffix sparingly (approve/reject).
* Methods: GET/POST/PUT/PATCH/DELETE semantics; set `Location` on 201.
* Filtering/sorting/pagination via query params (`page`, `pageSize`, `sort`) and return standardized pagination envelope. Default `pageSize=20`, max `100`.
* JSON: camelCase; ISO-8601 timestamps; UUID `id`; common fields `createdAt`, `updatedAt` (and `deletedAt` if soft delete).
* Headers: expose `X-RateLimit-*` when we implement rate limiting.

---

## 5) AuthN & AuthZ (mandatory)

* **Authentication**: JWT bearer in `Authorization: Bearer <token>` for all routes except `POST /auth/signup` and `POST /auth/login`. Token verification middleware populates `ctx` with `subject (userId)` and `roles`.
* **Authorization**: role checks:

    * `admin`: full access to user admin actions (approve/reject/enable/disable).
    * `user_manager`: can manage users per business rules.
* Implementation: `platform/go/auth` provides:

    * `JWT(verifyFn, extractFn)` middleware → shared entry point; pass either `FirebaseTokenVerifier` or `UnsignedTokenVerifier` as the verification function.
    * `RequireRoles(roles ...Role)` middleware → 403 with ProblemDetails when missing.
* **Error model**: on 401/403, return RFC7807 `ProblemDetails` with `type`, `title`, `status`, `detail`, `instance`.

> Note: PRD mentions Firebase Authentication for sign-up/login flow and JWTs for API protection—keep the API side purely JWT-based; any Firebase specifics happen in the auth domain’s implementation and token minting.
> We will reuse Firebase as the JWT authority (Auth only). Firestore is not used in this project. The middleware validates tokens against Firebase (verifying signature, audience, expiry) while the HTTP layer treats them as standard bearer tokens.

### Auth configuration

- `AUTH_PROVIDER=firebase|dev` (default `firebase`).
    - `firebase`: requires valid Firebase credentials (`FIREBASE_CONFIG` or ADC) and wires `platformauth.JWT(platformauth.FirebaseTokenVerifier(fbAuth), nil)`.
    - `dev`: wires `platformauth.JWT(platformauth.UnsignedTokenVerifier(), nil)` for local development. The verifier **does not** validate signatures; it simply decodes the JWT payload and copies claims (e.g., `email`, `name`, `isAdmin`, `firebase.tenant`). Use only in non-production environments and ensure your dev tokens never leak.

---

## 6) Domain Implementation Pattern

Each `/domains/<domain>/be` follows:

```
/domains/<domain>/be/
  handler/        # implements generated server interfaces; thin HTTP layer
  service/        # business logic; pure Go where possible
  repo/           # storage interface + adapters (DB, in-memory for tests)
  dto/            # mapping helpers to/from generated models if needed
  middleware/     # domain-specific hooks (rare)
  tests/
```

### Handler rules

* Only:

    * Parse/validate inputs (bind via generated types).
    * Call `service` methods.
    * Map service errors → `ProblemDetails`.
    * Set proper status codes (`201` with `Location`, `204` for deletes/empty).

### Service rules

* Stateless, testable; no HTTP, no DB specifics.
* Perform role checks if business-specific (beyond coarse middleware).
* Return domain errors (`ErrNotFound`, `ErrConflict`, `ErrInvalid`) for mapping.

### Repo rules

* Define interfaces (`Get`, `List`, `Create`, `Update`, `Delete`, `ApproveUser`, etc.).
* Provide an in-memory impl for tests; real impl left pluggable (DB not mandated here).

---

## 7) Errors & Logging

* Use structured logs (zap). No `fmt.Println` in production paths.
* Configure zap to emit Google Cloud Logging–compatible JSON (severity, message, trace) so logs flow directly into GCL without extra shims.
* Every error surfaced to HTTP is a **ProblemDetails** with stable `type` URIs (e.g., `"https://palmyra.pro/problems/validation-error"`). Log full internal error; never leak internals.

---

## 8) Pagination, Filtering, Sorting

* Always use shared `PaginationRequest` & `PaginationMeta` from `/contracts/common/pagination.yaml`.
* Handlers must read `page`, `pageSize`, `sort` query params and pass to repos.
* Responses: `{ items, page, pageSize, totalItems, totalPages }`.

---

## 9) Testing Requirements

* **Table-driven** unit tests for services and handlers (`testify`).
* Handlers: use httptest; assert status, body matches generated model or ProblemDetails.
* Include an MSW mock server on the frontend side (separate), but backend tests remain pure Go. (Mocks on FE are mentioned in PRD; keep parity.)

---

## 10) Performance & Ops

* Add gzip middleware (server-side) and ensure correct `Content-Encoding` negotiation.
* Timeouts: request timeout middleware (e.g., 15s default).
* Idempotency: GET safe; PUT idempotent; DELETE idempotent by contract.
* Observability (stretch): expose request ID, structured logs with latency & status.

---

## 11) JetBrains (GoLand/IntelliJ) Setup for Junie

* **Run Configs**:

    * `apps/api`: `go run ./apps/api`
    * Domain tests: `go test ./domains/<domain>/be/...`
* **File Templates**: Create templates for `service.go`, `handler.go`, `repo.go`, `service_test.go`, `handler_test.go`.
* **Inspections**: Enable Go vet, unused, error check; reformat on save.
* **Live Templates**: snippets for ProblemDetails construction, handler skeleton, role guard.
* **External Tool**: add `oapi-codegen` as an External Tool action bound to contracts save (manual regen for now).

---

## 12) Done Definition (per endpoint)

* Contract exists/updated in `/contracts/<domain>.yaml` and references shared components correctly.
* Codegen re-run; no manual edits in `/generated`.
* Handler implemented and **covered** by unit tests (happy + error paths).
* Auth (JWT) enforced; role checking validated.
* Returns only allowed success code + default ProblemDetails.
* Pagination/filters honored (for list endpoints).
* Logs structured; no lints; `go test ./...` green.

---

## 13) The AI agent Execution Prompt (paste this before each backend change)

> **System**: Follow the Backend Common Guideline. Do not modify generated code. Implement handlers by fulfilling the generated Chi server interfaces from `/generated/go/<domain>`. Use JWT middleware from `/platform/go/auth`. Map all errors to RFC7807 `ProblemDetails` from `/contracts/common/problemdetails.yaml`. Respect `/api/v1` versioning and response rules.
> **Task**: For domain `<domain>`, implement the following endpoints as defined in `/contracts/<domain>.yaml`, wiring them in `/apps/api`. Add service + repo interfaces with in-memory impl and unit tests. Enforce roles `[ ... ]` where specified by the contract/PRD. Only return success code X and default error.

---

## 14) Domain Notes for Current Scope (MVP)

* **Auth**: Signup/Login/Refresh; public routes only for these. Everything else requires bearer token.
* **Users**: Admin actions: approve/reject/enable/disable; list with filters (name, email, status) and pagination. Roles: `admin`, `user_manager`.
* **Schema Categories**: Admin-only CRUD for the hierarchical category tree backing schemas; all operations require `admin` role.
* **Schema Repository & Entities**: Manage versioned JSON Schemas (`schema-repository` domain) and JSON documents stored per schema/table (`entities` domain) as defined in the contracts and persistence-layer docs.

---

### Quick Acceptance Snippets (curl)

```bash
# Login
curl -s -X POST /api/v1/auth/login -d '{"email":"a@b","password":"***"}'

# List users (requires JWT)
curl -H "Authorization: Bearer $JWT" "/api/v1/users?page=1&pageSize=20&status=pending"

# Approve user
curl -X POST -H "Authorization: Bearer $JWT" "/api/v1/users/{id}:approve"
```

---

## Appendix A — Playbooks (Copy/Paste)

Playbook A1 — Contracts → Go codegen → Domain update
- Step 1: Edit contracts in `contracts/<domain>.yaml`; reuse `$ref: "./common/..."`.
- Step 2: Regenerate Go stubs: `go generate ./tools/codegen/openapi/go`.
- Step 3: Implement/adjust handlers in `domains/<domain>/be/handler` to satisfy generated interfaces.
- Step 4: Update services/repos as needed; keep JSON camelCase.
- Step 5: Wire routes in `apps/api` if new endpoints added.
- Validation:
  - `go build ./apps/api` succeeds
  - `go test ./domains/<domain>/be/...` green
  - New endpoints return one success code + default ProblemDetails

Playbook A2 — Add a new backend domain
- Create: `domains/<name>/be/{handler,service,repo,tests}`.
- Add contract: `contracts/<name>.yaml` and run codegen.
- Implement generated server interfaces in `handler/` and mount in `apps/api`.
- Add in-memory repo for tests; persist layer via `platform/go/persistence` when ready.
- Validation: domain tests pass; `apps/api` compiles; lint clean; responses match contract.

Playbook A3 — Protect a route with roles
- Ensure middleware: `platform/go/auth` JWT + `RequireRoles(...)`.
- In handler/service, enforce finer checks if business-specific.
- Confirm 401/403 return RFC7807 body with correct `status` and `type`.

---

## Appendix B — Checklists

Before PR
- [ ] BE-CON-001 Contract updated or confirmed unchanged
- [ ] BE-GEN-002 No edits under `/generated` (regen done)
- [ ] Tests added/updated (`go test ./...` green)
- [ ] Only one success code + default ProblemDetails
- [ ] JWT and roles enforced where required
- [ ] `go fmt ./...` applied

After a contract change
- [ ] `go generate ./tools/codegen/openapi/go`
- [ ] Handlers and DTOs aligned to new models
- [ ] Backward compatibility assessed (additive vs breaking)

---

## Appendix C — Common Pitfalls
- Editing generated files under `/generated`
- Returning ad-hoc error JSON instead of RFC7807 `ProblemDetails`
- Forgetting to set `Location` on 201 or `204` for empty deletes
- Bypassing JWT/role checks on admin endpoints
- Diverging field casing from camelCase or skipping pagination metadata

---

## Appendix D — Validation Gates
- Compile: `go build ./apps/api`
- Format: `go fmt ./...`
- Tests: `go test ./...`
- Manual: curl the new/changed endpoints; verify success + default error only

---

## Appendix E — Rule IDs Quick Reference
- BE-CON-001: Contract-first
- BE-GEN-002: Generated is read-only
- BE-HTTP-003: Two-response policy
- BE-AUTH-004: JWT + roles
- BE-JSON-005: JSON casing/time/ids
- BE-ROUTE-006: Path conventions
