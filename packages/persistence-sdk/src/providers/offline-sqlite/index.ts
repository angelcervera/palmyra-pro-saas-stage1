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
	type SchemaIdentifier,
	type SchemaMetadata,
} from "../../core";
import { describeProviderError, wrapProviderError } from "../../shared/errors";
import { fromWireJson, type JsonValue, toJsonObject } from "../../shared/json";

const DEFAULT_DB_NAME = "/palmyra-offline.db";
const DEFAULT_VFS = "opfs-sahpool";
const DEFAULT_PAGE_SIZE = 20;

type WorkerPromiser = (
	type: string,
	args?: Record<string, unknown>,
) => Promise<Record<string, unknown>>;

type WorkerPromiserFactoryModule = {
	v2(config?: Record<string, unknown>): Promise<WorkerPromiser>;
};
let cachedWorkerPromiser: WorkerPromiserFactoryModule | undefined;

async function resolveWorkerPromiser(): Promise<WorkerPromiserFactoryModule> {
	if (!cachedWorkerPromiser) {
		const mod = await import("@sqlite.org/sqlite-wasm");
		cachedWorkerPromiser =
			mod.sqlite3Worker1Promiser as unknown as WorkerPromiserFactoryModule;
	}
	return cachedWorkerPromiser;
}
type WorkerBind = ReadonlyArray<unknown> | Record<string, unknown>;

type WorkerExecResponse<T = unknown> = {
	dbId?: number;
	result?: {
		dbId?: number;
		resultRows?: T[];
		changeCount?: number;
	};
};

interface ExecContext {
	unsafe?: boolean;
}

export interface OfflineSqliteProviderOptions {
	readonly databaseName?: string;
	readonly vfs?: "opfs-sahpool" | "kvvfs";
	readonly workerFactory?: () => Worker;
	readonly promiserFactory?: () => Promise<WorkerPromiser>;
	readonly initialMetadata?: MetadataSnapshot;
	readonly logger?: {
		debug?: (...args: unknown[]) => void;
		error?: (...args: unknown[]) => void;
	};
	readonly enableJournal?: boolean;
}

export class OfflineSqliteProvider implements PersistenceProvider {
	readonly name = "offline-sqlite";
	readonly description =
		"Offline provider backed by sqlite-wasm and a Safari-friendly OPFS SyncAccessHandle pool";

	private readonly workerFactory: () => Worker;
	private readonly logger: OfflineSqliteProviderOptions["logger"];
	private readonly promiserPromise: Promise<WorkerPromiser>;
	private readonly vfs: string;
	private databaseName: string;
	private openPromise?: Promise<void>;
	private dbId?: number;
	private metadataCache?: MetadataSnapshot;
	private pendingMetadataSeed?: MetadataSnapshot;
	private readonly ensuredEntityTables = new Set<string>();

	constructor(options: OfflineSqliteProviderOptions = {}) {
		this.databaseName = options.databaseName ?? DEFAULT_DB_NAME;
		this.vfs = options.vfs ?? DEFAULT_VFS;
		this.logger = options.logger;
		this.workerFactory =
			options.workerFactory ?? (() => this.createDefaultWorker());
		this.promiserPromise = options.promiserFactory
			? options.promiserFactory()
			: this.createPromiser();
		if (options.initialMetadata) {
			this.metadataCache = options.initialMetadata;
			this.pendingMetadataSeed = options.initialMetadata;
		}
	}

	private sanitizeIdentifier(identifier: string): string {
		if (!/^[A-Za-z0-9_]+$/.test(identifier)) {
			throw new Error(`Invalid identifier: ${identifier}`);
		}
		return identifier;
	}

	private quoteIdentifier(identifier: string): string {
		return `"${this.sanitizeIdentifier(identifier)}"`;
	}

