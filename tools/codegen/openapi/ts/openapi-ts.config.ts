// OpenAPI → TypeScript codegen configuration (preferred approach)
// Run with: pnpm dlx @hey-api/openapi-ts@0.86.11 -c tools/codegen/openapi/ts/openapi-ts.config.ts

// Tip: Keep outputs under packages/api-sdk/src/generated/<domain> (read-only). SDK package consumes these.

export default [
  {
    input: './contracts/auth.yaml',
    output: './packages/api-sdk/src/generated/auth',
    client: 'fetch',
    base: '/api/v1',
    types: true,
    services: true,
    schemas: true,
  },
  {
    input: './contracts/users.yaml',
    output: './packages/api-sdk/src/generated/users',
    client: 'fetch',
    base: '/api/v1',
    types: true,
    services: true,
    schemas: true,
  },
  {
    input: './contracts/schema-categories.yaml',
    output: './packages/api-sdk/src/generated/schema-categories',
    client: 'fetch',
    base: '/api/v1',
    types: true,
    services: true,
    schemas: true,
  },
  {
    input: './contracts/schema-repository.yaml',
    output: './packages/api-sdk/src/generated/schema-repository',
    client: 'fetch',
    base: '/api/v1',
    types: true,
    services: true,
    schemas: true,
  },
  {
    input: './contracts/entities.yaml',
    output: './packages/api-sdk/src/generated/entities',
    client: 'fetch',
    base: '/api/v1',
    types: true,
    services: true,
    schemas: true,
  },
];
