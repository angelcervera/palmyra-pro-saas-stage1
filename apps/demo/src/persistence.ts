import {
	type BatchWrite,
	type EntityIdentifier,
	type EntityRecord,
	type PaginatedResult,
	type PaginationQuery,
	PersistenceClient,
	type PersistenceProvider,
	type SaveEntityInput,
	type SchemaDefinition,
	type SchemaIdentifier,
	type SchemaMetadata,
} from "@zengateglobal/persistence-sdk";

export type Person = {
	name: string;
	surname: string;
	age: number;
	dob: string;
	phoneNumber: string;
	photo: string;
};

export interface EntityWithSyncMeta<TPayload> {
	queuedForSync: boolean; // boolean indicating if the entity is queued for sync.
	lastSynced: Date | null; // timestamp of the last sync. null if never synced.
	lastSyncError: string | null; // error message of the last sync.
	entityId: string; // uuid entity identifier
	entityVersion: string; // semantic version e.g. "1.0.0"
	entitySchemaId: string; // uuid schema identifier
	entitySchemaVersion: string; // schema version e.g. "1.0.0"
	entity: TPayload; // the actual entity payload
}

export type PersonRecord = EntityWithSyncMeta<Person>;

const PERSON_TABLE = "persons";
const PERSON_SCHEMA_ID = "00000000-0000-4000-8000-000000000001";
const PERSON_SCHEMA_VERSION = "1.0.0";
const PERSON_SCHEMA_DEFINITION: SchemaDefinition = {
	$schema: "https://json-schema.org/draft/2020-12/schema",
	title: "Person",
	type: "object",
	additionalProperties: false,
	required: ["name", "surname", "age", "dob", "phoneNumber", "photo"],
	properties: {
		name: { type: "string", minLength: 1 },
		surname: { type: "string", minLength: 1 },
		age: { type: "integer", minimum: 0, maximum: 150 },
		dob: { type: "string", format: "date" },
		phoneNumber: { type: "string", pattern: "^\\+?[1-9]\\d{7,14}$" },
		photo: { type: "string", format: "uri" },
	},
};

type InMemoryRow = EntityRecord<PersonRecord>;

class InMemoryPersistenceProvider implements PersistenceProvider {
	readonly name = "demo-memory";
	readonly description = "In-memory persistence provider for demo UI";

	private readonly tables = new Map<string, Map<string, InMemoryRow>>();
	private readonly metadata = new Map<string, SchemaMetadata>([
		[
			PERSON_TABLE,
			{
				tableName: PERSON_TABLE,
				activeVersion: PERSON_SCHEMA_VERSION,
				versions: new Map<string, SchemaDefinition>([
					[PERSON_SCHEMA_VERSION, PERSON_SCHEMA_DEFINITION],
				]),
			},
		],
	]);

	private ensureTable(tableName: string): Map<string, InMemoryRow> {
		let table = this.tables.get(tableName);
		if (!table) {
			table = new Map<string, InMemoryRow>();
			this.tables.set(tableName, table);
		}
		return table;
	}

	async getMetadata() {
		const tables = new Map(this.metadata);
		return { tables, fetchedAt: new Date() };
	}

	async getEntity<TPayload = unknown>(
		ref: EntityIdentifier,
	): Promise<EntityRecord<TPayload>> {
		const table = this.tables.get(ref.tableName);
		if (!table) {
			throw new Error("Entity not found");
		}
		const row = table.get(ref.entityId);
		if (!row || row.isDeleted) {
			throw new Error("Entity not found");
		}
		return row as unknown as EntityRecord<TPayload>;
	}

	async queryEntities<TPayload = unknown>(
		scope: SchemaIdentifier,
		pagination: PaginationQuery = {},
	): Promise<PaginatedResult<EntityRecord<TPayload>>> {
		const table =
			this.tables.get(scope.tableName) ?? new Map<string, InMemoryRow>();
		const rows = Array.from(table.values()).filter((row) => !row.isDeleted);
		rows.sort((a, b) => b.ts.getTime() - a.ts.getTime());

		const page = Math.max(pagination.page ?? 1, 1);
		const pageSize = Math.max(pagination.pageSize ?? 10, 1);
		const start = (page - 1) * pageSize;
		const paged = rows.slice(start, start + pageSize);
		const totalItems = rows.length;
		const totalPages = Math.max(Math.ceil(totalItems / pageSize), 1);

		return {
			items: paged as unknown as EntityRecord<TPayload>[],
			page,
			pageSize,
			totalItems,
			totalPages,
		};
	}

	async saveEntity<TPayload = unknown>(
		input: SaveEntityInput<TPayload>,
	): Promise<EntityRecord<TPayload>> {
		const table = this.ensureTable(input.tableName);
		const entityId = input.entityId ?? crypto.randomUUID();
		const existing = table.get(entityId);
		const entityVersion = this.nextVersion(existing?.entityVersion);
		const ts = new Date();
		const payload = this.normalizePayload(
			input.payload as unknown as PersonRecord,
			entityId,
			entityVersion,
		);

		const row: InMemoryRow = {
			tableName: input.tableName,
			schemaVersion: PERSON_SCHEMA_VERSION,
			entityId,
			entityVersion,
			payload,
			ts,
			isDeleted: false,
		};
		table.set(entityId, row);
		return row as unknown as EntityRecord<TPayload>;
	}

