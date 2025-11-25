---
id: auth-system
version: 1.1.0
lastUpdated: 2025-11-13
appliesTo:
  - apps/api
  - domains/*/be
  - platforms/go/auth
relatedDocs:
  - docs/api-server.md
  - docs/project-requirements-document.md
---

# Authentication Architecture

## Overview

Palmyra relies on Firebase Authentication (or Identity Platform) as the single issuer for JWT bearer tokens. The API server exposes two auth modes controlled by `AUTH_PROVIDER` (see `apps/api/main.go`):

- `firebase` (default) — Verifies signed Firebase ID tokens via the Admin SDK and enforces all claims.
- `dev` — Accepts unsigned JWT payloads for local testing while preserving the same claim structure.

Every incoming request passes through `platform/go/auth/auth.JWT`, which validates the token, extracts standardized `UserCredentials`, and stores them in the request context for downstream handlers.

## Middleware Flow

1. `platform/go/auth/jwt.go#L13-L52` verifies the bearer token using either Firebase or the unsigned verifier, depending on configuration.
2. `platform/go/auth/auth.go#L47-L86` injects the resulting `UserCredentials` into the context.
3. Domain handlers call `platform/go/auth.UserFromContext` to retrieve identity data, including tenant membership and admin role, before performing authorization checks.

## JWT Format (Firebase-compliant)

Tokens issued by Firebase must align with this structure, including tenant scoping in the nested `firebase` claim:

```json
{
  "name": "Dev Testing",
  "iss": "https://securetoken.google.com/cacao---dev",
  "aud": "cacao---dev",
  "auth_time": 1763038864,
  "user_id": "1F2w2N3TBKMX0eM2fkkkMwX6CDL2",
  "sub": "1F2w2N3TBKMX0eM2fkkkMwX6CDL2",
  "iat": 1763038864,
  "exp": 1763042464,
  "email": "admin@choconiger.global",
  "email_verified": false,
  "firebase": {
    "identities": {
      "email": [
        "admin@choconiger.global"
      ]
    },
    "sign_in_provider": "password",
    "tenant": "ChocoNiger-o12xx"
  },
  "palmyraRoles": ["admin"],
  "tenantRoles": ["admin"]
}
```

Notes:

- `iss`/`aud` must match the Firebase project/tenant.
- `firebase.tenant` identifies the Identity Platform tenant. Our middleware treats this as the source of truth for multi-tenant routing and uses it as the input to resolve the active Tenant Space (see `docs/multitenancy/overview.md`).
- Custom claims such as `isAdmin` or `otherClaim` may be added when needed and are exposed through `UserCredentials`.

## Claim Extraction & Tenant Handling

- `platform/go/auth/auth.go#L95-L140` maps canonical claims (`uid`, `sub`, `email`, `email_verified`, etc.) into `UserCredentials`.
- `platform/go/auth/auth.go#L136-L150` derives `TenantID` exclusively from `firebase.tenant`, matching the token sample above.
- During Firebase verification, `platform/go/auth/auth.go#L189-L207` copies the raw claim set and explicitly re-attaches the `firebase` map so downstream code sees the same payload as the client token.

## Validation Status

- The current implementation reads `firebase.tenant` and exposes it via `UserCredentials.TenantID`, satisfying the tenant requirement from the provided JWT format.
- Middleware wiring in `apps/api/main.go#L80-L117` ensures every API call (except health checks and docs) passes through the JWT context, so tenant-aware authorization logic has the data it needs.

---

## Authentication Testing Playbook

This guide explains how to exercise the API when `AUTH_PROVIDER` is set to `firebase` (real verification) or `dev` (unsigned JWT decoding). Use it whenever you need to test role-based behaviour or debug auth flows.

### 1. Switch providers

Environment variables (see `.env.dockercompose` for local defaults):

- `AUTH_PROVIDER=firebase` — uses Firebase/Identity Platform. Requires valid credentials via `FIREBASE_CONFIG` or ADC.
- `AUTH_PROVIDER=dev` — uses the unsigned verifier. **Local/CI only.**

Restart the API after changing the value (`docker compose up --build` or `go run ./apps/api`).

### 2. Dev provider (unsigned JWTs)

When `AUTH_PROVIDER=dev`, the middleware calls:

```go
platformauth.JWT(platformauth.UnsignedTokenVerifier(), nil)
```

The verifier skips signature checks and simply decodes the JWT payload, copying standard claims (`email`, `name`, `isAdmin`, etc.). Missing/invalid tokens return `401`.

#### 2.1 Sample tokens

| Role  | Bearer token                                                                                                                                                                                                     |
|-------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Admin | `eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJpc3MiOiJodHRwczovL3NlY3VyZXRva2VuLmdvb2dsZS5jb20vbG9jYWwtcGFsbXlyYSIsImF1ZCI6ImxvY2FsLXBhbG15cmEiLCJhdXRoX3RpbWUiOjE3NjMwNDE0OTksInVzZXJfaWQiOiJhZG1pbi0xMjMiLCJzdWIiOiJhZG1pbi0xMjMiLCJpYXQiOjE3NjMwNDE0OTksImV4cCI6MTc2MzA0NTA5OSwiZW1haWwiOiJhZG1pbkBleGFtcGxlLmNvbSIsImVtYWlsX3ZlcmlmaWVkIjp0cnVlLCJuYW1lIjoiRGV2IEFkbWluIiwiaXNBZG1pbiI6dHJ1ZSwiZmlyZWJhc2UiOnsiaWRlbnRpdGllcyI6eyJlbWFpbCI6WyJhZG1pbkBleGFtcGxlLmNvbSJdfSwic2lnbl9pbl9wcm92aWRlciI6InBhc3N3b3JkIiwidGVuYW50IjoidGVuYW50LWRldiJ9fQ` |
| User  | `eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJpc3MiOiJodHRwczovL3NlY3VyZXRva2VuLmdvb2dsZS5jb20vbG9jYWwtcGFsbXlyYSIsImF1ZCI6ImxvY2FsLXBhbG15cmEiLCJhdXRoX3RpbWUiOjE3NjMwNDE1ODcsInVzZXJfaWQiOiJhZG1pbi0xMjMiLCJzdWIiOiJhZG1pbi0xMjMiLCJpYXQiOjE3NjMwNDE1ODcsImV4cCI6MTc2MzA0NTE4NywiZW1haWwiOiJhZG1pbkBleGFtcGxlLmNvbSIsImVtYWlsX3ZlcmlmaWVkIjp0cnVlLCJuYW1lIjoiRGV2ZWxvcGVyIFVzZXIiLCJpc0FkbWluIjpmYWxzZSwiZmlyZWJhc2UiOnsiaWRlbnRpdGllcyI6eyJlbWFpbCI6WyJhZG1pbkBleGFtcGxlLmNvbSJdfSwic2lnbl9pbl9wcm92aWRlciI6InBhc3N3b3JkIiwidGVuYW50IjoidGVuYW50LWRldiJ9fQ`    |

Generate your own with Node or any JWT tool, making sure to include the same claims Firebase issues (issuer/audience pair and `firebase.tenant`):

```bash
node - <<'JS'
function b64(obj){return Buffer.from(JSON.stringify(obj)).toString('base64url');}
const now=Math.floor(Date.now()/1000);
const header={alg:'none',typ:'JWT'};
const payload={
  iss:'https://securetoken.google.com/local-palmyra',
  aud:'local-palmyra',
  auth_time:now,
  user_id:'admin-123',
  sub:'admin-123',
  iat:now,
  exp:now+3600,
  email:'admin@example.com',
  email_verified:true,
  name:'Dev Admin',
  isAdmin:true,
  firebase:{
    identities:{email:['admin@example.com']},
    sign_in_provider:'password',
    tenant:'tenant-dev'
  }
};
console.log(`${b64(header)}.${b64(payload)}`);
JS
```

#### 2.2 Testing via curl

```bash
curl -H "Authorization: Bearer <token>" http://localhost:3000/api/v1/schema-categories
```

Swap `<token>` for the admin/user payload to confirm role checks (`RequireRole("admin")`) behave as expected.

#### 2.3 Setting the admin web app token via DevTools

When running `pnpm dev -C apps/web-platform-admin`, point the frontend at Docker’s API by exporting:

```bash
VITE_API_BASE_URL=http://localhost:3000/api/v1 pnpm dev -C apps/web-platform-admin
```

To simulate a signed-in session inside the browser:

1. Open the app (http://localhost:5173 by default) and launch DevTools → Console.
2. Paste the token you want to use (see Sample tokens above) and store it in `sessionStorage` under the key `jwt`:

   ```js
   const token = 'eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0...';
   sessionStorage.setItem('jwt', token);
   ```

3. Reload the page. The `AuthProvider` reads the token from `sessionStorage` and all subsequent requests from the SDK client will include the `Authorization` header automatically.
4. To swap identities, call `sessionStorage.setItem('jwt', '<new-token>')` and refresh, or clear the session with `sessionStorage.removeItem('jwt')`.

This approach affects every request originating from the SPA, so you can navigate through the UI and hit the Docker-backed API without manually attaching headers.

#### 2.4 Safety checklist

- Never deploy containers with `AUTH_PROVIDER=dev`.
- Monitor logs for `auth.provider != "firebase"` outside local/CI environments.
- Treat unsigned tokens as secrets; do not embed them in commits.

### 3. Firebase/Identity Platform provider

When `AUTH_PROVIDER=firebase`, middleware wiring becomes:

```go
platformauth.JWT(platformauth.FirebaseTokenVerifier(fbAuth), nil)
```

Steps:

1. Provide credentials: `FIREBASE_CONFIG=/path/to/service-account.json` (or use Application Default Credentials via `gcloud auth application-default login`).
2. Restart the API.
3. Obtain a real ID token (sign in through the frontend, Firebase Admin SDK, or REST API).
4. Call the API:

   ```bash
   curl -H "Authorization: Bearer ${ID_TOKEN}" http://localhost:3000/api/v1/schema-categories
   ```

Tenants/claims from Identity Platform appear in the same struct (`UserCredentials`). Use this mode for integration tests and pre-production validation.

### 4. Troubleshooting

| Symptom                                                     | Likely cause                               | Fix                                                               |
|-------------------------------------------------------------|--------------------------------------------|-------------------------------------------------------------------|
| `init firebase auth ... could not find default credentials` | Missing/incorrect `FIREBASE_CONFIG` or ADC | Export the env var or run `gcloud auth application-default login` |
| `401 unauthorized` in dev mode                              | Token missing or malformed                 | Ensure `Authorization: Bearer <unsigned-jwt>` header is set       |
| `403 forbidden`                                             | Role middleware blocked the request        | Confirm token has `isAdmin: true` (or expected claims)            |

### 5. Going further
- Add roles to operations in OpenAPI using a vendor extension: set `x-required-roles` to a list of roles defined in `contracts/common/iam.yaml#/components/schemas/UserRole`. Example:

  ```yaml
  paths:
    /admin/users:
      get:
        security:
          - bearerAuth: []
        x-required-roles: [admin, user_manager]
  ```

- Define the available roles once in `contracts/common/iam.yaml` and reference that enum across the specs. Current roles: `admin`, `user_manager`, `user`.

- Script token generation as part of fixture setup (e.g., create `scripts/dev-token.js`).
- Add automated smoke tests that hit `/api/v1/…` with both admin and non-admin tokens to ensure role protections stay intact.
- In CI, force `AUTH_PROVIDER=firebase` for end-to-end tests unless explicitly running an unsigned smoke suite.
