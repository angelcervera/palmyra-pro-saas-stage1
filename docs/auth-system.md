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
  "isAdmin": true,
  "otherClaim": "xxxxxxxx",
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
  }
}
```

Notes:

- `iss`/`aud` must match the Firebase project/tenant.
- `firebase.tenant` identifies the Identity Platform tenant. Our middleware treats this as the source of truth for multi-tenant routing.
- Custom claims such as `isAdmin` or `otherClaim` may be added when needed and are exposed through `UserCredentials`.

## Claim Extraction & Tenant Handling

- `platform/go/auth/auth.go#L95-L140` maps canonical claims (`uid`, `sub`, `email`, `email_verified`, etc.) into `UserCredentials`.
- `platform/go/auth/auth.go#L136-L150` ensures `TenantID` is populated from either top-level `tenantId` (if present) or `firebase.tenant` (Firebase default). This matches the token sample above, which only exposes the tenant inside the `firebase` map.
- During Firebase verification, `platform/go/auth/auth.go#L189-L207` copies the raw claim set and explicitly re-attaches the `firebase` map so downstream code sees the same payload as the client token.

## Local Testing

- When `AUTH_PROVIDER=dev`, unsigned tokens must mimic the Firebase structure (including `firebase.tenant`) so the extractor behaves identically. See `docs/auth-testing.md` for ready-to-use payloads that already include tenant data.

## Validation Status

- The current implementation reads `firebase.tenant` and exposes it via `UserCredentials.TenantID`, satisfying the tenant requirement from the provided JWT format.
- Middleware wiring in `apps/api/main.go#L80-L117` ensures every API call (except health checks and docs) passes through the JWT context, so tenant-aware authorization logic has the data it needs.
