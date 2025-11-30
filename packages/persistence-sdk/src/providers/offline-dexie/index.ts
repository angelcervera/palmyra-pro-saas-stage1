import { Dexie } from "dexie";

import {
	type BatchWrite,
	BatchWriteError,
	type DeleteEntityInput,
	type EntityIdentifier,
	type EntityRecord,
	type JournalEntry,
	type PaginatedResult,
	type PersistenceProvider,
	type QueryOptions,
	type SaveEntityInput,
	type Schema,
	type SchemaIdentifier,
} from "../../core";

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
): Promise<OfflineDexieProvider> => await OfflineDexieProvider.create(options);

// Helpers

const deriveDBName = (envKey: string, tenantId: string, appName: string) =>
	`${envKey}-${tenantId}-${appName}`;

const deriveActiveTableName = (tableName: string) => `active::${tableName}`;

const dexieStoresBuilder = (schemas: Schema[]) => {
	const stores: { [tableName: string]: string | null } = {};
	const entitiesTableNames = [...schemas.values()].map(
		(schema) => schema.tableName,
	);

	// Required stores for schema metadata and journal entries.
	stores[SCHEMAS_STORE] = "tableName, schemaVersion, createdAt";
	stores[deriveActiveTableName(SCHEMAS_STORE)] = "tableName, createdAt"; // TODO: Instead of having one table for active schemas, we can have an index on SCHEMAS_STORE. But no idea how to do it in Dexie.
	stores[JOURNAL_STORE] = "++changeId, createdAt";

	// Entity tables (one store per versioned entity table plus an active index).
	for (const tableName of entitiesTableNames) {
		stores[tableName] = "entityId, entityVersion, createdAt";
		stores[deriveActiveTableName(tableName)] = "entityId, createdAt"; // TODO: Instead of having one table for active schemas, we can have an index on the entity table. But no idea how to do it in Dexie.
	}

	return stores;
};

// Attempts to recover schema metadata and the current DB version without altering the schema.
// If the database does not exist, it returns empty schemas with version 0.
// If the database exists, it opens the latest version as-is and reads the metadata table.
const recoverSchemas = async (
	databaseName: string,
): Promise<{ schemas: Schema[]; currentVersion: number }> => {
	if (!(await Dexie.exists(databaseName))) {
		return { schemas: [], currentVersion: 0 };
	}

	// Open, read and close.
	const db = new Dexie(databaseName);
	try {
		await db.open();
		const currentVersion = db.verno;
		const schemas = await db.table<Schema>(SCHEMAS_STORE).toArray();
		return { schemas, currentVersion };
	} finally {
		db.close();
	}
};

const areSchemasCompatible = (a: Schema[], b: Schema[]): boolean =>
	new Set(a).size === new Set(b).size &&
	new Set([...a, ...b]).size === new Set(a).size;

// Initialize a Dexie instance using either provided schemas or ones recovered from disk.
const initDexie = async (
	options: OfflineDexieProviderOptions,
): Promise<Dexie> => {
	const databaseName = deriveDBName(
		options.envKey,
		options.tenantId,
		options.appName,
	);

	// Recover the latest version of the database, so: It will work offline and upgrade to new schemas.
	const latestVersion = await recoverSchemas(databaseName);

	// If schemas provided are not compatible with the latest version or there is no previous schema at all, we need to bump the DB version.
	const schemas =
		options.schemas.length === 0 ? latestVersion.schemas : options.schemas;
	let version = latestVersion.currentVersion;
	if (version === 0 || !areSchemasCompatible(schemas, latestVersion.schemas)) {
		version += 1;
	}

	// Finally, create the Dexie instance.
	const db = new Dexie(databaseName);
	db.version(version).stores(dexieStoresBuilder(schemas));
	await db.open(); // Dexie does not create the database until open() is called.

	// Ensure schema metadata stores contain the current schemas.
	if (options.schemas.length > 0) {
		const activeSchemasTableName = deriveActiveTableName(SCHEMAS_STORE);
		await db.transaction(
			"rw",
			[SCHEMAS_STORE, activeSchemasTableName],
			async () => {
				const schemasTable = db.table<Schema>(SCHEMAS_STORE);
				const activeTable = db.table<Schema>(activeSchemasTableName);
				await schemasTable.clear();
				await activeTable.clear();

				await schemasTable.bulkPut(schemas);

				const activeSchemas = schemas.filter((schema) => schema.isActive);
				if (activeSchemas.length > 0) {
					await activeTable.bulkPut(activeSchemas);
				}
			},
		);
	}

	return db;
};

