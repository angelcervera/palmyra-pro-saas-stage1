import {
	type BatchWrite,
	BatchWriteError,
	type DeleteEntityInput,
	type EntityIdentifier,
	type EntityRecord,
	type MetadataSnapshot,
	type PaginatedResult,
	type PaginationQuery,
	type PersistenceProvider,
	type SaveEntityInput,
	type SchemaDefinition,
	type SchemaIdentifier,
	type SchemaMetadata,
} from "../../core";
import { describeProviderError, wrapProviderError } from "../../shared/errors";
import { fromWireJson, type JsonValue, toWireJson } from "../../shared/json";

const DEFAULT_DB_NAME = "palmyra-offline-idb";
const DEFAULT_PAGE_SIZE = 20;
const METADATA_STORE = "schema-metadata";
const JOURNAL_STORE = "entity-journal";

type JournalChangeType = "create" | "update" | "delete";

export interface OfflineIndexedDbProviderOptions {
	readonly databaseName?: string;
	readonly tenantId: string;
	readonly initialMetadata?: MetadataSnapshot;
	readonly logger?: {
		debug?: (...args: unknown[]) => void;
		error?: (...args: unknown[]) => void;
	};
}

type MetadataRow = {
	key: string;
	tenantId: string;
	tableName: string;
	activeVersion: string;
	versions: Record<string, unknown>;
	fetchedAt: string;
};

type EntityRow = {
	entityId: string;
	entityVersion: string;
	schemaVersion: string;
	tableName: string;
	ts: number;
	isDeleted: boolean;
	payload: JsonValue;
};

type JournalRow = {
	changeId?: number;
	tenantId: string;
	tableName: string;
	entityId: string;
	entityVersion: string;
	schemaVersion: string;
	changeType: JournalChangeType;
	payload?: JsonValue;
};

export type OfflineIndexedDbJournalEntry = {
	changeId: number;
	tableName: string;
	entityId: string;
	entityVersion: string;
	schemaVersion: string;
	changeType: JournalChangeType;
	payload?: JsonValue;
};

export function createOfflineIndexedDbProvider(
	options: OfflineIndexedDbProviderOptions,
): OfflineIndexedDbProvider {
	return new OfflineIndexedDbProvider(options);
}

export class OfflineIndexedDbProvider implements PersistenceProvider {
	readonly name = "offline-indexeddb";
	readonly description =
		"Offline provider backed by IndexedDB with tenant-prefixed stores and a change journal";

	private dbPromise?: Promise<IDBDatabase>;
	private dbVersion?: number;
	private tenantId: string;
	private readonly databaseName: string;
	private readonly logger?: OfflineIndexedDbProviderOptions["logger"];
	private metadataCache?: MetadataSnapshot;
	private pendingMetadataSeed?: MetadataSnapshot;

	constructor(options: OfflineIndexedDbProviderOptions) {
		if (!options.tenantId) {
			throw new Error("tenantId is required for the IndexedDB provider");
		}
		this.tenantId = options.tenantId;
		this.databaseName = options.databaseName ?? DEFAULT_DB_NAME;
		this.logger = options.logger;
		if (options.initialMetadata) {
			this.metadataCache = options.initialMetadata;
			this.pendingMetadataSeed = options.initialMetadata;
		}
		if (typeof indexedDB === "undefined") {
			throw new Error("IndexedDB is not available in this environment");
		}
	}

	async getMetadata(): Promise<MetadataSnapshot> {
		await this.ensureDatabaseReady();
		if (this.metadataCache) {
			return this.metadataCache;
		}
		const db = await this.getDatabase([METADATA_STORE]);
		const rows = await this.getMetadataRows(db);
		const snapshot = this.buildSnapshotFromRows(rows);
		this.metadataCache = snapshot;
		return snapshot;
	}

	async getEntity<TPayload>(
		ref: EntityIdentifier,
	): Promise<EntityRecord<TPayload>> {
		await this.ensureDatabaseReady();
		const storeName = this.buildEntityStoreName(ref.tableName);
		const db = await this.ensureEntityStore(storeName);
		const tx = db.transaction(storeName, "readonly");
		const store = tx.objectStore(storeName);
		const row = (await requestToPromise<EntityRow | undefined>(
			store.get(ref.entityId),
		)) as EntityRow | undefined;
		await transactionDone(tx);
		if (!row) {
			throw new Error(
				`Entity ${ref.entityId} not found in ${ref.tableName} for tenant ${this.tenantId}`,
			);
		}
		return this.toEntityRecord<TPayload>(row);
	}

