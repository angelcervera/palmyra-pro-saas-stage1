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
COPY examples ./examples
RUN pnpm install --frozen-lockfile
RUN pnpm --filter @zengateglobal/persistence-sdk... build
RUN pnpm --filter @zengateglobal/demo... install --frozen-lockfile

FROM deps AS build
ARG VITE_API_BASE_URL=/api/v1
ARG VITE_API_TOKEN
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
ENV VITE_API_TOKEN=${VITE_API_TOKEN}
RUN pnpm -C examples/offline build

FROM nginx:1.27-alpine AS runner
WORKDIR /usr/share/nginx/html
COPY examples/offline/docker/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/examples/offline/dist ./
EXPOSE 4174
CMD ["nginx", "-g", "daemon off;"]
