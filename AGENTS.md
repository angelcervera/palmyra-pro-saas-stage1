# Repository Guidelines

## Context Ingestion Requirements
- Before handling any prompt, read and internalize the material in `docs/api.md`, `docs/api-server.md`, `docs/project-requirements-document.md`, and `docs/persistent-layer.md`.
- Maintain the information from these documents in the active context for every response.
- For any prompt involving CRUD creation or modification, also consume `docs/crud-ui-guideline.md` and `docs/playground-prompts.md` before proceeding.

## Project Structure & Module Organization
- Treat the OpenAPI specs in `contracts/` as the source of truth; shared schemas live in `contracts/common/` (ProblemDetails, Pagination, IAM, primitives).
- Generated code under `generated/go|ts/<domain>/` is read-only; when contracts change, regenerate instead of patching artifacts.
- Domain backend code sits in `domains/<domain>/be/{handler,service,repo,middleware,dto}` with test guidelines in `domains/<domain>/be/tests`.
- Frontend modules mirror the same vertical layout in `domains/<domain>/fe`, while shared utilities belong in `platform/go` and `platform/ts`; `apps/api` composes the Chi router and `apps/web-admin` hosts the admin React 19 shell.
- The shared persistence layer is implemented under `platform/go/persistence` and consumed by backend domains.

## Build, Test, and Development Commands
- `go generate ./tools/codegen/openapi/go` — rerun codegen with the config set whenever a contract changes.
- `go test ./...` — execute the domain suites (testify + httptest) and keep them green before reviews.
- `go fmt ./...` — rely on the formatter for Go sources; avoid manual whitespace tweaks.
- `pnpm install` — initializes the React workspace when frontend work resumes.

### Frontend Build Gate (must run after each FE change)
- After any change under `apps/web-admin`, `domains/*/fe`, or the SDK in `packages/api-sdk`, build before proceeding and fix any TypeScript errors:
  - Admin app: `pnpm build -C apps/web-admin`
  - SDK (when contracts or generated clients change): `pnpm -F @zengateglobal/api-sdk build`
- Do not submit changes with failing FE builds. Treat the build as a validation gate similar to `go test` for backend.

### Frontend Biome Gate (format → lint → check)
- Whenever modifying frontend TypeScript or React files (`apps/web-admin`, `packages/api-sdk`, `packages/persistence-sdk`), run Biome from each affected package before building:
  - `pnpm run format`
  - `pnpm run lint`
  - `pnpm run check`
- This ensures formatting, linting, and diagnostics stay consistent for the AI agent; backend Go code must continue to use `go fmt`.

## Coding Style & Contract Conventions
- Stay contract-first: update OpenAPI first, regenerate, then touch domain code.
- JSON payloads are camelCase; align struct tags with the schemas and reference shared components via `$ref`.
- Follow the two-response policy (one success code plus default ProblemDetails) and stage shared middleware in `platform/go`.
- Use [`pgxpool`](https://github.com/jackc/pgx) for PostgreSQL connectivity so every backend module relies on the same connection management primitives.
- Load configuration exclusively via [`envconfig`](https://github.com/kelseyhightower/envconfig)`;` keep settings in env vars to stay Twelve-Factor compliant.
- Emit structured logs with Zap configured for Google Cloud Logging (severity, trace, labels) instead of ad-hoc logging.
- Favor Twelve-Factor practices overall (stateless processes, config in env, logs as event streams) when making architecture decisions.
- Validate JSON payloads using [`santhosh-tekuri/jsonschema`](https://github.com/santhosh-tekuri/jsonschema) tied to the schema repository definitions before persisting or returning data.

## Testing Guidelines
- Add table-driven `*_test.go` cases alongside the code, using `testify` for assertions.
- Exercise handlers with `httptest`, checking success payloads, pagination metadata, and RFC7807 bodies.
- Start with in-memory repos or fakes; align fixtures with MSW mocks once the TypeScript SDK ships.
- Use Testcontainers when validating persistence logic so integration tests run against a real PostgreSQL instance and capture true database behavior.

## Security & Auth Expectations
- Require JWT bearer auth on every path except `POST /auth/signup` and `POST /auth/login`; wire the middleware from `platform/go/auth` in `apps/api`.
- Guard privileged routes with `RequireRoles` for `admin` and `user_manager`, returning ProblemDetails 401/403 when checks fail.
- Serve endpoints under `/api/v1/...` and emit structured zap logs with request IDs per `docs/api-server.md`.

## Commit & Pull Request Guidelines
- Use concise, present-tense commit subjects (~72 chars) and scope by domain when helpful (`users: implement approval workflow`).
- Commit regenerated artifacts and formatted code together; note the `go generate` and `go test` runs in the PR body.
- PRs should outline contract or behavior changes, list affected domains/apps, link issues, and add UI screenshots when frontend features shift; log future work (Bazel, more domains) as follow-ups.
