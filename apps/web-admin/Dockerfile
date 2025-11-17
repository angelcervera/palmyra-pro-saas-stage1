# syntax=docker/dockerfile:1.7
# Multi-stage build for the admin web application

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
ARG VITE_API_BASE_URL=/api/v1
ARG VITE_ENV=development
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
ENV VITE_ENV=${VITE_ENV}
RUN pnpm --filter @zengateglobal/web-admin... build

FROM nginx:1.27-alpine AS runner
WORKDIR /usr/share/nginx/html
COPY apps/web-admin/docker/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/apps/web-admin/dist ./
EXPOSE 4173
CMD ["nginx", "-g", "daemon off;"]
