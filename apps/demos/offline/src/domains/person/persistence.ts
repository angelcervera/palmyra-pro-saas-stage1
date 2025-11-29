import {
	type EntityRecord,
	type PaginatedResult,
	type Schema,
} from "@zengateglobal/persistence-sdk";

import { runWithClient, setDefaultSchemas } from "../persistence/client";

export type Person = {
	name: string;
	surname: string;
	age: number;
	dob: string;
	phoneNumber: string;
	photo: string;
};

export type PersonRecord = EntityRecord<Person>;

const PERSON_TABLE = "persons";
const PERSON_SCHEMA: Schema = {
	tableName: PERSON_TABLE,
	schemaVersion: "1.0.0",
	schemaDefinition: {
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
	},
	categoryId: "00000000-0000-4000-8000-000000000001",
	createdAt: new Date(),
	isDeleted: false,
	isActive: true,
};
setDefaultSchemas([PERSON_SCHEMA]);

export async function listPersons(options: {
	page?: number;
	pageSize?: number;
}): Promise<PaginatedResult<PersonRecord>> {
	const page = Math.max(options.page ?? 1, 1);
	const pageSize = Math.max(options.pageSize ?? 10, 1);
	const result = await runWithClient<PaginatedResult<EntityRecord<Person>>>(
		"List persons",
		(c) =>
			c.queryEntities<Person>(
				{ tableName: PERSON_TABLE },
				{ pagination: { page: 1, pageSize: 1000 }, onlyActive: true },
			),
	);
	const rows = (result.items ?? []) as PersonRecord[];
	const totalItems = rows.length;
	const totalPages = Math.max(Math.ceil(totalItems / pageSize), 1);
	const start = (page - 1) * pageSize;
	const items = rows.slice(start, start + pageSize);
	return { items, page, pageSize, totalItems, totalPages };
}

export async function getPerson(entityId: string): Promise<PersonRecord> {
	const row = await runWithClient<EntityRecord<Person> | undefined>(
		"Load person",
		(c) =>
		c.getEntity<Person>({
			tableName: PERSON_TABLE,
			entityId,
		}),
	);
	if (!row) {
		throw new Error(`Person ${entityId} not found`);
	}
	return row as PersonRecord;
}

export async function createPerson(input: Person): Promise<PersonRecord> {
	const row = await runWithClient<EntityRecord<Person>>("Create person", (c) =>
		c.saveEntity<Person>({
			tableName: PERSON_TABLE,
			payload: input,
		}),
	);
	return row as PersonRecord;
}

export async function updatePerson(
	entityId: string,
	input: Person,
): Promise<PersonRecord> {
	const row = await runWithClient<EntityRecord<Person>>("Update person", (c) =>
		c.saveEntity<Person>({
			tableName: PERSON_TABLE,
			entityId,
			payload: input,
		}),
	);
	return row as PersonRecord;
}

export async function deletePerson(entityId: string): Promise<void> {
	await runWithClient("Delete person", (c) =>
		c.deleteEntity({ tableName: PERSON_TABLE, entityId }),
	);
}