	private async ensureEntityTableExists(tableName: string): Promise<string> {
		const sanitized = this.sanitizeIdentifier(tableName);
		if (this.ensuredEntityTables.has(sanitized)) {
			return this.quoteIdentifier(sanitized);
		}
		const tableIdent = this.quoteIdentifier(sanitized);
		await this.runUnsafe(`CREATE TABLE IF NOT EXISTS ${tableIdent} (
			entity_id TEXT NOT NULL,
			entity_version TEXT NOT NULL,
			schema_version TEXT NOT NULL,
			payload TEXT NOT NULL,
			ts INTEGER NOT NULL,
			is_deleted INTEGER NOT NULL DEFAULT 0,
			is_active INTEGER NOT NULL DEFAULT 1,
			PRIMARY KEY(entity_id, entity_version)
		)`);
		const activeIndex = this.quoteIdentifier(`idx_${sanitized}_entity_active`);
		await this.runUnsafe(
			`CREATE UNIQUE INDEX IF NOT EXISTS ${activeIndex} ON ${tableIdent} (entity_id) WHERE is_active = 1`,
		);
		const tsIndex = this.quoteIdentifier(`idx_${sanitized}_entity_ts`);
		await this.runUnsafe(
			`CREATE INDEX IF NOT EXISTS ${tsIndex} ON ${tableIdent} (ts DESC)`,
		);
		this.ensuredEntityTables.add(sanitized);
		return tableIdent;
	}

	async getMetadata(): Promise<MetadataSnapshot> {
		await this.ensureDatabaseReady();
		if (this.metadataCache) {
			return this.metadataCache;
		}
		const rows = await this.all<MetadataRow>(
			`SELECT
				m.table_name AS table_name,
				m.active_version AS active_version,
				m.fetched_at AS fetched_at,
				v.schema_version AS schema_version,
				v.definition AS definition,
				v.is_active AS version_is_active
			FROM schema_metadata m
			LEFT JOIN schema_versions v ON v.table_name = m.table_name
			ORDER BY m.table_name, v.schema_version`,
		);
		const tables = new Map<string, SchemaMetadata>();
		let fetchedAtEpoch = 0;
		for (const row of rows) {
			let entry = tables.get(row.table_name);
			if (!entry) {
				entry = {
					tableName: row.table_name,
					activeVersion: row.active_version,
					versions: new Map(),
				};
				tables.set(row.table_name, entry);
			}
			if (row.schema_version) {
				try {
					entry.versions.set(
						row.schema_version,
						JSON.parse(row.definition ?? "{}"),
					);
				} catch (error) {
					this.logger?.error?.("Failed to parse schema definition", error);
				}
				if (row.version_is_active === 1) {
					entry.activeVersion = row.schema_version;
				}
			}
			if (!entry.activeVersion && row.active_version) {
				entry.activeVersion = row.active_version;
			}
			const fetched = Date.parse(row.fetched_at ?? "");
			if (!Number.isNaN(fetched)) {
				fetchedAtEpoch = Math.max(fetchedAtEpoch, fetched);
			}
		}
		for (const entry of tables.values()) {
			if (!entry.activeVersion) {
				const first = entry.versions.keys().next().value;
				if (first) {
					entry.activeVersion = first;
				}
			}
		}
		const snapshot: MetadataSnapshot = {
			tables,
			fetchedAt: fetchedAtEpoch ? new Date(fetchedAtEpoch) : new Date(0),
		};
		this.metadataCache = snapshot;
		return snapshot;
	}

	async getEntity<TPayload>(
		ref: EntityIdentifier,
	): Promise<EntityRecord<TPayload>> {
		await this.ensureDatabaseReady();
		try {
			const tableIdent = await this.ensureEntityTableExists(ref.tableName);
			const tableLiteral = `'${this.sanitizeIdentifier(ref.tableName)}'`;
			const rows = await this.all<EntityRow>(
				"SELECT " +
					tableLiteral +
					" AS table_name, entity_id, entity_version, schema_version, payload, ts, is_deleted, is_active FROM " +
					tableIdent +
					" WHERE entity_id = ? ORDER BY ts DESC LIMIT 1",
				[ref.entityId],
			);
			const row = rows[0];
			if (!row) {
				throw new Error(`Entity ${ref.entityId} not found in ${ref.tableName}`);
			}
			return this.toEntityRecord<TPayload>(row);
		} catch (error) {
			throw wrapProviderError("Failed to load entity", error);
		}
	}

