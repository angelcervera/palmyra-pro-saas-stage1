# Offline Demo E2E

This folder contains the Playwright-based E2E setup for the offline demo app.

## Run

From the repo root:

```bash
docker compose -f apps/demos/offline/e2e/docker-compose.yml up --build --abort-on-container-exit --exit-code-from playwright
```

The stack builds a Playwright runner (Node 24), installs browsers at runtime, starts the offline demo via the Playwright `webServer` hook, and runs `pnpm -C apps/demos/offline test:e2e`.
