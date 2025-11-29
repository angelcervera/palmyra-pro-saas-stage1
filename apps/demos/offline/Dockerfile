# syntax=docker/dockerfile:1.7

FROM node:24-bookworm AS base
WORKDIR /app
ENV PNPM_HOME=/usr/local/share/pnpm
ENV PATH="${PNPM_HOME}:$PATH"
RUN corepack enable

FROM base AS deps
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY biome.json ./
COPY tools ./tools
COPY apps ./apps
COPY domains ./domains
COPY packages ./packages
COPY platform ./platform
RUN pnpm install --frozen-lockfile

FROM deps AS build
RUN pnpm -C apps/demos/offline build

FROM nginx:1.27-alpine AS runner
WORKDIR /usr/share/nginx/html
COPY apps/demos/offline/docker/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/apps/demos/offline/dist ./
EXPOSE 4174
CMD ["nginx", "-g", "daemon off;"]
