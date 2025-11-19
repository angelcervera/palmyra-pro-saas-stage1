export type { ApiError, FetchClientOptions, ProblemDetails } from "./http";
export { createFetchClient } from "./http";

// NOTE: When adding a new domain, update the exports below so the SDK surface stays in sync.

// Auth domain
export * as Auth from "./generated/auth";
export type {
	Client as AuthClient,
	Config as AuthClientConfig,
} from "./generated/auth/client";
export { createClient as createAuthClient } from "./generated/auth/client";
export { client as authClient } from "./generated/auth/client.gen";
// Entities domain
export * as Entities from "./generated/entities";
export type {
	Client as EntitiesClient,
	Config as EntitiesClientConfig,
} from "./generated/entities/client";
export { createClient as createEntitiesClient } from "./generated/entities/client";
export { client as entitiesClient } from "./generated/entities/client.gen";

// Schema Categories domain
export * as SchemaCategories from "./generated/schema-categories";
export type {
	Client as SchemaCategoriesClient,
	Config as SchemaCategoriesClientConfig,
} from "./generated/schema-categories/client";
export { createClient as createSchemaCategoriesClient } from "./generated/schema-categories/client";
export { client as schemaCategoriesClient } from "./generated/schema-categories/client.gen";

// Schema Repository domain
export * as SchemaRepository from "./generated/schema-repository";
export type {
	Client as SchemaRepositoryClient,
	Config as SchemaRepositoryClientConfig,
} from "./generated/schema-repository/client";
export { createClient as createSchemaRepositoryClient } from "./generated/schema-repository/client";
export { client as schemaRepositoryClient } from "./generated/schema-repository/client.gen";
// Users domain
export * as Users from "./generated/users";
export type {
	Client as UsersClient,
	Config as UsersClientConfig,
} from "./generated/users/client";
export { createClient as createUsersClient } from "./generated/users/client";
export { client as usersClient } from "./generated/users/client.gen";