	async queryEntities<TPayload>(
		scope: SchemaIdentifier,
		pagination?: PaginationQuery,
	): Promise<PaginatedResult<EntityRecord<TPayload>>> {
		await this.ensureDatabaseReady();
		const pageSize = Math.max(1, pagination?.pageSize ?? DEFAULT_PAGE_SIZE);
		const page = Math.max(1, pagination?.page ?? 1);
		const offset = (page - 1) * pageSize;
		try {
			const tableIdent = await this.ensureEntityTableExists(scope.tableName);
			const tableLiteral = `'${this.sanitizeIdentifier(scope.tableName)}'`;
			const rows = await this.all<EntityRow>(
				"SELECT " +
					tableLiteral +
					" AS table_name, entity_id, entity_version, schema_version, payload, ts, is_deleted, is_active FROM " +
					tableIdent +
					" WHERE is_active = 1 AND is_deleted = 0 ORDER BY ts DESC LIMIT ? OFFSET ?",
				[pageSize, offset],
			);
			const total = await this.all<{ total: number }>(
				"SELECT COUNT(1) AS total FROM " +
					tableIdent +
					" WHERE is_active = 1 AND is_deleted = 0",
			);
			const totalItems = Number(total[0]?.total ?? 0);
			const totalPages =
				totalItems === 0 ? 0 : Math.ceil(totalItems / pageSize);
			return {
				items: rows.map((row) => this.toEntityRecord<TPayload>(row)),
				page,
				pageSize,
				totalItems,
				totalPages,
			};
		} catch (error) {
			throw wrapProviderError("Failed to query local entities", error);
		}
	}

