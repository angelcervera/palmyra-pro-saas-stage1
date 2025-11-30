import type {
	BatchWrite,
	DeleteEntityInput,
	EntityIdentifier,
	EntityRecord,
	SaveEntityInput,
} from "./entities";
import type { JournalEntry } from "./journal";
import type { PaginatedResult, QueryOptions } from "./pagination";
import type { Schema, SchemaIdentifier } from "./schemas";

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
	getMetadata(): Promise<Schema[]>;

	/**
	 * Replaces the locally cached metadata snapshot (used by offline providers).
	 */
	setMetadata(snapshot: Schema[]): Promise<void>;

	/**
	 * Executes multiple save/delete operations in a single round-trip.
	 * It keeps the order, so asume that the latest active is the one actually active.
	 * By default, this operation must not write in the journal.
	 */
	batchWrites(operations: BatchWrite, writeInJournal: boolean): Promise<void>;

	/**
	 * Upserts an entity using the latest schema version for the table.
	 * If entityId is not provided, a new unique identifier will be generated.
	 */
	saveEntity<TPayload = unknown>(
		input: SaveEntityInput<TPayload>,
	): Promise<EntityRecord<TPayload>>;

	/**
	 * Retrieves the latest version of the specified entity.
	 * If the entity does not exist, returns undefined.
	 */
	getEntity<TPayload = unknown>(
		ref: EntityIdentifier,
	): Promise<EntityRecord<TPayload> | undefined>;

	/**
	 * Soft-deletes an entity (marks the latest version as deleted).
	 */
	deleteEntity(input: DeleteEntityInput): Promise<void>;

	/**
	 * Lists the latest versions of entities in a table.
	 * `QueryOptions` currently carries pagination, active/deleted flags, and will
	 * later include filter DSL expressions.
	 */
	queryEntities<TPayload = unknown>(
		tableName: SchemaIdentifier,
		options?: QueryOptions,
	): Promise<PaginatedResult<EntityRecord<TPayload>>>;

	/**
	 * Returns pending journal entries, if the provider supports a change journal.
	 * Providers that do not support journaling should return an empty array.
	 */
	listJournalEntries(): Promise<JournalEntry[]>;

	/**
 * Clears pending journal entries, if the provider supports a change journal.
 * Providers without journaling should treat this as a no-op.
 */
	clearJournalEntries(): Promise<void>;

	/**
	 * Wipes all entities for the given table/schema in this provider.
	 * Intended for sync/reset flows; providers that cannot support it should throw.
	 */
	clear(table: SchemaIdentifier): Promise<void>;

	/**
	 * Releases any underlying resources (DB handles, workers, etc.).
	 */
	close(): Promise<void>;
}
