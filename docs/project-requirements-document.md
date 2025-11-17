# Project Requirements Document

## Overview & Goals

ZenGate Global operates several data‑intensive workflows that require accurate, well‑governed structured data across products and teams.

Today, critical datasets are fragmented across systems, with inconsistent schemas, limited validation, and no single source of truth for how entities are defined or stored.

The goal of this project is to build Palmyra Pro: a contract‑first platform centered on versioned JSON Schemas and the entities that conform to them, providing a governed, continuously updated data backbone for ZenGate services.

## User Flow

The main sidebar exposes navigation entries for the primary admin areas (Users, Schema Categories, Schema Repository, Entities, etc.).

The main content area adapts dynamically to show the relevant CRUD interface based on the selected section.

### Users admin

The Authentication system uses Firebase Authentication and JWT tokens for secure access.

The application provides a public sign-up page accessible to all users.

Newly registered users are added to an approval list and cannot access the system until approved.

An existing user with the admin role can review and manage pending users through an administration page that lists all
registered users.

For each user, the admin can perform the following actions:

- Approve or Disapprove a pending user.
- Enable or Disable an existing user.

The administration page includes filters to refine the list by:

- User name
- User email
- User status: rejected, pending, enabled, disabled

### Schema & Entity Management

Beyond user administration, the application exposes CRUD flows for:

- **Schema Categories**: administrators manage a hierarchical tree of categories used to organize schemas. Each category has a slug, name, optional parent, description, and audit fields.
- **Schema Repository**: administrators create and inspect versioned JSON Schemas. Each schema version captures a `schemaId`, semantic `schemaVersion`, JSON Schema definition, table name, slug, category link, and lifecycle flags (timestamps, `isActive`).
- **Entities**: authorized users work with JSON documents stored per schema/table. The UI surfaces paginated lists backed by the persistence layer, with filtering, sorting, and detail views that respect the underlying schema definitions.

All these experiences follow the CRUD UI guideline and API conventions (pagination, ProblemDetails) defined elsewhere in this repo.

## Core Features & Scope

The features included in the current scope are:

- Authentication
- Authorization with support for multiple user roles: admin, user_manager
- User Sign-Up
- User Management including CRUD, filtering and pagination

All other features are considered out of scope for this phase.

## Tech Architecture

Key architectural principles:

- OpenAPI-driven / contract-first design: The OpenAPI specification serves as the single source of truth and contract
  between the backend
  and frontend layers.
- Monolithic Frontend (domain-based): The frontend is organized by domains, with separate pages and models for each
  domain.
- Monolithic Backend (domain-based): The backend follows the same domain-based structure, with dedicated services and
  models per domain.