	async saveEntity<TPayload>(
		input: SaveEntityInput<TPayload>,
	): Promise<EntityRecord<TPayload>> {
		await this.ensureDatabaseReady();
		try {
			return await this.persistEntity(input);
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
			await this.performDelete(input);
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
		await this.transaction(async () => {
			for (const operation of operations) {
				try {
					if (operation.type === "save") {
						await this.persistEntity(operation.data, { unsafe: true });
					} else {
						await this.performDelete(operation.data, { unsafe: true });
					}
				} catch (error) {
					throw new BatchWriteError({
						tableName: operation.data.tableName,
						entityId: (operation.data as DeleteEntityInput).entityId,
						reason: describeProviderError(error),
					});
				}
			}
		});
	}

	async replaceMetadata(snapshot: MetadataSnapshot): Promise<void> {
		await this.ensureDatabaseReady();
		await this.writeMetadataSnapshot(snapshot);
	}

	async setActiveDatabase(name: string): Promise<void> {
		if (!name) {
			throw new Error("Database name must be provided");
		}
		if (name === this.databaseName && this.dbId) {
			return;
		}
		await this.close();
		this.databaseName = name;
		this.metadataCache = undefined;
		await this.ensureDatabaseReady();
	}

	async close(): Promise<void> {
		if (!this.dbId) {
			this.openPromise = undefined;
			return;
		}
		const promiser = await this.promiserPromise;
		await promiser("close", { dbId: this.dbId });
		this.dbId = undefined;
		this.openPromise = undefined;
	}

	private createDefaultWorker(): Worker {
		if (typeof Worker === "undefined") {
			throw new Error("Web Workers are not supported in this environment");
		}
		return new Worker(new URL("./sqlite-worker.js", import.meta.url), {
			type: "module",
		});
	}

	private async createPromiser(): Promise<WorkerPromiser> {
		// Tests (and potential future environments) can inject a custom promiser
		// instead of relying on a Worker + sqlite-wasm bundle. We still lazily
		// import the browser version here so Node-only runs never pull the wasm
		// worker unless needed. If we add more providers later, consider splitting
		// implementations into dedicated packages to keep these concerns isolated.
		this.logger?.debug?.("spawning sqlite worker");
		const promiserModule = await resolveWorkerPromiser();
		return promiserModule.v2({
			worker: () => this.workerFactory(),
			onerror: (error: unknown) =>
				this.logger?.error?.("sqlite worker error", error),
		});
	}

	private async ensureDatabaseReady(): Promise<void> {
		if (this.dbId) {
			return;
		}
		if (!this.openPromise) {
			const task = this.openDatabase();
			this.openPromise = task
				.then(() => {
					this.openPromise = undefined;
				})
				.catch((error) => {
					this.openPromise = undefined;
					throw error;
				});
		}
		await this.openPromise;
	}

	private async openDatabase(): Promise<void> {
		if (this.dbId) {
			return;
		}
		this.logger?.debug?.("opening database", this.databaseName);
		const promiser = await this.promiserPromise;
		const filename = this.buildFilename();
		this.logger?.debug?.("computed filename", filename);
		const response = (await promiser("open", {
			filename,
		})) as WorkerExecResponse;
		const dbId = response.dbId ?? response.result?.dbId;
		if (typeof dbId !== "number") {
			throw new Error("Failed to obtain sqlite database handle");
		}
		this.dbId = dbId;
		this.logger?.debug?.("database opened", dbId);
		await this.bootstrapSchema();
	}

	private buildFilename(): string {
		if (this.vfs === "kvvfs") {
			return `file:${this.databaseName}?vfs=kvvfs`;
		}
		const normalized = this.databaseName.startsWith("/")
			? this.databaseName
			: `/${this.databaseName}`;
		return `file:${normalized}?vfs=${this.vfs}`;
	}

	private async bootstrapSchema(): Promise<void> {
		await this.runUnsafe("PRAGMA journal_mode=DELETE");
		await this.runUnsafe("PRAGMA foreign_keys=ON");
		await this.runUnsafe(`CREATE TABLE IF NOT EXISTS schema_metadata (
			table_name TEXT PRIMARY KEY,
			active_version TEXT NOT NULL,
			fetched_at TEXT NOT NULL
		)`);
		await this.runUnsafe(`CREATE TABLE IF NOT EXISTS schema_versions (
			table_name TEXT NOT NULL,
			schema_version TEXT NOT NULL,
			definition TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY(table_name, schema_version)
		)`);
		await this.runUnsafe(`CREATE TABLE IF NOT EXISTS entity_journal (
			change_id INTEGER PRIMARY KEY AUTOINCREMENT,
			table_name TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			entity_version TEXT NOT NULL,
			schema_version TEXT NOT NULL,
			change_type TEXT NOT NULL,
			payload TEXT
		)`);
		if (this.pendingMetadataSeed) {
			await this.writeMetadataSnapshot(this.pendingMetadataSeed, {
				unsafe: true,
			});
			this.pendingMetadataSeed = undefined;
		}
	}

	private async persistEntity<TPayload>(
		input: SaveEntityInput<TPayload>,
		context?: ExecContext,
	): Promise<EntityRecord<TPayload>> {
		const target = await this.requireSchemaMetadata(input.tableName);
		const tableIdent = await this.ensureEntityTableExists(input.tableName);
		const entityId = input.entityId ?? this.generateEntityId();
		const entityVersion = this.generateEntityVersion();
		const ts = Date.now();
		const payload = JSON.stringify(toJsonObject(input.payload));
		const run = context?.unsafe
			? this.runUnsafe.bind(this)
			: this.run.bind(this);
		const runWithChanges = context?.unsafe
			? this.runWithChangeCountUnsafe.bind(this)
			: this.runWithChangeCount.bind(this);
		const previousVersions = await runWithChanges(
			"UPDATE " +
				tableIdent +
				" SET is_active = 0 WHERE entity_id = ? AND is_active = 1",
			[entityId],
		);
		await run(
			"INSERT INTO " +
				tableIdent +
				" (entity_id, entity_version, schema_version, payload, ts, is_deleted, is_active) VALUES (?, ?, ?, ?, ?, 0, 1)",
			[entityId, entityVersion, target.activeVersion, payload, ts],
		);
		await this.appendJournalEntry({
			changeType: previousVersions > 0 ? "update" : "create",
			tableName: input.tableName,
			entityId,
			entityVersion,
			schemaVersion: target.activeVersion,
			payload,
		});
		return {
			tableName: input.tableName,
			entityId,
			entityVersion,
			schemaVersion: target.activeVersion,
			payload: input.payload,
			ts: new Date(ts),
			isDeleted: false,
		};
	}

	private async performDelete(
		input: DeleteEntityInput,
		context?: ExecContext,
	): Promise<void> {
		const tableIdent = await this.ensureEntityTableExists(input.tableName);
		const tableLiteral = `'${this.sanitizeIdentifier(input.tableName)}'`;
		const existingRows = await this.all<EntityRow>(
			"SELECT " +
				tableLiteral +
				" AS table_name, entity_id, entity_version, schema_version, payload, ts, is_deleted, is_active FROM " +
				tableIdent +
				" WHERE entity_id = ? AND is_active = 1 LIMIT 1",
			[input.entityId],
		);
		const existing = existingRows[0];
		if (!existing) {
			throw new Error(
				`Entity ${input.entityId} not found in ${input.tableName}`,
			);
		}
		const runWithChanges = context?.unsafe
			? this.runWithChangeCountUnsafe.bind(this)
			: this.runWithChangeCount.bind(this);
		const ts = Date.now();
		const updated = await runWithChanges(
			"UPDATE " +
				tableIdent +
				" SET is_deleted = 1, is_active = 0, ts = ? WHERE entity_id = ? AND is_active = 1",
			[ts, input.entityId],
		);
		if (updated === 0) {
			throw new Error(
				`Entity ${input.entityId} not found in ${input.tableName}`,
			);
		}
		await this.appendJournalEntry({
			changeType: "delete",
			tableName: input.tableName,
			entityId: input.entityId,
			entityVersion: existing.entity_version,
			schemaVersion: existing.schema_version,
			payload: existing.payload,
		});
	}

	async listJournalEntries(): Promise<OfflineChangeJournalEntry[]> {
		await this.ensureDatabaseReady();
		const rows = await this.all<JournalRow>(
			"SELECT change_id, table_name, entity_id, entity_version, schema_version, change_type, payload FROM entity_journal ORDER BY change_id ASC",
		);
		return rows.map((row) => ({
			changeId: row.change_id,
			tableName: row.table_name,
			entityId: row.entity_id,
			entityVersion: row.entity_version,
			schemaVersion: row.schema_version,
			changeType: row.change_type as JournalChangeType,
			payload: row.payload ? fromWireJson(JSON.parse(row.payload)) : undefined,
		}));
	}

	async clearJournalEntries(): Promise<void> {
		await this.run("DELETE FROM entity_journal");
		await this.run("DELETE FROM sqlite_sequence WHERE name = 'entity_journal'");
	}

	private async writeMetadataSnapshot(
		snapshot: MetadataSnapshot,
		context?: ExecContext,
	): Promise<void> {
		const run = context?.unsafe
			? this.runUnsafe.bind(this)
			: this.run.bind(this);
		const transaction = context?.unsafe
			? this.transactionUnsafe.bind(this)
			: this.transaction.bind(this);
		await transaction(async () => {
			await run("DELETE FROM schema_versions");
			await run("DELETE FROM schema_metadata");
			for (const [tableName, meta] of snapshot.tables.entries()) {
				await run(
					"INSERT INTO schema_metadata (table_name, active_version, fetched_at) VALUES (?, ?, ?)",
					[tableName, meta.activeVersion, snapshot.fetchedAt.toISOString()],
				);
				for (const [schemaVersion, definition] of meta.versions.entries()) {
					await run(
						"INSERT INTO schema_versions (table_name, schema_version, definition, is_active) VALUES (?, ?, ?, ?)",
						[
							tableName,
							schemaVersion,
							JSON.stringify(definition ?? {}),
							schemaVersion === meta.activeVersion ? 1 : 0,
						],
					);
				}
			}
		});
		for (const tableName of snapshot.tables.keys()) {
			await this.ensureEntityTableExists(tableName);
		}
		this.metadataCache = snapshot;
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

	private async appendJournalEntry(entry: JournalEntryPayload): Promise<void> {
		await this.runUnsafe(
			"INSERT INTO entity_journal (table_name, entity_id, entity_version, schema_version, change_type, payload) VALUES (?, ?, ?, ?, ?, ?)",
			[
				entry.tableName,
				entry.entityId,
				entry.entityVersion,
				entry.schemaVersion,
				entry.changeType,
				entry.payload ?? null,
			],
		);
	}

	private toEntityRecord<TPayload>(row: EntityRow): EntityRecord<TPayload> {
		let parsed: JsonValue;
		try {
			parsed = JSON.parse(row.payload ?? "null");
		} catch (error) {
			this.logger?.error?.("Failed to parse entity payload", error);
			parsed = null;
		}
		return {
			tableName: row.table_name,
			entityId: row.entity_id,
			entityVersion: row.entity_version,
			schemaVersion: row.schema_version,
			payload: fromWireJson<TPayload>(parsed),
			ts: new Date(row.ts),
			isDeleted: Boolean(row.is_deleted),
		};
	}

	private async run(sql: string, bind?: WorkerBind): Promise<void> {
		await this.ensureDatabaseReady();
		await this.runUnsafe(sql, bind);
	}

	private async runUnsafe(sql: string, bind?: WorkerBind): Promise<void> {
		const promiser = await this.promiserPromise;
		await promiser("exec", {
			dbId: this.dbId,
			sql,
			bind,
		});
	}

	private async runWithChangeCount(
		sql: string,
		bind?: WorkerBind,
	): Promise<number> {
		await this.ensureDatabaseReady();
		return this.runWithChangeCountUnsafe(sql, bind);
	}

	private async runWithChangeCountUnsafe(
		sql: string,
		bind?: WorkerBind,
	): Promise<number> {
		const promiser = await this.promiserPromise;
		const response = (await promiser("exec", {
			dbId: this.dbId,
			sql,
			bind,
			countChanges: 1,
		})) as WorkerExecResponse;
		return Number(response.result?.changeCount ?? 0);
	}

	private async all<T>(sql: string, bind?: WorkerBind): Promise<T[]> {
		await this.ensureDatabaseReady();
		return this.allUnsafe<T>(sql, bind);
	}

	private async allUnsafe<T>(sql: string, bind?: WorkerBind): Promise<T[]> {
		const promiser = await this.promiserPromise;
		const response = (await promiser("exec", {
			dbId: this.dbId,
			sql,
			bind,
			rowMode: "object",
			resultRows: [],
		})) as WorkerExecResponse<T>;
		return response.result?.resultRows ?? [];
	}

	private async transaction<T>(fn: () => Promise<T>): Promise<T> {
		await this.ensureDatabaseReady();
		return this.transactionUnsafe(fn);
	}

	private async transactionUnsafe<T>(fn: () => Promise<T>): Promise<T> {
		await this.runUnsafe("BEGIN IMMEDIATE");
		try {
			const result = await fn();
			await this.runUnsafe("COMMIT");
			return result;
		} catch (error) {
			await this.runUnsafe("ROLLBACK").catch(() => undefined);
			throw error;
		}
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

type MetadataRow = {
	table_name: string;
	active_version: string;
	fetched_at?: string;
	schema_version?: string;
	definition?: string;
	version_is_active?: number;
};

type EntityRow = {
	table_name: string;
	entity_id: string;
	entity_version: string;
	schema_version: string;
	payload: string;
	ts: number;
	is_deleted: number;
	is_active: number;
};

type JournalChangeType = "create" | "update" | "delete";

type JournalRow = {
	change_id: number;
	table_name: string;
	entity_id: string;
	entity_version: string;
	schema_version: string;
	change_type: JournalChangeType;
	payload?: string | null;
};

type JournalEntryPayload = {
	tableName: string;
	entityId: string;
	entityVersion: string;
	schemaVersion: string;
	changeType: JournalChangeType;
	payload?: string | null;
};

export type OfflineChangeJournalEntry = {
	changeId: number;
	tableName: string;
	entityId: string;
	entityVersion: string;
	schemaVersion: string;
	changeType: JournalChangeType;
	payload?: JsonValue;
};

export function createOfflineSqliteProvider(
	options?: OfflineSqliteProviderOptions,
): OfflineSqliteProvider {
	return new OfflineSqliteProvider(options);
}
