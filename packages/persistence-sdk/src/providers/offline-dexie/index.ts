import { Dexie } from "dexie";

import {
	type BatchWrite,
	BatchWriteError,
	type EntityIdentifier,
	type EntityRecord,
	type PersistenceProvider,
	type SaveEntityInput,
	type Schema,
	type SchemaDefinition,
} from "../../core";
import { fromWireJson, toWireJson } from "../../shared/json";

const DB_VERSION = 1; // WARNING: Changing this value will result in a database migration and possible loss of data. Investigate behavior!
const SCHEMAS_STORE = "__schema-metadata";
const JOURNAL_STORE = "__entity-journal";

export type OfflineDexieProviderOptions = {
	tenantId: string;
	envKey: string;
	appName: string;
	schemas: Schema[];
};

export const createOfflineDexieProvider = async (
	options: OfflineDexieProviderOptions,
): Promise<OfflineDexieProvider> => OfflineDexieProvider.create(options);

// Helpers

const deriveDBName = (envKey: string, tenantId: string, appName: string) =>
	`${envKey}-${tenantId}-${appName}`;

const deriveActiveTableName = (tableName: string) => `active::${tableName}`;

const dexieStoresBuilder = (schemas: Schema[]) => {
	const stores: { [tableName: string]: string | null } = {};
	const entititesTableNames = [...schemas.values()].map(
		(schema) => schema.tableName,
	);

	// Required stores for schema metadata and journal entries.
	stores[SCHEMAS_STORE] = "tableName, schemaVersion";
	stores[deriveActiveTableName(SCHEMAS_STORE)] = "tableName"; // TODO: Instead of having one table for active schemas, we can have an index on SCHEMAS_STORE. But no idea how to do it in Dexie.
	stores[JOURNAL_STORE] = "++changeId";

	// Entity tables (one store per versioned entity table plus an active index).
	for (const tableName of entititesTableNames) {
		stores[tableName] = "entityId, entityVersion";
		stores[deriveActiveTableName(tableName)] = "entityId"; // TODO: Instead of having one table for active schemas, we can have an index on the entity table. But no idea how to do it in Dexie.
	}

	return stores;
};

// recoverMetadata attempts to recover the schema metadata from the latest database.
// If the database does not exist, it returns an empty metadata snapshot.
// If the database exists but the metadata table is empty, it returns an empty metadata snapshot.
// If the database exists and the metadata table is not empty, it returns the metadata snapshot from the database.
const recoverMetadata = (databaseName: string): Schema[] => {
	const db = new Dexie(databaseName);
	db.version(DB_VERSION).stores(dexieStoresBuilder([]));

	// AI TODO:
	//  Implement metadata recovery logic. Consider using Dexie's transaction and query capabilities to fetch the latest schema metadata from the database.
	// The name of the table is at SCHEMAS_STORE

	db.close();
	return [];
};

const initDexie = (options: OfflineDexieProviderOptions): Dexie => {
	const databaseName = deriveDBName(
		options.envKey,
		options.tenantId,
		options.appName,
	);

	// if `metadata.tables` is not empty, try to recover the schema from the latest database.

	const db = new Dexie(databaseName);
	db.version(DB_VERSION).stores(dexieStoresBuilder(options.metadata));

	return db;
};

// Implementation
export class OfflineDexieProvider implements PersistenceProvider {
	readonly name: string = "Offline Dexie";
	readonly description: string =
		"A persistence provider that stores data locally using Dexie";

	private constructor(
		private dexie: Dexie,
		readonly options: OfflineDexieProviderOptions,
	) {}

	static async create(
		options: OfflineDexieProviderOptions,
	): Promise<OfflineDexieProvider> {
		const provider = new OfflineDexieProvider(initDexie(options), options);
		await provider.dexie.open(); // Dexie does not create the database until open() is called.
		return provider;
	}

	getMetadata(): Promise<Schema[]> {
		throw new Error("Method not implemented.");
	}