	async queryEntities<TPayload>(
		scope: SchemaIdentifier,
		pagination?: PaginationQuery,
	): Promise<PaginatedResult<EntityRecord<TPayload>>> {
		await this.ensureDatabaseReady();
		const storeName = this.buildEntityStoreName(scope.tableName);
		const db = await this.ensureEntityStore(storeName);
		const tx = db.transaction(storeName, "readonly");
		const store = tx.objectStore(storeName);
		const rows = await requestToPromise<EntityRow[]>(store.getAll());
		await transactionDone(tx);
		const activeRows = rows
			.filter((row) => !row.isDeleted)
			.sort((a, b) => b.ts - a.ts);
		const pageSize = Math.max(1, pagination?.pageSize ?? DEFAULT_PAGE_SIZE);
		const page = Math.max(1, pagination?.page ?? 1);
		const start = (page - 1) * pageSize;
		const slice = activeRows.slice(start, start + pageSize);
		const totalItems = activeRows.length;
		const totalPages = totalItems === 0 ? 0 : Math.ceil(totalItems / pageSize);
		return {
			items: slice.map((row) => this.toEntityRecord<TPayload>(row)),
			page,
			pageSize,
			totalItems,
			totalPages,
		};
	}

	async saveEntity<TPayload>(
		input: SaveEntityInput<TPayload>,
	): Promise<EntityRecord<TPayload>> {
		await this.ensureDatabaseReady();
		try {
			const metadata = await this.requireSchemaMetadata(input.tableName);
			const storeName = this.buildEntityStoreName(input.tableName);
			const db = await this.ensureEntityStore(storeName);
			const entityId = input.entityId ?? this.generateEntityId();
			const entityVersion = this.generateEntityVersion();
			const ts = Date.now();
			const payload = toWireJson(input.payload);
			const tx = db.transaction([storeName, JOURNAL_STORE], "readwrite");
			const store = tx.objectStore(storeName);
			const previous = (await requestToPromise<EntityRow | undefined>(
				store.get(entityId),
			)) as EntityRow | undefined;
			const row: EntityRow = {
				entityId,
				entityVersion,
				schemaVersion: metadata.activeVersion,
				tableName: input.tableName,
				ts,
				isDeleted: false,
				payload,
			};
			store.put(row);
			this.appendJournal(tx, {
				changeType: previous ? "update" : "create",
				entityId,
				entityVersion,
				schemaVersion: metadata.activeVersion,
				tableName: input.tableName,
				payload,
			});
			await transactionDone(tx);
			return this.toEntityRecord<TPayload>(row);
		} catch (error) {
			throw wrapProviderError(
				`Failed to persist entity in ${input.tableName}`,
				error,
			);
		}
	}

	async deleteEntity(input: DeleteEntityInput): Promise<void> {
		await this.ensureDatabaseReady();
		try {
			const storeName = this.buildEntityStoreName(input.tableName);
			const db = await this.ensureEntityStore(storeName);
			const tx = db.transaction([storeName, JOURNAL_STORE], "readwrite");
			const store = tx.objectStore(storeName);
			const existing = (await requestToPromise<EntityRow | undefined>(
				store.get(input.entityId),
			)) as EntityRow | undefined;
			if (!existing) {
				throw new Error(
					`Entity ${input.entityId} not found in ${input.tableName}`,
				);
			}
			const updated: EntityRow = {
				...existing,
				ts: Date.now(),
				isDeleted: true,
			};
			store.put(updated);
			this.appendJournal(tx, {
				changeType: "delete",
				entityId: input.entityId,
				entityVersion: existing.entityVersion,
				schemaVersion: existing.schemaVersion,
				tableName: input.tableName,
				payload: existing.payload,
			});
			await transactionDone(tx);
		} catch (error) {
			throw wrapProviderError(
				`Failed to delete entity ${input.entityId} from ${input.tableName}`,
				error,
			);
		}
	}

