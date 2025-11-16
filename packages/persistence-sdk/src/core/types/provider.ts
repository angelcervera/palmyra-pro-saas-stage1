import type {
	BatchWrite,
	DeleteEntityInput,
	EntityIdentifier,
	EntityRecord,
	SaveEntityInput,
} from "./entities";
import type { PaginatedResult, PaginationQuery } from "./pagination";
import type { MetadataSnapshot, SchemaIdentifier } from "./schemas";

/**
 * Defines the contract between the SDK client and any persistence provider.
 * Providers always operate on the latest schema version and latest entity state.
 */
export interface PersistenceProvider {
	/**
	 * Human-friendly identifier (e.g., "online" or "offline" or "offline-psql" or "online-firestore").
	 */
	readonly name: string;

	readonly description: string;

	/**
	 * Returns the list of tenant tables/schemas and their available versions.
	 */
	getMetadata(): Promise<MetadataSnapshot>;

	/**
	 * Retrieves the latest version of the specified entity.
	 */
	getEntity<TPayload = unknown>(
		ref: EntityIdentifier,
	): Promise<EntityRecord<TPayload>>;

	/**
	 * Lists the latest versions of entities in a table
	 */
	queryEntities<TPayload = unknown>(
		// TODO: Add filtering support
		scope: SchemaIdentifier,
		pagination?: PaginationQuery,
	): Promise<PaginatedResult<EntityRecord<TPayload>>>;

	/**
	 * Upserts an entity using the latest schema version for the table.
	 */
	saveEntity<TPayload = unknown>(
		input: SaveEntityInput<TPayload>,
	): Promise<EntityRecord<TPayload>>;

	/**
	 * Soft-deletes an entity (marks the latest version as deleted).
	 */
	deleteEntity(input: DeleteEntityInput): Promise<void>;

	/**
	 * Executes multiple save/delete operations in a single round-trip.
	 */
	batchWrites(operations: BatchWrite[]): Promise<void>;
}