	async deleteEntity(input: EntityIdentifier): Promise<void> {
		const table = this.ensureTable(input.tableName);
		const row = table.get(input.entityId);
		if (!row) return;
		table.set(input.entityId, { ...row, isDeleted: true, ts: new Date() });
	}

	async batchWrites(operations: BatchWrite[]): Promise<void> {
		for (const op of operations) {
			if (op.type === "save") {
				await this.saveEntity(op.data as SaveEntityInput<PersonRecord>);
			} else if (op.type === "delete") {
				await this.deleteEntity(op.data);
			}
		}
	}

	private nextVersion(current?: string): string {
		if (!current) return "1.0.0";
		const [major, minor, patch] = current
			.split(".")
			.map((v) => Number.parseInt(v, 10) || 0);
		const nextPatch = (Number.isFinite(patch) ? patch : 0) + 1;
		return `${major}.${minor}.${nextPatch}`;
	}

	private normalizePayload(
		payload: PersonRecord,
		entityId: string,
		entityVersion: string,
	): PersonRecord {
		return {
			queuedForSync: payload.queuedForSync ?? true,
			lastSynced: payload.lastSynced ?? null,
			lastSyncError: payload.lastSyncError ?? null,
			entityId,
			entityVersion,
			entitySchemaId: PERSON_SCHEMA_ID,
			entitySchemaVersion: PERSON_SCHEMA_VERSION,
			entity: payload.entity,
		};
	}
}

const provider = new InMemoryPersistenceProvider();
const client = new PersistenceClient([provider]);

function unwrap(row: EntityRecord<PersonRecord>): PersonRecord {
	return row.payload;
}

export async function listPersons(options: {
	page?: number;
	pageSize?: number;
	queuedOnly?: boolean;
}): Promise<PaginatedResult<PersonRecord>> {
	const page = Math.max(options.page ?? 1, 1);
	const pageSize = Math.max(options.pageSize ?? 10, 1);
	const result = await client.queryEntities<PersonRecord>(
		{ tableName: PERSON_TABLE },
		{ page: 1, pageSize: 1000 },
	);
	const filteredItems = options.queuedOnly
		? result.items.map(unwrap).filter((item) => item.queuedForSync)
		: result.items.map(unwrap);
	const totalItems = filteredItems.length;
	const totalPages = Math.max(Math.ceil(totalItems / pageSize), 1);
	const start = (page - 1) * pageSize;
	const items = filteredItems.slice(start, start + pageSize);
	return { items, page, pageSize, totalItems, totalPages };
}

export async function getPerson(entityId: string): Promise<PersonRecord> {
	const row = await client.getEntity<PersonRecord>({
		tableName: PERSON_TABLE,
		entityId,
	});
	return unwrap(row);
}

export async function createPerson(input: Person): Promise<PersonRecord> {
	const payload: PersonRecord = {
		queuedForSync: true,
		lastSynced: null,
		lastSyncError: null,
		entityId: "",
		entityVersion: "0.0.0",
		entitySchemaId: PERSON_SCHEMA_ID,
		entitySchemaVersion: PERSON_SCHEMA_VERSION,
		entity: input,
	};
	const row = await client.saveEntity<PersonRecord>({
		tableName: PERSON_TABLE,
		payload,
	});
	return unwrap(row);
}

export async function updatePerson(
	entityId: string,
	input: Person,
): Promise<PersonRecord> {
	const existing = await getPerson(entityId);
	const payload: PersonRecord = {
		...existing,
		queuedForSync: true,
		lastSyncError: null,
		entity: input,
	};
	const row = await client.saveEntity<PersonRecord>({
		tableName: PERSON_TABLE,
		entityId,
		payload,
	});
	return unwrap(row);
}

export async function deletePerson(entityId: string): Promise<void> {
	await client.deleteEntity({ tableName: PERSON_TABLE, entityId });
}

export async function syncAllPersons(): Promise<void> {
	const list = await client.queryEntities<PersonRecord>(
		{ tableName: PERSON_TABLE },
		{ page: 1, pageSize: 1000 },
	);
	const now = new Date();
	const operations: BatchWrite[] = list.items
		.filter((row) => !row.payload.lastSynced || row.payload.queuedForSync)
		.map((row) => ({
			type: "save",
			data: {
				tableName: PERSON_TABLE,
				entityId: row.entityId,
				payload: {
					...row.payload,
					queuedForSync: false,
					lastSynced: now,
					lastSyncError: null,
				},
			},
		}));
	if (operations.length > 0) {
		await client.batchWrites(operations);
	}
}

export async function seedDemoPerson(): Promise<PersonRecord> {
	const existing = await listPersons({ page: 1, pageSize: 1 });
	if (existing.totalItems > 0) return existing.items[0];
	return createPerson({
		name: "Ada",
		surname: "Lovelace",
		age: 36,
		dob: "1815-12-10",
		phoneNumber: "+447000000000",
		photo: "https://avatars.githubusercontent.com/u/583231?v=4",
	});
}
