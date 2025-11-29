FROM node:24-bookworm

# Enable pnpm via corepack.
RUN corepack enable && corepack prepare pnpm@10.20.0 --activate

WORKDIR /workspace

# Default command installs repo deps, installs Playwright browser deps/binaries
# using the repo's pinned version, then runs the offline demo E2E suite.
CMD pnpm install --no-frozen-lockfile \
 && pnpm -C apps/demos/offline exec playwright install-deps chromium \
 && pnpm -C apps/demos/offline exec playwright install chromium \
 && pnpm -C apps/demos/offline test:e2e
