# Offline Demo E2E

This folder contains the Playwright-based E2E setup for the offline demo.

## Run

From the repo root:

```bash
docker compose -f apps/demos/offline/e2e/docker-compose.yml up --build --abort-on-container-exit --exit-code-from playwright
```

The stack builds API, admin web, and a Playwright runner (Node 24) that installs browsers at runtime and runs `pnpm -C apps/demos/offline test:e2e`.
