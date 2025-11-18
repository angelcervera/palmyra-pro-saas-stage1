import { sqlite3Worker1Promiser } from "@sqlite.org/sqlite-wasm";
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

type WorkerPromiser = Awaited<ReturnType<typeof sqlite3Worker1Promiser.v2>>;
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
	readonly initialMetadata?: MetadataSnapshot;
	readonly logger?: {
		debug?: (...args: unknown[]) => void;
		error?: (...args: unknown[]) => void;
	};
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

	constructor(options: OfflineSqliteProviderOptions = {}) {
		this.databaseName = options.databaseName ?? DEFAULT_DB_NAME;
		this.vfs = options.vfs ?? DEFAULT_VFS;
		this.logger = options.logger;
		this.workerFactory =
			options.workerFactory ?? (() => this.createDefaultWorker());
		this.promiserPromise = this.createPromiser();
		if (options.initialMetadata) {
			this.metadataCache = options.initialMetadata;
			this.pendingMetadataSeed = options.initialMetadata;
		}
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
				v.definition AS definition
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
			}
			const fetched = Date.parse(row.fetched_at ?? "");
			if (!Number.isNaN(fetched)) {
				fetchedAtEpoch = Math.max(fetchedAtEpoch, fetched);
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
			const rows = await this.all<EntityRow>(
				`SELECT table_name, entity_id, entity_version, schema_version, payload, ts, is_deleted
				FROM entities
				WHERE table_name = ? AND entity_id = ?
				LIMIT 1`,
				[ref.tableName, ref.entityId],
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
			const rows = await this.all<EntityRow>(
				`SELECT table_name, entity_id, entity_version, schema_version, payload, ts, is_deleted
				FROM entities
				WHERE table_name = ? AND is_deleted = 0
				ORDER BY ts DESC
				LIMIT ? OFFSET ?`,
				[scope.tableName, pageSize, offset],
			);
			const total = await this.all<{ total: number }>(
				`SELECT COUNT(1) AS total FROM entities WHERE table_name = ? AND is_deleted = 0`,
				[scope.tableName],
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

	private createPromiser(): Promise<WorkerPromiser> {
		return sqlite3Worker1Promiser.v2({
			worker: () => this.workerFactory(),
			onerror: (error: unknown) =>
				this.logger?.error?.("sqlite worker error", error),
		});
	}

	private async ensureDatabaseReady(): Promise<void> {
		if (this.openPromise) {
			return this.openPromise;
		}
		this.openPromise = this.openDatabase();
		try {
			await this.openPromise;
		} finally {
			this.openPromise = undefined;
		}
	}

	private async openDatabase(): Promise<void> {
		const promiser = await this.promiserPromise;
		const filename = this.buildFilename();
		const response = (await promiser("open", {
			filename,
		})) as WorkerExecResponse;
		const dbId = response.dbId ?? response.result?.dbId;
		if (typeof dbId !== "number") {
			throw new Error("Failed to obtain sqlite database handle");
		}
		this.dbId = dbId;
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
			PRIMARY KEY(table_name, schema_version)
		)`);
		await this.runUnsafe(`CREATE TABLE IF NOT EXISTS entities (
			table_name TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			entity_version TEXT NOT NULL,
			schema_version TEXT NOT NULL,
			payload TEXT NOT NULL,
			ts INTEGER NOT NULL,
			is_deleted INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY(table_name, entity_id)
		)`);
		await this.runUnsafe(
			"CREATE INDEX IF NOT EXISTS idx_entities_table_ts ON entities(table_name, ts DESC)",
		);
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
		const entityId = input.entityId ?? this.generateEntityId();
		const entityVersion = this.generateEntityVersion();
		const ts = Date.now();
		const payload = JSON.stringify(toJsonObject(input.payload));
		const run = context?.unsafe
			? this.runUnsafe.bind(this)
			: this.run.bind(this);
		await run(
			`INSERT INTO entities (table_name, entity_id, entity_version, schema_version, payload, ts, is_deleted)
			VALUES (?, ?, ?, ?, ?, ?, 0)
			ON CONFLICT(table_name, entity_id) DO UPDATE SET
				entity_version = excluded.entity_version,
				schema_version = excluded.schema_version,
				payload = excluded.payload,
				ts = excluded.ts,
				is_deleted = 0`,
			[
				input.tableName,
				entityId,
				entityVersion,
				target.activeVersion,
				payload,
				ts,
			],
		);
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
		const exec = context?.unsafe
			? this.execWithChangesUnsafe.bind(this)
			: this.execWithChanges.bind(this);
		const entityVersion = this.generateEntityVersion();
		const ts = Date.now();
		const updated = await exec(
			`UPDATE entities
			SET is_deleted = 1,
				entity_version = ?,
				ts = ?
			WHERE table_name = ? AND entity_id = ?`,
			[entityVersion, ts, input.tableName, input.entityId],
		);
		if (updated === 0) {
			throw new Error(
				`Entity ${input.entityId} not found in ${input.tableName}`,
			);
		}
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
						"INSERT INTO schema_versions (table_name, schema_version, definition) VALUES (?, ?, ?)",
						[tableName, schemaVersion, JSON.stringify(definition ?? {})],
					);
				}
			}
		});
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

	private async execWithChanges(
		sql: string,
		bind?: WorkerBind,
	): Promise<number> {
		await this.ensureDatabaseReady();
		return this.execWithChangesUnsafe(sql, bind);
	}

	private async execWithChangesUnsafe(
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
};

type EntityRow = {
	table_name: string;
	entity_id: string;
	entity_version: string;
	schema_version: string;
	payload: string;
	ts: number;
	is_deleted: number;
};

export function createOfflineSqliteProvider(
	options?: OfflineSqliteProviderOptions,
): OfflineSqliteProvider {
	return new OfflineSqliteProvider(options);
}
