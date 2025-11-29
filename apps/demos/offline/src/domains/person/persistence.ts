// Demo wiring for the persistence-sdk using the offline Dexie provider.
// Keep everything in one file so readers can copy/paste into their apps.
import {
	createOfflineDexieProvider,
	type EntityRecord,
	type PaginatedResult,
	PersistenceClient,
	type Schema,
	type SchemaDefinition,
} from "@zengateglobal/persistence-sdk";
import { pushToast } from "../../components/toast";

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

const PERSON_SCHEMA: Schema = {
	tableName: PERSON_TABLE,
	schemaVersion: PERSON_SCHEMA_VERSION,
	schemaDefinition: PERSON_SCHEMA_DEFINITION,
	categoryId: PERSON_SCHEMA_ID,
	createdAt: new Date(),
	isDeleted: false,
	isActive: true,
};

async function createDexieClient(): Promise<PersistenceClient> {
	const provider = await createOfflineDexieProvider({
		envKey: "demo",
		tenantId: "demo-tenant",
		appName: "offline-demo",
		schemas: [PERSON_SCHEMA],
	});
	return new PersistenceClient([provider]);
}

const clientPromise = createDexieClient();

// Wrap calls so we can show clean, user-facing errors in the UI instead of noisy stack traces.
async function runWithClient<T>(
	opLabel: string,
	fn: (c: PersistenceClient) => Promise<T>,
): Promise<T> {
	try {
		const client = await clientPromise;
		return await fn(client);
	} catch (error) {
		const message = `${opLabel} failed: ${describeError(error)}`;
		pushToast({ kind: "error", title: opLabel, description: message });
		throw new Error(message);
	}
}

// Helpers below mirror a tiny repository layer the UI can call directly.
function unwrap(row: EntityRecord<PersonRecord>): PersonRecord {
	return {
		...row.payload,
		// Ensure caller sees the canonical metadata from the stored record.
		entityId: row.entityId,
		entityVersion: row.entityVersion,
		entitySchemaVersion: row.schemaVersion,
	};
}

export async function listPersons(options: {
	page?: number;
	pageSize?: number;
	queuedOnly?: boolean;
}): Promise<PaginatedResult<PersonRecord>> {
	// List locally cached persons; optionally show only items still queued for sync.
	const page = Math.max(options.page ?? 1, 1);
	const pageSize = Math.max(options.pageSize ?? 10, 1);
	const result = await runWithClient("List persons", (c) =>
		c.queryEntities<PersonRecord>(
			{ tableName: PERSON_TABLE },
			{ pagination: { page: 1, pageSize: 1000 }, onlyActive: true },
		),
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
	// Fetch a single person by ID from the offline store.
	const row = await runWithClient("Load person", (c) =>
		c.getEntity<PersonRecord>({
			tableName: PERSON_TABLE,
			entityId,
		}),
	);
	return unwrap(row);
}

export async function createPerson(input: Person): Promise<PersonRecord> {
	// Create a new person locally; marked as queued for the next sync.
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
	const row = await runWithClient("Create person", (c) =>
		c.saveEntity<PersonRecord>({
			tableName: PERSON_TABLE,
			payload,
		}),
	);
	return unwrap(row);
}

export async function updatePerson(
	entityId: string,
	input: Person,
): Promise<PersonRecord> {
	// Update an existing person; keep them queued for sync until the next push.
	const existing = await getPerson(entityId);
	const payload: PersonRecord = {
		...existing,
		queuedForSync: true,
		lastSyncError: null,
		entity: input,
	};
	const row = await runWithClient("Update person", (c) =>
		c.saveEntity<PersonRecord>({
			tableName: PERSON_TABLE,
			entityId,
			payload,
		}),
	);
	return unwrap(row);
}

export async function deletePerson(entityId: string): Promise<void> {
	// Soft-delete the person locally.
	await runWithClient("Delete person", (c) =>
		c.deleteEntity({ tableName: PERSON_TABLE, entityId }),
	);
}

function describeError(error: unknown): string {
	if (error instanceof Error) return error.message;
	if (typeof error === "string") return error;
	try {
		return JSON.stringify(error);
	} catch {
		return "Unknown error";
	}
}