	async batchWrites(operations: BatchWrite[]): Promise<void> {
		if (operations.length === 0) {
			return;
		}
		await this.ensureDatabaseReady();
		const storeNames = new Set<string>();
		for (const op of operations) {
			storeNames.add(this.buildEntityStoreName(op.data.tableName));
		}
		storeNames.add(JOURNAL_STORE);
		const db = await this.getDatabase([...storeNames]);
		const metadata = await this.getMetadata();
		const tx = db.transaction([...storeNames], "readwrite");
		try {
			for (const op of operations) {
				const storeName = this.buildEntityStoreName(op.data.tableName);
				const store = tx.objectStore(storeName);
				if (op.type === "save") {
					const meta = metadata.tables.get(op.data.tableName);
					if (!meta) {
						throw new BatchWriteError({
							tableName: op.data.tableName,
							entityId: op.data.entityId,
							reason: "Schema metadata missing",
						});
					}
					const entityId = op.data.entityId ?? this.generateEntityId();
					const entityVersion = this.generateEntityVersion();
					const previous = (await requestToPromise<EntityRow | undefined>(
						store.get(entityId),
					)) as EntityRow | undefined;
					const row: EntityRow = {
						entityId,
						entityVersion,
						schemaVersion: meta.activeVersion,
						tableName: op.data.tableName,
						ts: Date.now(),
						isDeleted: false,
						payload: toWireJson(op.data.payload),
					};
					store.put(row);
					this.appendJournal(tx, {
						changeType: previous ? "update" : "create",
						entityId,
						entityVersion,
						schemaVersion: meta.activeVersion,
						tableName: op.data.tableName,
						payload: row.payload,
					});
				} else {
					const data = op.data;
					const existing = (await requestToPromise<EntityRow | undefined>(
						store.get(data.entityId),
					)) as EntityRow | undefined;
					if (!existing) {
						throw new BatchWriteError({
							tableName: data.tableName,
							entityId: data.entityId,
							reason: "Entity not found",
						});
					}
					store.put({
						...existing,
						ts: Date.now(),
						isDeleted: true,
					});
					this.appendJournal(tx, {
						changeType: "delete",
						entityId: data.entityId,
						entityVersion: existing.entityVersion,
						schemaVersion: existing.schemaVersion,
						tableName: data.tableName,
						payload: existing.payload,
					});
				}
			}
			await transactionDone(tx);
		} catch (error) {
			tx.abort();
			if (error instanceof BatchWriteError) {
				throw error;
			}
			const op = operations[0];
			throw new BatchWriteError({
				tableName: op.data.tableName,
				entityId: op.type === "delete" ? op.data.entityId : op.data.entityId,
				reason: describeProviderError(error),
			});
		}
	}

	async replaceMetadata(snapshot: MetadataSnapshot): Promise<void> {
		await this.ensureDatabaseReady();
		const storeNames = [...snapshot.tables.keys()].map((table) =>
			this.buildEntityStoreName(table),
		);
		const db = await this.getDatabase([METADATA_STORE, ...storeNames]);
		await this.writeMetadataSnapshot(db, snapshot);
		this.metadataCache = snapshot;
	}

	async listJournalEntries(): Promise<OfflineIndexedDbJournalEntry[]> {
		await this.ensureDatabaseReady();
		const db = await this.getDatabase([JOURNAL_STORE]);
		const tx = db.transaction(JOURNAL_STORE, "readonly");
		const store = tx.objectStore(JOURNAL_STORE);
		const index = store.index("byTenantChangeId");
		const entries: OfflineIndexedDbJournalEntry[] = [];
		const range = IDBKeyRange.bound(
			[this.tenantId, Number.MIN_SAFE_INTEGER],
			[this.tenantId, Number.MAX_SAFE_INTEGER],
		);
		await iterateCursor(index.openCursor(range), (cursor) => {
			const value = cursor.value as JournalRow;
			if (value.changeId !== undefined) {
				entries.push({
					changeId: value.changeId,
					tableName: value.tableName,
					entityId: value.entityId,
					entityVersion: value.entityVersion,
					schemaVersion: value.schemaVersion,
					changeType: value.changeType,
					payload: value.payload,
				});
			}
		});
		await transactionDone(tx);
		return entries;
	}

