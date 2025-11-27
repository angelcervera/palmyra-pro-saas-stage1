import { Dexie } from "dexie";

import type {
	BatchWrite,
	DeleteEntityInput,
	EntityIdentifier,
	EntityRecord,
	JournalEntry,
	MetadataSnapshot,
	PaginatedResult,
	PaginationQuery,
	PersistenceProvider,
	SaveEntityInput,
	SchemaIdentifier,
} from "../../core";

const DB_VERSION = 1;
const SCHEMAS_STORE = "__schema-metadata";
const JOURNAL_STORE = "__entity-journal";

export type OfflineDixieProviderOptions = {
	tenantId: string;
	envKey: string;
	appName: string;
	initialMetadata: MetadataSnapshot;
};

export const createOfflineDixieProvider = async (
	options: OfflineDixieProviderOptions,
): Promise<OfflineDixieProvider> => OfflineDixieProvider.create(options);

// Helpers

const deriveDBName = (envKey: string, tenantId: string, appName: string) =>
	`${envKey}-${tenantId}-${appName}`;

const deriveActiveTableName = (tableName: string) => `active::${tableName}`;

const dixieStoresBuilder = (metadata: MetadataSnapshot) => {
	const stores: { [tableName: string]: string | null } = {};

	// Required stores for schema metadata and journal entries.
	stores[SCHEMAS_STORE] = "tableName, schemaVersion";
	stores[deriveActiveTableName(SCHEMAS_STORE)] = "tableName";
	stores[JOURNAL_STORE] = "++changeId";

	// Entity tables (one store per versioned entity table plus an active index).
	for (const [tableName] of metadata.tables) {
		stores[tableName] = "entityId, entityVersion";
		stores[deriveActiveTableName(tableName)] = "entityId";
	}

	return stores;
};

const initDexie = (options: OfflineDixieProviderOptions): Dexie => {
	const databaseName = deriveDBName(
		options.envKey,
		options.tenantId,
		options.appName,
	);
	const db = new Dexie(databaseName);
	db.version(DB_VERSION).stores(dixieStoresBuilder(options.initialMetadata));
	return db;
};

// Implementation
export class OfflineDixieProvider implements PersistenceProvider {
	readonly name: string = "Offline Dixie";
	readonly description: string =
		"A persistence provider that stores data locally using Dixie";

	private constructor(
		private dexie: Dexie,
		readonly options: OfflineDixieProviderOptions,
	) {}

	static async create(
		options: OfflineDixieProviderOptions,
	): Promise<OfflineDixieProvider> {
		const provider = new OfflineDixieProvider(initDexie(options), options);
		await provider.dexie.open(); // Dexie does not create the database until open() is called.
		return provider;
	}

	getMetadata(): Promise<MetadataSnapshot> {
		throw new Error("Method not implemented.");
	}

	setMetadata(snapshot: MetadataSnapshot): Promise<void> {
		if (!this.dexie.hasBeenClosed()) {
			this.dexie.close();
		}

		this.options.initialMetadata = snapshot;
		this.dexie = initDexie(this.options);

		return Promise.resolve();
	}

	getEntity<TPayload = unknown>(
		_ref: EntityIdentifier,
	): Promise<EntityRecord<TPayload>> {
		throw new Error("Method not implemented.");
	}

	queryEntities<TPayload = unknown>(
		_scope: SchemaIdentifier,
		_pagination?: PaginationQuery,
	): Promise<PaginatedResult<EntityRecord<TPayload>>> {
		throw new Error("Method not implemented.");
	}

	saveEntity<TPayload = unknown>(
		_input: SaveEntityInput<TPayload>,
	): Promise<EntityRecord<TPayload>> {
		throw new Error("Method not implemented.");
	}

	deleteEntity(_input: DeleteEntityInput): Promise<void> {
		throw new Error("Method not implemented.");
	}

	batchWrites(_operations: BatchWrite[]): Promise<void> {
		throw new Error("Method not implemented.");
	}

	listJournalEntries(): Promise<JournalEntry[]> {
		throw new Error("Method not implemented.");
	}

	clearJournalEntries(): Promise<void> {
		throw new Error("Method not implemented.");
	}

	async close(): Promise<void> {
		this.dexie.close();
	}
}
