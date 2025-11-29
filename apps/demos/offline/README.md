# Offline Demo App

Lightweight, Dexie-backed demo UI for local CRUD. No API required.

## Develop locally

1) Start the backend stack (if desired) via Compose, omitting the offline demo so the port stays free:
```bash
docker compose up postgres api web-platform-admin
```

2) Run the offline app dev server:
```bash
pnpm -C apps/demos/offline dev --host --port 4174
```
Then open http://localhost:4174.

## Build
```bash
pnpm -C apps/demos/offline build
```

## E2E
```bash
docker compose -f apps/demos/offline/e2e/docker-compose.yml up --build --abort-on-container-exit --exit-code-from playwright
```
Runs a Playwright runner (Node 24) that installs browsers at runtime, starts the offline app via the Playwright `webServer` hook, and executes `pnpm -C apps/demos/offline test:e2e`.
