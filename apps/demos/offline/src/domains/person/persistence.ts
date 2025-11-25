import {
	 type BatchWrite,
	 type EntityRecord,
	 type MetadataSnapshot,
	 type PaginatedResult,
	 PersistenceClient,
	 type SchemaDefinition,
} from "@zengateglobal/persistence-sdk";
import { createOfflineSqliteProvider } from "@zengateglobal/persistence-sdk";

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

function buildMetadataSnapshot(): MetadataSnapshot {
	return {
		tables: new Map([
			[
				PERSON_TABLE,
				{
					tableName: PERSON_TABLE,
					activeVersion: PERSON_SCHEMA_VERSION,
					versions: new Map([[PERSON_SCHEMA_VERSION, PERSON_SCHEMA_DEFINITION]]),
				},
			],
		]),
		fetchedAt: new Date(),
	};
}

const provider = createOfflineSqliteProvider({
	// Shared demo DB within OPFS; adjust per-tenant if needed.
	databaseName: "/offline/persons-demo.db",
	initialMetadata: buildMetadataSnapshot(),
});
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