const updateEntityVersion = (
	currentVersion: string,
	_oldSchemaVersion: string,
	_newSchemaVersion: string,
): string => {
	// TODO: If the schema version is different, we need to check if both versions are compatible and bump the entity version accordingly.

	const match = /^(\d+)\.(\d+)\.(\d+)$/.exec(currentVersion.trim());
	if (!match) throw new Error(`Invalid semver: ${currentVersion}`);

	const [major, minor, patch] = match.slice(1).map(Number);
	return `${major}.${minor}.${patch + 1}`;
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

	queryEntities<TPayload = unknown>(
		tableName: SchemaIdentifier,
		options?: QueryOptions,
	): Promise<PaginatedResult<EntityRecord<TPayload>>> {
		const page =
			options?.pagination?.page && options.pagination.page > 0
				? options.pagination.page
				: 1;
		const pageSize =
			options?.pagination?.pageSize && options.pagination.pageSize > 0
				? options.pagination.pageSize
				: 20;

		const useActive = options?.onlyActive !== false;
		const includeDeleted = options?.includeDeleted === true;
		const targetTableName = useActive
			? deriveActiveTableName(tableName.tableName)
			: tableName.tableName;

		return this.dexie.transaction<PaginatedResult<EntityRecord<TPayload>>>(
			"r",
			[targetTableName],
			async () => {
				const table = this.dexie.table<EntityRecord<TPayload>>(targetTableName);
				let collection = table.orderBy("createdAt").reverse();
				// NOTE: We keep using `filter` instead of `where` so we can preserve the
				// existing orderBy and avoid adding an index on isDeleted. Revisit if we
				// introduce an index and want a fully indexed cursor in the future.
				if (!includeDeleted) {
					collection = collection.filter((record) => !record.isDeleted);
				}

				const totalItems = await collection.count();
				const totalPages =
					totalItems === 0 ? 0 : Math.ceil(totalItems / pageSize);
				const offset = (page - 1) * pageSize;
				const items = await collection.offset(offset).limit(pageSize).toArray();

				return { items, page, pageSize, totalItems, totalPages };
			},
		);
	}

	static async create(
		options: OfflineDexieProviderOptions,
	): Promise<OfflineDexieProvider> {
		// Check platform.
		if (!globalThis.crypto?.randomUUID) {
			throw new Error("Random UUID generation is required for entity IDs.");
		}

		// Create the Dexie instance.
		const dixie = await initDexie(options);
		return new OfflineDexieProvider(dixie, options);
	}

	async getMetadata(): Promise<Schema[]> {
		return this.dexie.table<Schema>(SCHEMAS_STORE).toArray();
	}

	async setMetadata(snapshot: Schema[]): Promise<void> {
		if (!this.dexie.hasBeenClosed()) {
			this.dexie.close();
		}

		this.options.schemas = snapshot;
		this.dexie = await initDexie(this.options);

		return Promise.resolve();
	}

	async batchWrites(
		entities: BatchWrite,
		writeInJournal: boolean = true,
	): Promise<void> {
		if (entities.length === 0) {
			return;
		}

		const storeNames = new Set<string>([JOURNAL_STORE]);
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

					if (writeInJournal) await this.appendJournal(entity);
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

	// Delete an entity.
	// It is a soft deletion, so we will create a version with isDeleted on.
	async deleteEntity(input: DeleteEntityInput): Promise<void> {
		const activeEntityTableName = deriveActiveTableName(input.tableName);
		const tableNames = [activeEntityTableName, input.tableName, JOURNAL_STORE];

		const existing = await this.getEntity({
			tableName: input.tableName,
			entityId: input.entityId,
		});

		if (!existing) {
			throw new Error(
				`Entity ${input.entityId} not found in ${input.tableName}`,
			);
		}

		await this.dexie.transaction("rw", tableNames, async () => {
			const entityTable = this.dexie.table<EntityRecord>(input.tableName);
			const activeTable = this.dexie.table<EntityRecord>(activeEntityTableName);

			// Mark the previous active record as inactive and drop it from the active table.
			existing.isActive = false;
			await entityTable.put(existing);
			await activeTable.delete(existing.entityId);

			// Store a new version flagged as deleted.
			const deletedRecord: EntityRecord = {
				...existing,
				entityVersion: updateEntityVersion(
					existing.entityVersion,
					existing.schemaVersion,
					existing.schemaVersion,
				),
				schemaVersion: existing.schemaVersion,
				createdAt: new Date(),
				isDeleted: true,
				isActive: true,
			};

			await entityTable.put(deletedRecord);
			await activeTable.put(deletedRecord);
			await this.appendJournal(deletedRecord);
		});
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
			JOURNAL_STORE,
		];

		// If it exists, we need the older one to updated it.
		let oldActiveEntityRecord: EntityRecord<TPayload> | undefined;
		if (input.entityId) {
			oldActiveEntityRecord = await this.getEntity<TPayload>({
				tableName: input.tableName,
				entityId: input.entityId,
			});
		}

		return this.dexie.transaction<EntityRecord<TPayload>>(
			"rw",
			tableNames,
			async () => {
				// Search metadata and validate the json schema.
				const schema = await this.dexie
					.table<Schema>(activeSchemasTableName)
					.get(input.tableName);

				if (!schema) {
					throw new Error(`Schema not found for table: ${input.tableName}`);
				}

				// TODO: Add json schema validation for the payload.

				let entityVersion = "1.0.0";

				// If it exists, we need the older one to update it.
				if (oldActiveEntityRecord) {
					oldActiveEntityRecord.isActive = false;

					// Persist the previous active record as inactive and drop it from the active table.
					const entityTable = this.dexie.table<EntityRecord<TPayload>>(
						input.tableName,
					);
					const activeTable = this.dexie.table<EntityRecord<TPayload>>(
						activeEntityTableName,
					);
					await entityTable.put(oldActiveEntityRecord);
					await activeTable.delete(oldActiveEntityRecord.entityId);

					// Because it is an update, we need to bump the entity version.
					entityVersion = updateEntityVersion(
						oldActiveEntityRecord.entityVersion,
						oldActiveEntityRecord.schemaVersion,
						schema.schemaVersion,
					);
				}

				// Store a new version of the entity.
				const entityId = input.entityId ?? globalThis.crypto.randomUUID();
				const entityRecord: EntityRecord<TPayload> = {
					tableName: input.tableName,
					schemaVersion: schema.schemaVersion,
					entityId,
					entityVersion,
					payload: input.payload,
					isActive: true,
					createdAt: new Date(),
					isDeleted: false,
				};

				const entityTable = this.dexie.table<EntityRecord<TPayload>>(
					input.tableName,
				);
				const activeTable = this.dexie.table<EntityRecord<TPayload>>(
					activeEntityTableName,
				);
				await entityTable.put(entityRecord);
				await activeTable.put(entityRecord);

				await this.appendJournal(entityRecord);

				return entityRecord;
			},
		);
	}

	async listJournalEntries(): Promise<JournalEntry[]> {
		return this.dexie.table<JournalEntry>(JOURNAL_STORE).toArray();
	}

	async clearJournalEntries(): Promise<void> {
		return this.dexie.table(JOURNAL_STORE).clear();
	}

	async close(): Promise<void> {
		return this.dexie.close();
	}

	private async appendJournal(entity: EntityRecord): Promise<void> {
		type JournalRow = Omit<JournalEntry, "changeId"> & { changeId?: number };

		const journalEntry: JournalRow = {
			...entity,
		};
		await this.dexie.table<JournalRow>(JOURNAL_STORE).add(journalEntry);
	}
}