	async clearJournalEntries(): Promise<void> {
		await this.ensureDatabaseReady();
		const db = await this.getDatabase([JOURNAL_STORE]);
		const tx = db.transaction(JOURNAL_STORE, "readwrite");
		const store = tx.objectStore(JOURNAL_STORE);
		const index = store.index("byTenant");
		const range = IDBKeyRange.only(this.tenantId);
		await iterateCursor(index.openCursor(range), (cursor) => {
			cursor.delete();
		});
		await transactionDone(tx);
	}

	async setActiveTenant(tenantId: string): Promise<void> {
		if (!tenantId) {
			throw new Error("tenantId must be provided");
		}
		if (tenantId === this.tenantId) {
			return;
		}
		this.tenantId = tenantId;
		this.metadataCache = undefined;
	}

	async close(): Promise<void> {
		if (!this.dbPromise) {
			return;
		}
		const db = await this.dbPromise;
		db.close();
		this.dbPromise = undefined;
		this.dbVersion = undefined;
	}

	private async ensureDatabaseReady(): Promise<void> {
		if (!this.dbPromise) {
			this.dbPromise = this.openDatabase();
		}
		const db = await this.dbPromise;
		if (this.pendingMetadataSeed) {
			await this.writeMetadataSnapshot(db, this.pendingMetadataSeed);
			this.pendingMetadataSeed = undefined;
		}
	}

	private openDatabase(version?: number): Promise<IDBDatabase> {
		return new Promise((resolve, reject) => {
			const request = indexedDB.open(this.databaseName, version);
			request.onupgradeneeded = () => {
				const db = request.result;
				this.ensureBaseStores(db);
			};
			request.onerror = () =>
				reject(request.error ?? new Error("Failed to open IndexedDB database"));
			request.onsuccess = () => {
				this.dbVersion = request.result.version;
				resolve(request.result);
			};
		});
	}

	private ensureBaseStores(db: IDBDatabase): void {
		if (!db.objectStoreNames.contains(METADATA_STORE)) {
			const store = db.createObjectStore(METADATA_STORE, { keyPath: "key" });
			store.createIndex("byTenant", "tenantId", { unique: false });
		}
		if (!db.objectStoreNames.contains(JOURNAL_STORE)) {
			const store = db.createObjectStore(JOURNAL_STORE, {
				keyPath: "changeId",
				autoIncrement: true,
			});
			store.createIndex("byTenant", "tenantId", { unique: false });
			store.createIndex("byTenantChangeId", ["tenantId", "changeId"], {
				unique: false,
			});
		}
	}

	private async ensureEntityStore(storeName: string): Promise<IDBDatabase> {
		return await this.getDatabase([storeName]);
	}

	private async getDatabase(requiredStores: string[]): Promise<IDBDatabase> {
		if (!this.dbPromise) {
			this.dbPromise = this.openDatabase();
		}
		let db = await this.dbPromise;
		const missing = requiredStores.filter(
			(name) => !db.objectStoreNames.contains(name),
		);
		if (missing.length > 0) {
			db.close();
			const nextVersion = (this.dbVersion ?? db.version ?? 1) + 1;
			this.dbPromise = new Promise((resolve, reject) => {
				const request = indexedDB.open(this.databaseName, nextVersion);
				request.onupgradeneeded = () => {
					const upgradeDb = request.result;
					this.ensureBaseStores(upgradeDb);
					for (const storeName of missing) {
						if (!upgradeDb.objectStoreNames.contains(storeName)) {
							upgradeDb.createObjectStore(storeName, {
								keyPath: "entityId",
							});
						}
					}
				};
				request.onerror = () =>
					reject(
						request.error ?? new Error("Failed to upgrade IndexedDB database"),
					);
				request.onsuccess = () => {
					this.dbVersion = request.result.version;
					resolve(request.result);
				};
			});
			db = await this.dbPromise;
		}
		return db;
	}

	private async getMetadataRows(db: IDBDatabase): Promise<MetadataRow[]> {
		const tx = db.transaction(METADATA_STORE, "readonly");
		const store = tx.objectStore(METADATA_STORE);
		const index = store.index("byTenant");
		const rows = (await requestToPromise<MetadataRow[]>(
			index.getAll(this.tenantId),
		)) as MetadataRow[];
		await transactionDone(tx);
		return rows;
	}

