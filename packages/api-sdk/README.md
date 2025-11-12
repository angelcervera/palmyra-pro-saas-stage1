# @tcglanddev/api-sdk

Publishable TypeScript SDK for the TCG Land API. Wraps generated OpenAPI clients and provides a thin HTTP helper for auth and ProblemDetails.

- ESM output, TypeScript types
- Generated clients are emitted into `packages/api-sdk/src/generated/<domain>` by the codegen step
- Consume via the SDK from apps; do not import generated sources directly

## Usage

Install (internal apps via workspace):

```
pnpm add @tcglanddev/api-sdk
```

Create client:

```ts
import { createFetchClient } from '@tcglanddev/api-sdk'

const api = createFetchClient({
  baseUrl: import.meta.env.VITE_API_BASE_URL ?? '/api/v1',
  getToken: () => sessionStorage.getItem('jwt') ?? undefined,
})

// If you render on the server, pass an absolute base URL:
// createFetchClient({ baseUrl: 'https://api.example.com/api/v1', getToken })

// Example: await api('users')

// Domain helpers (Auth, Users, SchemaCategories, …) are exposed via namespaces:
// import { Users } from '@tcglanddev/api-sdk'
// const result = await Users.usersList({ query: { page: 1, pageSize: 20 } })
```

## Build

```
pnpm -F @tcglanddev/api-sdk build
```

This will compile sources (including `src/generated`) and emit ESM to `dist/`.

## Publish

```
pnpm -F @tcglanddev/api-sdk publish --access public
```

Ensure `.npmrc` maps the `@tcglanddev` scope to your registry.
