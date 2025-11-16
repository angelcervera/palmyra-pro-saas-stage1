# @zengateglobal/api-sdk

Publishable TypeScript SDK for the Palmyra Pro API. Wraps generated OpenAPI clients and provides a thin HTTP helper for auth and ProblemDetails.

- ESM output, TypeScript types
- Generated clients are emitted into `packages/api-sdk/src/generated/<domain>` by the codegen step
- Consume via the SDK from apps; do not import generated sources directly

## Usage

Install (internal apps via workspace):

```
pnpm add @zengateglobal/api-sdk
```

Create client:

```ts
import { createFetchClient } from '@zengateglobal/api-sdk'

const api = createFetchClient({
  baseUrl: import.meta.env.VITE_API_BASE_URL ?? '/api/v1',
  getToken: () => sessionStorage.getItem('jwt') ?? undefined,
})

// If you render on the server, pass an absolute base URL:
// createFetchClient({ baseUrl: 'https://api.example.com/api/v1', getToken })

// Example: await api('users')

// Domain helpers (Auth, Users, SchemaCategories, …) are exposed via namespaces:
// import { Users } from '@zengateglobal/api-sdk'
// const result = await Users.usersList({ query: { page: 1, pageSize: 20 } })
```

## Build

```
pnpm -F @zengateglobal/api-sdk build
```

This will compile sources (including `src/generated`) and emit ESM to `dist/`.

## Regenerate OpenAPI Clients

Whenever the contracts under `contracts/*.yaml` change, regenerate the TypeScript clients from the **repository root**:

```
pnpm run openapi:ts
```

This script runs the shared OpenAPI-to-TypeScript generator and overwrites the `src/generated/<domain>` folders.

## Publish

```
pnpm -F @zengateglobal/api-sdk publish --access public
```

Ensure `.npmrc` maps the `@zengateglobal` scope to your registry.