	private buildSnapshotFromRows(rows: MetadataRow[]): MetadataSnapshot {
		const tables = new Map<string, SchemaMetadata>();
		let fetchedAt = 0;
		for (const row of rows) {
			const versions = new Map<string, SchemaDefinition>();
			for (const [version, definition] of Object.entries(row.versions ?? {})) {
				versions.set(version, (definition ?? {}) as SchemaDefinition);
			}
			tables.set(row.tableName, {
				tableName: row.tableName,
				activeVersion: row.activeVersion,
				versions,
			});
			const parsed = Date.parse(row.fetchedAt);
			if (!Number.isNaN(parsed)) {
				fetchedAt = Math.max(fetchedAt, parsed);
			}
		}
		return {
			tables,
			fetchedAt: fetchedAt ? new Date(fetchedAt) : new Date(0),
		};
	}

	private async writeMetadataSnapshot(
		db: IDBDatabase,
		snapshot: MetadataSnapshot,
	): Promise<void> {
		const tx = db.transaction(METADATA_STORE, "readwrite");
		const store = tx.objectStore(METADATA_STORE);
		const index = store.index("byTenant");
		const range = IDBKeyRange.only(this.tenantId);
		await iterateCursor(index.openCursor(range), (cursor) => cursor.delete());
		for (const [tableName, meta] of snapshot.tables.entries()) {
			const key = this.buildMetadataKey(tableName);
			store.put({
				key,
				tenantId: this.tenantId,
				tableName,
				activeVersion: meta.activeVersion,
				versions: Object.fromEntries(meta.versions.entries()),
				fetchedAt: snapshot.fetchedAt.toISOString(),
			});
		}
		await transactionDone(tx);
	}

	private buildEntityStoreName(tableName: string): string {
		return `${this.tenantId}::${tableName}`;
	}

	private buildMetadataKey(tableName: string): string {
		return `${this.tenantId}::${tableName}`;
	}

	private async requireSchemaMetadata(
		tableName: string,
	): Promise<SchemaMetadata> {
		const metadata = await this.getMetadata();
		const entry = metadata.tables.get(tableName);
		if (!entry) {
			throw new Error(`Schema metadata missing for ${tableName}`);
		}
		return entry;
	}

	private appendJournal(
		tx: IDBTransaction,
		entry: Omit<JournalRow, "tenantId">,
	): void {
		const store = tx.objectStore(JOURNAL_STORE);
		store.add({
			tenantId: this.tenantId,
			...entry,
		});
	}

	private toEntityRecord<TPayload>(row: EntityRow): EntityRecord<TPayload> {
		return {
			tableName: row.tableName,
			entityId: row.entityId,
			entityVersion: row.entityVersion,
			schemaVersion: row.schemaVersion,
			payload: fromWireJson<TPayload>(row.payload),
			ts: new Date(row.ts),
			isDeleted: row.isDeleted,
		};
	}

	private generateEntityId(): string {
		if (globalThis.crypto?.randomUUID) {
			return globalThis.crypto.randomUUID();
		}
		return `ent_${Math.random().toString(36).slice(2, 11)}`;
	}

	private generateEntityVersion(): string {
		return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
	}
}

async function requestToPromise<T>(request: IDBRequest<T>): Promise<T> {
	return await new Promise<T>((resolve, reject) => {
		request.onsuccess = () => resolve(request.result);
		request.onerror = () =>
			reject(request.error ?? new Error("IndexedDB request failed"));
	});
}

async function transactionDone(tx: IDBTransaction): Promise<void> {
	return await new Promise<void>((resolve, reject) => {
		tx.oncomplete = () => resolve();
		tx.onerror = () =>
			reject(tx.error ?? new Error("IndexedDB transaction failed"));
		tx.onabort = () =>
			reject(tx.error ?? new Error("IndexedDB transaction aborted"));
	});
}

async function iterateCursor(
	request: IDBRequest<IDBCursorWithValue | null>,
	fn: (cursor: IDBCursorWithValue) => void,
): Promise<void> {
	return await new Promise<void>((resolve, reject) => {
		request.onerror = () =>
			reject(request.error ?? new Error("IndexedDB cursor error"));
		request.onsuccess = () => {
			const cursor = request.result;
			if (!cursor) {
				resolve();
				return;
			}
			fn(cursor);
			cursor.continue();
		};
	});
}
