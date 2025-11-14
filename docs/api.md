# API

The project uses **OpenAPI 3.0.4** or later to support embedded JSON Schemas.  
The API definition is modular, organized by domain.

## API Conventions

### Endpoint Naming

- Use plural nouns for resource collections: `/api/v1/users`, `/api/v1/schema-categories`
- Use kebab-case for multi-word resources: `/api/v1/user-profiles`
- Avoid verbs in endpoint paths; let HTTP methods convey the action
- Use nested resources sparingly and only when the relationship is strong: `/api/v1/schema-repository/schemas/{schemaId}/versions`
- **Action-style endpoints** (`:actionName` suffix) should be used only for operations that:
    - Don't fit standard CRUD semantics (e.g., `:approve`, `:reject`, `:sync`)
    - Represent RPC-style operations or commands (e.g., `:listPublic`, `:search`, `:validate`)
    - Perform complex business logic that can't be expressed through standard HTTP methods
    - Example: `POST /api/v1/users/{userId}:approve` or `GET /api/v1/locations:listPublic`
    - Use sparingly and document clearly in OpenAPI - prefer RESTful patterns when possible

### HTTP Methods

- `GET`: Retrieve resource(s) - must be idempotent and safe
- `POST`: Create a new resource - returns `201 Created` with `Location` header
- `PUT`: Full resource replacement - idempotent
- `PATCH`: Partial resource update - include only changed fields
- `DELETE`: Remove a resource - returns `204 No Content` on success

### HTTP Status Codes

For simplicity and the first version, we will have only two responses per path:
- Success response, with its own code depending on the type of request.
  - `200 OK`: Successful GET, PUT, or PATCH
  - `201 Created`: Successful POST - include `Location` header pointing to new resource
  - `204 No Content`: Successful DELETE or PUT/PATCH with no response body
- Error response, market as `default` and type `application/problem+json`.

In the future, we may add more status codes to support more complex scenarios.
- `400 Bad Request`: Client validation error - return ProblemDetails with field-level errors
- `401 Unauthorized`: Missing or invalid authentication
- `403 Forbidden`: Authenticated but lacks required permissions
- `404 Not Found`: Resource does not exist
- `409 Conflict`: Resource conflict (e.g., duplicate code/slug)
- `422 Unprocessable Entity`: Semantic validation error
- `500 Internal Server Error`: Unexpected server error - log details, return generic ProblemDetails

### Request/Response Patterns

- Due to [oapi-codegen issue 2113](https://github.com/oapi-codegen/oapi-codegen/issues/2113), default responses should inline the ProblemDetails schema instead of referencing a shared component response. This ensures code generation produces the expected handler signatures.
- All endpoints must return consistent response envelopes defined in `/contracts/common/`
- Use **ProblemDetails** (RFC 7807) for all error responses with:
    - `type`: URI identifying the error type
    - `title`: Human-readable error summary
    - `status`: HTTP status code
    - `detail`: Specific error explanation
    - `instance`: Request identifier for tracing
    - `errors`: Optional map of field-level validation errors
- Collection endpoints must use the standardized **Pagination** model with:
    - `items`: Array of resources
    - `page`: Current page number (1-indexed)
    - `pageSize`: Items per page
    - `totalItems`: Total resource count
    - `totalPages`: Calculated total pages
- Single resource responses return the resource object directly (no envelope)

### Filtering, Sorting & Pagination

- Use query parameters for filters: `?name=value&status=enabled`
- Use `sort` parameter with field names: `?sort=name` (ascending) or `?sort=-createdAt` (descending)
- Use `page` and `pageSize` for pagination: `?page=1&pageSize=20`
- Default `pageSize` should be `20`, with max `100`
- Always include pagination metadata in collection responses

### Versioning

- API version prefix: `/api/v1/`
- Version is fixed in URL path, not via headers or query parameters
- Breaking changes require a new version number

### Field Naming

- Use `camelCase` for all JSON field names
- Use ISO 8601 format for dates and timestamps: `2025-10-13T14:30:00Z`
- Use UUIDs for resource identifiers: `id` field as string

### Common Fields

- All resources should include (where applicable):
    - `id`: UUID identifier
    - `createdAt`: ISO 8601 timestamp
    - `updatedAt`: ISO 8601 timestamp (equals `createdAt` for new resources)
    - `deletedAt`: ISO 8601 timestamp (for soft deletes, if applicable)

### Content Negotiation

- Accept: `application/json` (required)
- Content-Type: `application/json` for request/response bodies
- Charset: UTF-8 (default, non-negotiable)

### Security

- All endpoints except `/auth/signup` and `/auth/login` require JWT authentication
- Include JWT in `Authorization: Bearer <token>` header
- Token expiration and refresh handled via separate auth endpoints
- Rate limiting headers should be included where applicable:
    - `X-RateLimit-Limit`
    - `X-RateLimit-Remaining`
    - `X-RateLimit-Reset`