	setMetadata(snapshot: Schema[]): Promise<void> {
		if (!this.dexie.hasBeenClosed()) {
			this.dexie.close();
		}

		this.options.metadata = snapshot;
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

	async getEntity<TPayload = unknown>(
		ref: EntityIdentifier,
	): Promise<EntityRecord<TPayload> | undefined> {
		const activeTableName = deriveActiveTableName(ref.tableName);
		return this.dexie.transaction<EntityRecord<TPayload> | undefined>(
			"r",
			[activeTableName],
			async () =>
				this.dexie
					.table<EntityRecord<TPayload>>(activeTableName)
					.get(ref.entityId),
		);
	}

	async saveEntity<TPayload = unknown>(
		input: SaveEntityInput<TPayload>,
	): Promise<EntityRecord<TPayload>> {
		const activeSchemasTableName = deriveActiveTableName(SCHEMAS_STORE);
		const activeEntityTableName = deriveActiveTableName(input.tableName);
		const tableNames = [
			activeEntityTableName,
			activeSchemasTableName,
			input.tableName,
		];

		// If it exists, we need the older one to updated it.
		let oldActiveEntityRecord: EntityRecord<TPayload> | undefined;
		if (input.entityId) {
			oldActiveEntityRecord = await this.getEntity<TPayload>({
				tableName: input.tableName,
				entityId: input.entityId,
			});
		}

		return this.dexie.transaction<EntityRecord<TPayload> | undefined>(
			"rw",
			tableNames,
			async () => {
				// Search metadata and validate the json schema.
				const schema = await this.dexie
					.table<SchemaDefinition<TPayload>>(activeSchemasTableName)
					.get(input.tableName);

				// TODO: Add json schema validation for the payload.

				const entityVersion = "1.0.0";

				// If it exists, we need the older one to updated it.
				if (oldActiveEntityRecord) {
					oldActiveEntityRecord.isActive = false;
					// AI TODO: Set oldActiveEntityRecord in dexie

					// AI TODO: set entityVersion as one more semantic version patch that oldActiveEntityRecord.entityVersion
				}

				const entityId =
					input.entityId ?? globalThis.crypto?.randomUUID?.() ?? undefined;

				if (!entityId) throw new Error("Entity ID generation failed");

				const entityRecord: EntityRecord<TPayload> = {
					entityId,
					tableName: input.tableName,
					payload: input.payload,
					entityVersion,
					isActive: true,
					ts: new Date(),
					isDeleted: false,
				};

				// AI TODO: Store entityRecord in activeEntityTableName and input.tableName

				return entityRecord;
			},
		);

		if (input.payload === undefined) {
		} else {
		}

		const metadata = this.options.metadata.tables.get(input.tableName);
		if (!metadata) {
			throw new Error(
				`Schema metadata missing for table ${input.tableName}. Did you seed metadata first?`,
			);
		}

		const entityId =
			input.entityId ??
			globalThis.crypto?.randomUUID?.() ??
			`ent_${Math.random().toString(36).slice(2, 11)}`;
		const entityVersion = `${Date.now()}-${Math.random()
			.toString(36)
			.slice(2, 8)}`;
		const ts = new Date();
		const wirePayload = toWireJson(input.payload);

		const storedRecord: EntityRecord = {
			tableName: input.tableName,
			entityId,
			entityVersion,
			schemaVersion: metadata.activeVersion,
			payload: wirePayload as unknown as TPayload,
			ts,
			isDeleted: false,
			isActive: true,
		};

		return this.dexie
			.transaction(
				"rw",
				[input.tableName, deriveActiveTableName(input.tableName)],
				async () => {
					const table = this.dexie.table<EntityRecord>(input.tableName);
					const activeTable = this.dexie.table<EntityRecord>(
						deriveActiveTableName(input.tableName),
					);

					await table.put({ ...storedRecord, payload: wirePayload });
					await activeTable.put({ ...storedRecord, payload: wirePayload });

					return {
						...storedRecord,
						payload: fromWireJson<TPayload>(wirePayload),
					};
				},
			)
			.catch((error) => {
				if (error instanceof Error) {
					throw new Error(
						`Failed to persist entity in ${input.tableName}: ${error.message}`,
					);
				}
				throw new Error(`Failed to persist entity in ${input.tableName}`);
			});
	}

	// queryEntities<TPayload = unknown>(
	//     _scope: SchemaIdentifier,
	//     _pagination?: PaginationQuery,
	// ): Promise<PaginatedResult<EntityRecord<TPayload>>> {
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