- Monorepo setup: A unified repository simplifies dependency management, deployments, and release orchestration.
- Twelve-Factor alignment: services should follow [The Twelve-Factor App](https://12factor.net/) guidance (config via env, stateless processes, logs as event streams, etc.) wherever feasible to keep deployment and operations consistent.
- Configuration management uses [`envconfig`](https://github.com/kelseyhightower/envconfig) to load settings from environment variables, keeping parity with the Twelve-Factor expectations while remaining simple.

This architecture promotes decoupling and scalability, both in terms of performance and development efficiency.

## Tech Stack

### Monorepo

The project is organized as a **polyglot monorepo** managed with **Bazel 8 (bzlmod)** to guarantee hermetic,
reproducible builds and consistent development workflows across multiple languages.
It serves as the single source of truth for the entire platform, containing the **frontend (TypeScript/React)**, *
*backend (Go/Chi)**, **OpenAPI contracts**, and **automatically generated SDKs**.

#### Structure and Organization

The repository is organized **vertically by domain**, not by technology.
Each domain (e.g., `auth`, `users`, `schema-categories`, `schema-repository`, `entities`) contains both its backend code and, where applicable, frontend modules:

```
/contracts/                      # OpenAPI definitions (one per domain)
/contracts/auth.yaml             # signup/login/refresh/logout + securitySchemes
/contracts/users.yaml            # user CRUD/listing + admin actions
/contracts/schema-categories.yaml  # admin CRUD for schema category hierarchy
/contracts/schema-repository.yaml  # schema definitions and versions
/contracts/entities.yaml         # JSON entity documents per schema/table
/contracts/common/               # Shared OpenAPI components (Pagination, Address, ProblemDetails, etc.)
/contracts/common/info.yaml
/contracts/common/primitives.yaml      # UUID, Timestamp, Email, Slug, Code
/contracts/common/problemdetails.yaml  # ProblemDetails, StandardError
/contracts/common/pagination.yaml      # PaginationRequest, PaginationMeta, params
/contracts/common/security.yaml        # bearerAuth + global security
/contracts/common/iam.yaml             # UserRole, Permission, Claim, maybe Policy (shared IAM types)
/domains/
  auth/
    fe/                    # React components, stores, hooks
    be/                    # Go models, handlers, services
  users/
    fe/                    # React components, stores, hooks
    be/                          # Go models, handlers, services
  schema-categories/
    fe/                          # React components, stores, hooks
    be/                          # Go models, handlers, services
  schema-repository/
    fe/                    # React components, stores, hooks
    be/                          # Go models, handlers, services
  entities/
    be/                          # Go models, handlers, services
/generated/                      # Codegen output from OpenAPI (Go + TS SDK)
  go/
    auth/
    users/
    schema-categories/
    schema-repository/
/packages/api-sdk/src/generated/ # TypeScript clients and types
  auth/
  users/
  schema-categories/
  schema-repository/
/platform/
  ts/                            # Shared frontend utilities, UI, form and auth helpers
  go/                            # Shared backend libraries (logging, config, middleware, persistence)
/apps/
  web-admin/                     # React 19 + Vite admin application shell
  api/                           # Go API entrypoint (Chi router)
```

The `/generated` directory is intentionally **separated from the domain folders**, because it contains **build artifacts
**, not human-edited source code.
This separation prevents circular dependencies and ensures a clean one-way flow:

```
contracts → generated → domains
```

It also allows Bazel to cache and rebuild SDKs independently, simplifies CI/CD, and supports both **TypeScript** and *
*Go** code generation under a single structure.

Cross-domain dependencies (e.g., shared UI, auth, or logging) are stored under `/platform/` to promote reuse and
consistency.
Shared OpenAPI components such as **Pagination**, **Address**, **Error/ProblemDetails**, and **Standard Responses** are
defined once under `/contracts/common/` and referenced by all domain specifications through `$ref`.
This ensures consistency and prevents schema duplication across domains.

#### Dependency Management

The repository uses a **single root `go.mod`** and **single root `package.json`** to unify dependency versions and
simplify updates.
The frontend uses **pnpm 10.x (>=10.20.0 <11)** for workspace management, while the backend uses standard Go modules.
All dependencies are declared at the root level and consumed via Bazel-managed toolchains.

#### Build System and Toolchains

In the future, it will use Bazel 8 handles all builds, tests, and dependency resolution through:

* **rules_go** + **gazelle** for Go builds, linting, and dependency resolution.
* **rules_js** and **rules_ts** (Aspect Workflows) for TypeScript and NodeJS builds.
* **rules_oci** (optional) for container packaging and deployment automation.
* **bzlmod** (new module system) for reproducible external dependencies and toolchain registration.

Each domain defines its own Bazel packages and targets.
Build outputs are cached, incremental, and hermetic — allowing consistent local and CI/CD environments.

#### OpenAPI-driven Code Generation

Every domain maintains its own **OpenAPI 3.0.4+ definition**, serving as the canonical contract between backend and
frontend.
A shared components library (`/contracts/common/`) contains reusable definitions such as:

* **Pagination**: common request and response models for paginated resources.
* **Address**: standardized postal address schema used across multiple domains.
* **ProblemDetails**: standardized error response format compatible with RFC 7807.
* **Standard Responses**: common success/error envelopes and metadata objects.

For simplicity and first version, we will have only two responses per path:
- Success response, with its own code depending on the type of request.
- Error response, market as `default` and type `application/problem+json`.

Each domain OpenAPI file references these shared components using `$ref: "../common/<component>.yaml"`.
This approach ensures consistency in validation and response shape while keeping each domain contract lightweight.

At the moment, we will run the script that generate the code manually. In the future, Bazel automates SDK generation through custom macros that run:
 
* `oapi-codegen` for Go (Chi server stubs and type-safe models)
* `@hey-api/openapi-ts` for TypeScript (frontend clients and Zod schemas)

Generated Go stubs are placed under `/generated/go/<domain>`. TypeScript clients are generated into the SDK at `packages/api-sdk/src/generated/<domain>`.
These SDKs are imported by domain packages in both backend and frontend, ensuring **zero drift between layers**.

Also, we will generate a Mock implementation using MSW (Mock Service Worker).

#### CI/CD Integration

* **Consistency:** All environments (local, CI, production) build with the same Bazel configuration and dependency
  versions.
* **Caching:** Remote build caching reduces build times in CI.
* **Validation:** Codegen, linting, and tests run per domain before merging.
* **Deployment:** The backend (`apps/api`) and frontend (`apps/web-admin`) are packaged via Bazel and deployed to container or
  static hosting targets.

#### Benefits of This Approach

* **Strong modularity:** Each domain is a self-contained, testable unit.
* **Contract-first development:** Frontend and backend can be developed independently using mock OpenAPI servers.
* **Hermetic builds:** Eliminates “it works on my machine” issues.
* **Fast, incremental builds:** Only changed targets are rebuilt.
* **Scalable:** New domains or services can be added without restructuring the repo.
* **Unified versioning:** All languages and environments share a single build graph and dependency state.

### OpenAPI

The project uses **OpenAPI 3.0.4** or later to support embedded JSON Schemas.  
The API definition is modular, organized by domain.

More information, at the [API.md](api/api.md) document.

### Frontend

The frontend is a **React 19 + Vite SPA/PWA** designed for **offline-first operation** and **modular composition by
domain**.

Each domain exposes its own React components, stores, and hooks under `/domains/<domain>/fe`, while the `/apps/web-admin`
shell handles routing, layout, and PWA registration.

The application leverages **VitePWA (Workbox)** for runtime caching, background sync, and auto-updates, achieving full
PWA capabilities.

Key components:

- **Build Tool**: [Vite](https://vitejs.dev/)
- **VitePWA (Workbox)**: [vite-plugin-pwa](https://vite-plugin-pwa.netlify.app/)
- **Package Manager**: [pnpm](https://pnpm.io/) 10.x (>=10.20.0 <11)
- **Framework & Language**: React 19 with TypeScript 5, running on Node.js 24
- **UI Library**: [ShadCN UI](https://ui.shadcn.com/docs/components), based on [Tailwind CSS](https://tailwindcss.com/)
- **Additional Components**: [Radix UI](https://www.radix-ui.com/) for faster prototyping
- **Form Management**: [React Hook Form](https://react-hook-form.com/)
- **Schema Validation**: [Zod](https://zod.dev/)
- **Open API Client Generation**: [@hey-api/openapi-ts](https://heyapi.dev/) (compatible with OpenAPI 3.0.4)
- **State Management**: [Nanostores](https://github.com/nanostores/nanostores/)
- **Routing**: [React Router](https://reactrouter.com/)
- **Internationalization (i18n)**: [react-i18next](https://react.i18next.com/)
- **Date Handling**: [date-fns v4](https://date-fns.org/)
- **Testing**:
  [Vitest](https://vitest.dev/), [React Testing Library](https://testing-library.com/docs/react-testing-library/intro/),
  and
  [Cypress](https://www.cypress.io/)
- **Linting & Formatting**: [ESLint](https://eslint.org/) and [Prettier](https://prettier.io/)

### Backend

The Banckend is built with **Go 1.25+**, following a **domain-based monolithic architecture**.

Each domain encapsulates its own models, services, handlers, and tests under `/domains/<domain>/be`.

Key components:

- **Rest API Framework**: [Chi](https://github.com/go-chi/chi)
- **Database Driver**: [`pgxpool`](https://github.com/jackc/pgx) for PostgreSQL connection pooling, instrumentation, and middleware-friendly schema injection.
- **Structured Logging**: [Zap](https://github.com/uber-go/zap) configured to emit Google Cloud Logging JSON (severity, trace, labels) so API logs ingest natively in GCL.
- **Testing**: [Testify](https://github.com/stretchr/testify) powers unit/service/handler tests (assertions, suites, mocks) to keep Go coverage consistent.
- **JSON Validation**: [`santhosh-tekuri/jsonschema`](https://github.com/santhosh-tekuri/jsonschema) validates entity payloads against schema-repository definitions before persistence.
- **Persistence Testing**: Back-end integration tests use [Testcontainers](https://testcontainers.com/) to spin up real PostgreSQL instances so persistence logic reflects actual database behavior.

## Design Guidelines

To ensure consistency, maintainability, and a high-quality user experience across the application, the following design
guidelines must be followed:

### General Layout

The app layout is based on the ShadCN UI Dashboard pattern, aligning with common admin interfaces such as Vercel.

Layout consists of:

- Sidebar Navigation: Persistent, collapsible on both mobile and desktop, with toggles using state or media queries.
- Top Bar: Hosts profile access, actions, and (optional) search or notifications.
- Main Content Area: Flexible container that changes based on the selected route (Users, Schema Categories, Schema Repository, Entities, Admin, etc.).

### Component Standards

UI components must follow ShadCN UI standards and use primitives from Radix UI where needed.

Whenever possible, reuse pre-built ShadCN components and blocks. Before creating a custom UI element, confirm that no suitable ShadCN component (including the documented blocks) exists that can be composed to meet the requirement.

Form implementation must rely on React Hook Form + Zod for schema validation.

Follow Tailwind spacing, font, and size utility conventions to maintain visual consistency.

### Color Scheme - Default: Zinc

- The entire app must use ShadCN’s default `zinc` color palette: https://ui.shadcn.com/colors#zinc
- Use Tailwind utility classes aligned with this scheme (e.g., `bg-background`, `text-muted-foreground`,
  `border-input`).
- The UI must support both light and dark modes using ShadCN’s theming system. The `zinc` color palette should adapt
  naturally to each theme, ensuring consistent contrast ratios and readability.

### Responsive Design

- The UI must be fully responsive across breakpoints.
- Sidebar behavior:
    - Mobile: Hidden by default, shown via hamburger toggle.
    - Desktop: Collapsible, with toggle control to show/hide.
- Content must gracefully adapt using Tailwind responsive classes (`sm:`, `md:`, `lg:`).
- Ensure mobile-first layouts: vertical stacking, full-width buttons, touch-friendly inputs.
- The sidebar toggle state (expanded/collapsed) should persist between sessions using `localStorage` or an equivalent
  client-side storage mechanism.

### Reusability & Accessibility

- Design all components to be:
    - Composable and modular per domain.
    - Accessible (a11y): Use semantic HTML, correct ARIA roles, and keyboard support.
- All interactive elements (`buttons`, `toggles`, `links`) must have visible focus states and descriptive ARIA labels to
  ensure screen reader compatibility.

### Layout Composition

- Page structure:
    - Outer container: `max-w-screen-xl mx-auto px-4 sm:px-6 lg:px-8`
    - Element spacing: `space-y-4` for vertical flow
- Follow consistent layout patterns across all domains and CRUD interfaces.

### UI Consistency

- Components must use standardized ShadCN variants:
    - Buttons: `variant="default"`, `outline`, `destructive`
    - Inputs: Label above, `space-y-2` grouping
    - Tables: ShadCN data tables with sorting, filtering, pagination
- Keep iconography and spacing uniform across domains.
