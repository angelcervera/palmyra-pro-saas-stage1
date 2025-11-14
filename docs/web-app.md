---
id: web-app
version: 1.1.0
lastUpdated: 2025-11-02
appliesTo:
  - apps/web-admin
  - apps/web-pwa
  - domains/*/fe
  - packages/api-sdk/src/generated/*
  - platform/ts/*
relatedDocs:
  - docs/api.md
  - docs/api-server.md
  - docs/project-requirements-document.md
  - docs/adr/index.md
commandsUsed:
  - pnpm dev -C apps/web-admin
  - pnpm dev -C apps/web-pwa
  - pnpm build -C apps/web-admin
  - pnpm build -C apps/web-pwa
---

# Frontend Apps (Admin + PWA) Guideline

## 0) Golden Rules (Normative)

- FE-CON-001 (MUST) Contract-first: UI types originate from `/contracts/*.yaml` and `/contracts/common/*`. Never add fields not in the contract.
- FE-GEN-002 (MUST) Generated TS in `packages/api-sdk/src/generated/<domain>` is read-only; regen on contract change.
- FE-HTTP-003 (MUST) Handle two-response policy: one success code + default RFC7807 `ProblemDetails` on errors.
- FE-AUTH-004 (MUST) Attach JWT to all API calls except `POST /auth/signup` and `POST /auth/login`.
- FE-MOD-005 (MUST) Domain-first composition: consume domain exports in `/apps/web-admin` and `/apps/web-pwa`; no cross-domain internal imports.
- FE-JSON-006 (MUST) JSON payloads camelCase; align form fields and serializers to contracts.
- FE-TS-008 (MUST) TypeScript-only sources: use `.ts/.tsx` exclusively. Do not introduce `.js/.jsx`; migrate any JS to TS.
- FE-SDK-009 (MUST) Consume the publishable API SDK `@zengateglobal/api-sdk` from apps. Do not import from `packages/api-sdk/src/generated/*`; those files are internal to the SDK build.
- FE-DEP-010 (SHOULD) Prefer the latest stable versions when adding new dependencies and keep existing ones updated regularly; verify compatibility with React 19/Vite 7 toolchain before bumping majors.
- FE-PKG-007 (MUST) Use the `@zengateglobal` scope for all workspace package names. Mark app packages as `"private": true`; reference internal libs with `workspace:*` ranges.
- FE-UX-011 (MUST) Surface status, success, warning, and error feedback through the shared toast system (`sonner`). Wire toast notifications for optimistic success, validation issues, and fatal errors instead of ad-hoc alerts or inline banners.
- FE-UI-012 (MUST) Reach for shadcn/ui components and documented blocks before building new UI primitives; only implement custom components when no suitable shadcn composition satisfies the requirement.

---

## App Catalog

- Admin App (`apps/web-admin`)
  - Audience: internal administrators and user managers
  - Scope: CRUD and approval workflows (users, schema categories, schema repository, entities), advanced filtering/pagination, role‑gated UI
  - Online-only by default; graceful handling of transient network issues

- PWA App (`apps/web-pwa`)
  - Audience: operational users who need offline‑first access (e.g., field operations, inventory curation)
  - Scope: selected read/write flows optimized for intermittent connectivity, background sync for mutations, local caching of critical data
  - Uses VitePWA (Workbox) for installability, offline cache, updates

---

## 1) Monorepo Layout (Frontend)

```
/contracts/                     # OpenAPI per domain + shared components
  common/                       # ProblemDetails, Pagination, IAM, primitives

/domains/<domain>/fe            # Handwritten React modules: components, hooks, stores, views
  components/
  hooks/
  stores/
  views/
  tests/                        # Vitest + RTL; fixtures align with MSW

/packages/api-sdk/src/generated/<domain>/  # TS types/clients from OpenAPI (read‑only)

/platform/ts                    # Shared UI primitives, HTTP client, auth utilities, form helpers

/packages/api-sdk               # Publishable TS SDK wrapping generated clients (ESM), name: @zengateglobal/api-sdk
/apps/web-admin                 # React 19 + Vite admin application shell (routes, layout, theming)
/apps/web-pwa                   # React 19 + Vite PWA shell (offline‑first, installable)
```

Rules:
- Keep domain UI isolated inside `/domains/<domain>/fe`. Export public surface via an index to be consumed by `/apps/web-admin` and `/apps/web-pwa`.
- Only put cross‑cutting utilities in `/platform/ts` (UI primitives, auth, http client, date, i18n helpers).
- Do not import from other domain internals; depend on their exported surface.

---

## 2) Toolchain & Project Baseline

### 2.1 Versions
- Node.js: 24.x LTS
- Package manager: pnpm 10.x (>=10.20.0 <11)
- Framework: React 19 + TypeScript 5
- Bundler/Dev server: Vite
- UI: shadcn/ui + Tailwind CSS (Radix primitives)
- Forms: React Hook Form + Zod
- Routing: React Router
- Server state: TanStack Query (or equivalent) for caching, pagination, retries
- Testing: Vitest, React Testing Library, MSW; Cypress for e2e

### 2.2 Conventions
- Files: TypeScript (`.ts`, `.tsx`) only; strict mode enabled (no `.js/.jsx`).
- Styling: Tailwind utility classes; shadcn components for primitives; no ad‑hoc design tokens outside theme.
- i18n: `react-i18next`; keep copy in `/apps/web-admin/src/i18n` and `/apps/web-pwa/src/i18n` with domain‑scoped namespaces.
- Lint/format: ESLint + Prettier; rely on workspace config.
- Dependencies: check updates with `pnpm outdated` and upgrade proactively (minor/patch). For majors, test locally (`pnpm dev`, `pnpm build -C apps/web-admin`) before adopting.
- Providers live under `src/providers` (theme, i18n, query, auth). Wire them once in `src/main.tsx` so the entire app can consume shared context.

### 2.3 Workspace package naming (scope)
- Scope all workspace packages with `@zengateglobal`:
  - `apps/web-admin/package.json`: `"name": "@zengateglobal/web-admin", "private": true`
  - `apps/web-pwa/package.json`: `"name": "@zengateglobal/web-pwa", "private": true`
  - Libraries (e.g., `platform/ts`): `"name": "@zengateglobal/platform-ts"`
- Use `workspace:*` for internal dependencies, e.g. `"@zengateglobal/platform-ts": "workspace:*"`.
- If using a private registry, map the scope in `.npmrc` at repo root:
  - `@zengateglobal:registry=https://npm.pkg.github.com`

---

## 3) Contracts → TypeScript Codegen

- Treat OpenAPI specs in `/contracts/` as the source of truth.
- Generate TypeScript types and client into `packages/api-sdk/src/generated/<domain>` using `@hey-api/openapi-ts` with a config file at `tools/codegen/openapi/ts/openapi-ts.config.ts` (read‑only artifacts located within the SDK package).
- Package the clients via the SDK package at `/packages/api-sdk` and consume that from apps.
- Always reference shared components via `$ref: "./common/..."` so shared TS types are consistent (Pagination, ProblemDetails, IAM, primitives).

Client usage pattern:
- Use `createFetchClient` exported from `@zengateglobal/api-sdk` to:
  - Inject `Authorization: Bearer <token>` header
  - Handle `ProblemDetails` decoding
  - Emit structured debug logs in dev only
  - Map `429`, `5xx`, and network failures to retry policies used by TanStack Query
- Import domain namespaces (e.g., `Auth`, `Users`) from `@zengateglobal/api-sdk`; never import directly from `packages/api-sdk/src/generated/*`.

Run codegen:
- `pnpm openapi:ts` (uses the config file above)

---

## 4) UI System & Theming

- Use shadcn/ui components as the primitive layer; customize through Tailwind theme tokens.
- Dark/light theme via `data-theme` + CSS variables; persist preference in localStorage.
- Accessibility: follow WAI‑ARIA; keyboard navigation complete; use Radix primitives to ensure focus management.
- Do not duplicate component variants per domain; factor shared primitives into `/platform/ts/ui`.

---

## 5) Routing & Shell Composition

- The admin shell lives in `/apps/web-admin` and mounts domain routes.
- The PWA shell lives in `/apps/web-pwa` and mounts a subset of routes optimized for offline.
- Use React Router with lazy routes. Example structure:

```
/apps/web-admin/src
  main.tsx              # entrypoint
  app.tsx               # layout shell (navbar, sidebar, theme)
  routes.tsx            # route tree wiring domains
  providers/            # QueryClient, AuthProvider, I18nProvider
  features/
    users/              # optional wrappers delegating to domains/users/fe
```

- Domains expose route elements (views) and any loaders/actions via their FE module’s public API, e.g. `domains/users/fe` exports `UsersRoutes`.
- 404 and error boundaries are implemented at shell level; domain views can provide nested boundaries.
- The PWA app should avoid route trees that require full online hydration; prefer screens that can render from cache and sync when online.

---

## 6) AuthN & AuthZ (Frontend)

- Authentication: obtain a JWT via the auth flow (e.g., Firebase sign‑in + backend JWT minting, or direct backend login per domain contract).
- Storage: keep tokens in memory when possible; if persistence is needed, use `sessionStorage`. Avoid long‑lived `localStorage` to reduce risk. Never store refresh tokens in web‑visible storage.
- PWA: if background sync requires token access, use short‑lived tokens and renew on service worker boot via a message channel; never persist refresh tokens in the service worker.
- Propagation: attach `Authorization: Bearer <token>` to all API calls except `POST /auth/signup` and `POST /auth/login`.
- Role gating: use a lightweight front‑end guard to show/hide admin UI (e.g., `RequireRoles(['admin', 'user_manager'])`). This is UX only—server enforces true authorization.
- 401/403 handling: intercept responses; on 401 → sign‑in route; on 403 → show `ProblemDetails.title` and log `type` for diagnostics.
- Providers (wrap in `main.tsx`):
  - `ThemeProvider` (next-themes) manages dark/light/system themes.
  - `I18nProvider` (react-i18next) supplies translations; start with `en` and expand.
  - `QueryProvider` (TanStack Query) provides caching/retry hooks; use `useQuery`/`useMutation` in domain screens.
  - `AuthProvider` stores JWT + roles; exposes `useAuth()` and `RequireRoles()` helpers.
  - `api()` in `src/lib/api.ts` returns a `createFetchClient` instance configured with `VITE_API_BASE_URL` and current JWT.

---

## 7) Data Fetching, Caching, and Errors

- Use TanStack Query for server state with keys scoped by domain and parameters (e.g., `['users', {page, pageSize, sort}]`).
- Pagination: follow backend conventions; surface `Pagination` metadata from response envelope in table/grid components.
- ProblemDetails: decode RFC7807; display `title` and `detail`; include `instance` and request ID when available for support.
- Retries: enable limited retries for idempotent GETs; disable for mutations unless safe.
- PWA: queue mutations while offline (e.g., Workbox Background Sync). On reconnect, replay in order and reconcile via Query invalidations.
- Loading states: optimistic UI for fast mutations; rollback on error using Query’s `onError`.

---

## 8) Forms & Validation

- Forms use React Hook Form + Zod schemas colocated with the component.
- Map contract field names (camelCase) to form fields; keep server payloads aligned with OpenAPI.
- Show field‑level errors from both Zod and `ProblemDetails` `invalidParams` (if present in our error extension fields).
- Submit buttons disabled while pending; avoid double submission.

---

## 9) State Management

- Prefer local component state and server state (Query). Introduce a lightweight store (e.g., nanostores/zustand) only when multiple distant components must coordinate.
- Do not cache server state manually; rely on Query invalidation after mutations.

---

## 10) Testing Strategy

- Unit tests: Vitest + React Testing Library colocated under `__tests__` or `tests` next to code in each domain.
- Integration/mocks: use MSW to mock `/api/v1/...` endpoints; fixtures reflect OpenAPI types from `@zengateglobal/api-sdk` exports.
- E2E: Cypress against the built app with API either mocked via MSW or pointed to a dev backend.
- Assertions: verify success payloads, pagination metadata, and RFC7807 bodies for error paths.

---

## 11) Local Development

Prerequisites:
- Node 24, pnpm installed.

Commands:
- `pnpm install` — install workspace deps
- SDK: `pnpm -F @zengateglobal/api-sdk build` (or `pnpm -F @zengateglobal/api-sdk dev` if configured)
- Admin: `pnpm dev -C apps/web-admin`, `pnpm build -C apps/web-admin`
- PWA: `pnpm dev -C apps/web-pwa`, `pnpm build -C apps/web-pwa`
- Domain tests: `pnpm test -C domains/<domain>/fe`

Environment variables (Vite):
- Shared: `VITE_API_BASE_URL` (e.g., `/api/v1` when proxied), `VITE_ENV` (`development` | `staging` | `production`)
- Optional: `VITE_SENTRY_DSN`
- PWA: `VITE_APP_VERSION` (for update banners), any PWA feature flags

Dev server proxy example (Vite): proxy `/api` to backend during dev to avoid CORS.

---

## 12) Build & Deployment

- Build: Vite creates static bundles per app; outputs served by the chosen static host or behind the API reverse proxy.
- Asset hashing for cache‑busting; HTML references auto‑updated by Vite.
- Environment: inject `VITE_*` variables at build time via `.env.*` or CI env.
- CI: build each app independently, e.g. `pnpm build -C apps/web-admin` and `pnpm build -C apps/web-pwa`; run tests; publish artifacts.
- PWA specifics (`apps/web-pwa`):
  - Use `vite-plugin-pwa` with `registerType: 'autoUpdate'` and `workbox` runtime caching for `/api/v1/*` GETs
  - Precache app shell, fonts, icons; provide an offline fallback route
  - Implement update flow: listen to SW `waiting`/`controllerchange` and show an “Update available” banner
 - SDK package (`packages/api-sdk`):
   - ESM output with type definitions; semantic versioning; publish to `@zengateglobal` scope
   - Consumers: internal apps via `workspace:*`, external apps via private registry per `.npmrc`

---

## 13) Observability & Logging

- Log errors with stack traces in dev; silence console in production except error reporting.
- Optionally wire Sentry (or equivalent) via `/apps/web-admin/src/providers` and `/apps/web-pwa/src/providers`.
- Propagate backend `X-Request-ID` in error UIs when available to correlate client reports with server logs.
- PWA: log SW lifecycle events (installed/updated) and surface current `VITE_APP_VERSION` for support.

---

## 14) Security & Privacy

- Do not embed secrets in the bundle; use env vars only for non‑secret config.
- Use HTTPS only; same‑site cookies if cookies are introduced.
- Sanitize any HTML content; avoid `dangerouslySetInnerHTML`.
- Keep dependencies up to date; rely on pnpm dedupe and lockfiles.

---

## 15) Performance

- Code‑split by route; lazy load heavy domain screens.
- Use React 19 features (e.g., `use` with Suspense) where beneficial; wrap with sensible fallbacks.
- Virtualize long lists (e.g., card grids) and paginate on the server.
- Avoid unnecessary re‑renders by memoizing large tables and selectors.
- PWA: prefer small, cacheable chunks; prefetch critical routes; tune Workbox cache TTLs and max entries.

---

## 16) Backwards Compatibility

- Frontend follows the contracts: when an endpoint or schema changes, regenerate the SDK sources (codegen) and update the affected domain UI.
- Tolerate additive changes (new fields). Treat breaking contract changes as coordinated migration PRs.

---

## 17) Error Handling Patterns

- Centralize error mapping in `/platform/ts/http`: network → `offline`, 401 → `unauthorized`, 403 → `forbidden`, 404 → `notFound`, `ProblemDetails` → `apiError` with surfaceable `title/detail`.
- Show non‑blocking toasts for background operation failures; show inline errors for form submissions.
- For 5xx, allow retry affordances on list/detail screens.

---

## 18) Appendix — Example Wiring

Domain export (users):
```ts
// domains/users/fe/index.ts
export { UsersRoutes } from './views/UsersRoutes';
export { useUsersQuery, useApproveUser } from './hooks/users';
```

Shell composition (admin):
```tsx
// apps/web-admin/src/routes.tsx
import { UsersRoutes } from '../../domains/users/fe';

export const routes = [
  { path: '/users/*', element: <UsersRoutes /> },
  // ...other domain routes
];
```

Shell composition (pwa):
```tsx
// apps/web-pwa/src/routes.tsx
import { SchemaCategoriesRoutes } from '../../domains/schema-categories/fe';

export const routes = [
  { path: '/schema-categories/*', element: <SchemaCategoriesRoutes /> },
  // lean subset for offline‑first flows
];
```

HTTP client:
```ts
// platform/ts/http/client.ts
export function createClient(getToken: () => string | undefined) {
  return async function request(input: RequestInfo, init: RequestInit = {}) {
    const token = getToken();
    const headers = new Headers(init.headers);
    if (token) headers.set('Authorization', `Bearer ${token}`);
    headers.set('Accept', 'application/json');
    const res = await fetch(input, { ...init, headers });
    if (!res.ok) {
      const maybeProblem = await res.clone().json().catch(() => undefined);
      throw { status: res.status, problem: maybeProblem };
    }
    return res.json();
  };
}
```

This guideline ensures the web app remains contract‑first, domain‑oriented, and consistent with the backend conventions defined in `docs/api.md` and `docs/api-server.md`.

---

## Appendix A — Playbooks (Copy/Paste)

Playbook A1 — Contracts → TS codegen → UI update
- Step 1: Edit contracts in `contracts/<domain>.yaml`; reuse `$ref: "./common/..."`.
- Step 2: Regenerate TS types/clients into `packages/api-sdk/src/generated/<domain>` (example):
  - `pnpm dlx @hey-api/openapi-ts@0.86.11 -i contracts/<domain>.yaml -o packages/api-sdk/src/generated/<domain>`
- Step 3: Update domain FE code under `domains/<domain>/fe` to use new types; adapt forms and views.
- Step 4: Verify admin shell compiles: `pnpm build -C apps/web-admin` (and PWA if relevant).
- Step 4.1: Rebuild the SDK package: `pnpm -F @zengateglobal/api-sdk build` and update app deps if versioned.
- Validation:
  - Type errors resolved in domain FE
  - Runtime calls handle new/changed fields

Playbook A2 — Add a new frontend domain
- Create: `domains/<name>/fe/{components,hooks,stores,views,tests}` with an `index.ts` that exports routes and hooks.
- Generate client/types for the new contract into `packages/api-sdk/src/generated/<name>`.
- Wire routes in `apps/web-admin/src/routes.tsx` (and in `apps/web-pwa` if included).
- Expose domain exports from the SDK (types/services) as needed for external consumers.
- Add tests (Vitest/RTL) for views and hooks; MSW fixtures reflect generated types.

Playbook A3 — Enable offline for a route (PWA)
- Add route to `apps/web-pwa` and ensure it can render from cached data.
- Configure Workbox runtime caching for its GET endpoints; choose `StaleWhileRevalidate` or `NetworkFirst` per UX.
- Queue mutations via Background Sync and replay on reconnect; invalidate TanStack Query keys.
- Implement an update banner reacting to SW `waiting` → `skipWaiting` → `controllerchange`.

Playbook A4 — Publish the API SDK (@zengateglobal/api-sdk)
- Ensure contracts and SDK generated sources are up to date
- Build the SDK: `pnpm -F @zengateglobal/api-sdk build`
- Bump version (semver) and publish: `pnpm -F @zengateglobal/api-sdk publish --access public` (or private registry)
- Validate install from a clean app: `pnpm add @zengateglobal/api-sdk@<version>`

---

## Appendix B — Checklists

Before PR
- [ ] FE-CON-001 Contract fields match UI types
- [ ] FE-GEN-002 No edits under `packages/api-sdk/src/generated` (regen done)
- [ ] FE-AUTH-004 JWT added to all protected API calls
- [ ] Two-response policy handled (ProblemDetails surfaced)
- [ ] Pagination metadata rendered where applicable
- [ ] No `.js/.jsx` files introduced; TS-only sources
- [ ] Apps import from `@zengateglobal/api-sdk` (no direct `packages/api-sdk/src/generated/*` imports)

After a contract change
- [ ] TS codegen updated in `packages/api-sdk/src/generated/<domain>`
- [ ] Domain forms/views updated
- [ ] Admin shell builds; PWA shell builds (if impacted)

---

## Appendix C — Common Pitfalls
- Editing generated TS under `packages/api-sdk/src/generated`
- Ignoring RFC7807 details; not surfacing `title/detail`
- Manually caching server state instead of using Query invalidation
- Storing refresh tokens in web-visible storage or service worker
- Diverging from camelCase field names

---

## Appendix D — Validation Gates
- Admin build: `pnpm build -C apps/web-admin`
- PWA build: `pnpm build -C apps/web-pwa`
- Domain tests: `pnpm test -C domains/<domain>/fe`
- Manual: simulate offline in DevTools; verify PWA screens render and queued mutations replay

---

## Appendix E — Rule IDs Quick Reference
- FE-CON-001: Contract-first
- FE-GEN-002: Generated TS read-only
- FE-HTTP-003: Two-response policy handling
- FE-AUTH-004: JWT on all protected calls
- FE-MOD-005: Domain-first composition, no cross-domain internals
- FE-JSON-006: JSON camelCase alignment
