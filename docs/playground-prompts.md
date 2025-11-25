# Playground Prompts

## CRUD Template (Codex)

**Prompt:**  
Implement full CRUD support for the `{{RESOURCE}}` domain across backend and frontend, reusing existing components wherever possible. When new UI is required, consume shadcn/ui components or blocks pulled from its registry rather than hand‑rolling custom elements.

### 1. Contract & Codegen
1. Update `contracts/{{resource}}.yaml` to describe the new domain (including optional update fields like `slug` if applicable).
2. Regenerate artifacts:
   - `go generate ./tools/codegen/openapi/go`
   - `pnpm openapi:ts`

### 2. Backend
1. Handler (`domains/{{resource}}/be/handler/handler.go`):
   - Map request bodies to service inputs, handle new fields, and reuse existing error helpers.
2. Service (`domains/{{resource}}/be/service/service.go`):
   - Add validation, slug normalization, and parent checks.
   - Ensure new fields flow through update/create paths.
3. Persistence (`platform/go/persistence/{{resource}}_repository.go`):
   - Extend create/update methods to store new attributes.
4. Tests:
   - Update service tests and handler tests to cover happy paths and validation errors.

### 3. API Client & Hooks
1. Expose a client factory in `apps/web-platform-admin/src/lib/api.ts` (e.g., `{{resource}}Client()`).
2. Create React Query hooks under `apps/web-platform-admin/src/app/{{resource}}/use-{{resource}}.ts`:
   - List, get, create, update, delete wrappers that reuse the generated SDK.

### 4. Frontend Pages (ShadCN)
1. Listing page (`apps/web-platform-admin/src/app/{{resource}}/page.tsx`):
   - Use shadcn blocks/components (table, dropdown, pagination) to render data.
   - Include column visibility toggles, row selection, and delete actions.
2. Shared form (`apps/web-platform-admin/src/app/{{resource}}/{{resource}}-form.tsx`):
   - Build with shadcn form primitives (`@/components/ui/form`, `Input`, `Textarea`, etc.).
3. Create & edit pages (`/new`, `/:id/edit`):
   - Reuse the form and hooks; surface toast notifications for success/error.
4. Routing:
   - Register pages in `apps/web-platform-admin/src/routes.tsx`.

### 5. Validation & Builds
1. Run backend tests: `GOCACHE=$(pwd)/.gocache go test ./...`
2. Build SDK: `pnpm -F @zengateglobal/api-sdk build`
3. Build admin app: `pnpm -C apps/web-platform-admin build`

**Reminder:** Always prefer existing components/blocks. When something new is required, import and compose shadcn-provided pieces instead of authoring custom UI from scratch.  
