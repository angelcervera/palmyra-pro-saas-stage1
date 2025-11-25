# Admin Web App

React 19 + Vite dashboard shell for ZenGate Global’s Palmyra Pro internal tooling. The app consumes the contract‑driven SDK (`@zengateglobal/api-sdk`), uses shadcn/ui for primitives, and shares providers (theme, auth, i18n, TanStack Query) with the rest of the frontend stack.

## Prerequisites
- Node.js 24.x
- pnpm 10.20.0 (enforced via `packageManager`)
- Go API running locally or proxied to `/api/v1` for real data

Install workspace dependencies from the repo root:

```bash
pnpm install
```

## Scripts

| Command | Description |
| --- | --- |
| `pnpm dev -C apps/web-admin` | Start the Vite dev server (watches files, enables React Query Devtools in dev).
| `pnpm build -C apps/web-admin` | Type-check and produce production assets (must run after every FE change per build gate).
| `pnpm preview -C apps/web-admin` | Preview the production build locally.
| `pnpm lint -C apps/web-admin` | Run ESLint with the flat config bundle.

> **Build Gate**: before pushing or opening a PR, run `pnpm build -C apps/web-admin` and fix any errors. This mirrors the requirement in `AGENTS.md`.

## Environment Variables
Copy `.env.example` to `.env` or set the variables before running the dev server:

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `VITE_API_BASE_URL` | ✅ | `/api/v1` | Base path for the API client; proxied to the Go backend during dev.
| `VITE_ENV` | ☐ | — | Optional label used for environment badges/logging (`development`, `staging`, `production`).
| `VITE_SENTRY_DSN` | ☐ | — | Optional DSN for browser error reporting.

The dev server reads `import.meta.env.*`; no rebuild is needed when variables change—restart `pnpm dev` to pick up new values.

## Project Layout
```
apps/web-admin/
  src/
    App.tsx            # renders route tree + devtools (in dev)
    main.tsx           # provider stack + BrowserRouter
    routes.tsx         # React Router config
    providers/         # theme, auth, i18n, TanStack Query contexts
    components/        # shadcn/ui + navigation + dashboard widgets
    app/               # route-aligned pages (dashboard, users, schema-categories)
  .env.example         # sample env vars
  package.json         # pnpm workspace config for this app
```

## Providers
`main.tsx` wraps the app with:
- `ThemeProvider` (next-themes) → dark/light/system theme handling.
- `BrowserRouter` → React Router entry point.
- `I18nProvider` (react-i18next) → translations.
- `QueryProvider` (TanStack Query) → data fetching cache/devtools.
- `AuthProvider` → JWT & role state exposed via `useAuth()`/`RequireRoles()`.

`src/lib/api.ts` returns a `createFetchClient` configured with `VITE_API_BASE_URL` and the current JWT.

## Debugging
- React Query Devtools load automatically in development (`import.meta.env.DEV`). Use the bottom-right toggle button.
- ESLint (flat config) includes React 19 compiler checks. Some template components disable specific rules until the custom UI replaces them.

## Additional Docs
- [docs/web-app.md](../../docs/web-app.md) — full frontend guidelines (routing, providers, codegen).
- [docs/project-requirements-document.md](../../docs/project-requirements-document.md) — product requirements & domain context.
