import { Dexie } from "dexie";

import {
	type BatchWrite,
	BatchWriteError,
	type EntityRecord,
	type MetadataSnapshot,
	type PersistenceProvider,
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
	stores[deriveActiveTableName(SCHEMAS_STORE)] = "tableName"; // TODO: Instead of having one table for active schemas, we can have an index on SCHEMAS_STORE. But no idea how to do it in Dexie.
	stores[JOURNAL_STORE] = "++changeId";

	// Entity tables (one store per versioned entity table plus an active index).
	for (const [tableName] of metadata.tables) {
		stores[tableName] = "entityId, entityVersion";
		stores[deriveActiveTableName(tableName)] = "entityId"; // TODO: Instead of having one table for active schemas, we can have an index on the entity table. But no idea how to do it in Dexie.
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

	async batchWrites(entities: BatchWrite): Promise<void> {
		if (entities.length === 0) {
			return;
		}

		const storeNames = new Set<string>();
		for (const entity of entities) {
			storeNames.add(entity.tableName);
			if (entity.isActive) {
				storeNames.add(deriveActiveTableName(entity.tableName));
			}
		}

		let lastRecord = 0;
		try {
			await this.dexie.transaction("rw", [...storeNames], async () => {
				for (const entity of entities) {
					lastRecord++;
					const table = this.dexie.table<EntityRecord>(entity.tableName);
					const activeTable = this.dexie.table(
						deriveActiveTableName(entity.tableName),
					);

					await table.put(entity);
					if (entity.isActive) {
						await activeTable.put(entity);
					}
				}
			});
		} catch (error) {
			if (error instanceof BatchWriteError) {
				throw error;
			}

			const last = entities[lastRecord];
			throw new BatchWriteError({
				tableName: last.tableName,
				entityId: last.entityId,
				reason: error instanceof Error ? error.message : String(error),
			});
		}
	}

	// getEntity<TPayload = unknown>(
	//     _ref: EntityIdentifier,
	// ): Promise<EntityRecord<TPayload>> {
	//     throw new Error("Method not implemented.");
	// }
	//
	// queryEntities<TPayload = unknown>(
	//     _scope: SchemaIdentifier,
	//     _pagination?: PaginationQuery,
	// ): Promise<PaginatedResult<EntityRecord<TPayload>>> {
	//     throw new Error("Method not implemented.");
	// }
	//
	// saveEntity<TPayload = unknown>(
	//     _input: SaveEntityInput<TPayload>,
	// ): Promise<EntityRecord<TPayload>> {
	//     throw new Error("Method not implemented.");
	// }
	//
	// deleteEntity(_input: DeleteEntityInput): Promise<void> {
	//     throw new Error("Method not implemented.");
	// }
	//
	// listJournalEntries(): Promise<JournalEntry[]> {
	// 	throw new Error("Method not implemented.");
	// }
	//
	// clearJournalEntries(): Promise<void> {
	// 	throw new Error("Method not implemented.");
	// }

	async close(): Promise<void> {
		this.dexie.close();
	}
}
