import "fake-indexeddb/auto";

import { Dexie } from "dexie";
import { describe, expect, test } from "vitest";

import type { EntityRecord, Schema } from "../../core";
import {
	createOfflineDexieProvider,
	type OfflineDexieProviderOptions,
} from "./index";

const SCHEMAS_STORE = "__schema-metadata";
const JOURNAL_STORE = "__entity-journal";
const deriveActiveTableName = (tableName: string) => `active::${tableName}`;

const buildSchema = (tableName: string, version = "1.0.0"): Schema => ({
	tableName,
	schemaVersion: version,
	schemaDefinition: { type: "object" },
	categoryId: crypto.randomUUID(),
	createdAt: new Date(),
	isDeleted: false,
	isActive: true,
});

const buildOptions = (schemas: Schema[]): OfflineDexieProviderOptions => ({
	envKey: `env-${crypto.randomUUID()}`,
	tenantId: `tenant-${crypto.randomUUID()}`,
	appName: `app-${crypto.randomUUID()}`,
	schemas,
});

const dbName = (options: OfflineDexieProviderOptions) =>
	`${options.envKey}-${options.tenantId}-${options.appName}`;

const readAll = async <T>(
	options: OfflineDexieProviderOptions,
	store: string,
): Promise<T[]> => {
	const db = new Dexie(dbName(options));
	await db.open();
	try {
		return await db.table<T>(store).toArray();
	} finally {
		db.close();
	}
};

const seedSchemas = async (
	options: OfflineDexieProviderOptions,
	schemas: Schema[],
): Promise<void> => {
	const db = new Dexie(dbName(options));
	await db.open();
	try {
		const metadata = db.table<Schema>(SCHEMAS_STORE);
		const active = db.table<Schema>(deriveActiveTableName(SCHEMAS_STORE));
		await metadata.clear();
		await active.clear();
		await metadata.bulkPut(schemas);
		await active.bulkPut(schemas);
	} finally {
		db.close();
	}
};

const expectedStores = (schemas: Schema[]): string[] => {
	const names = [
		SCHEMAS_STORE,
		deriveActiveTableName(SCHEMAS_STORE),
		JOURNAL_STORE,
	];
	for (const schema of schemas) {
		names.push(schema.tableName, deriveActiveTableName(schema.tableName));
	}
	return names;
};

describe("offline-dexie provider", () => {
	test("creates object stores for provided schemas", async () => {
		const schemas = [buildSchema("entities"), buildSchema("orders")];
		const options = buildOptions(schemas);

		const provider = await createOfflineDexieProvider(options);

		const stores = await new Promise<string[]>((resolve, reject) => {
			const openReq = indexedDB.open(dbName(options));
			openReq.onerror = () =>
				reject(openReq.error ?? new Error("failed to open db"));
			openReq.onsuccess = () => {
				const names = Array.from(openReq.result.objectStoreNames);
				openReq.result.close();
				resolve(names);
			};
		});

		expect(stores).toEqual(expect.arrayContaining(expectedStores(schemas)));
		expect(stores.length).toBe(expectedStores(schemas).length);

		await provider.close();
		await Dexie.delete(dbName(options));
	});
});

test("batchWrites persists records and active copies", async () => {
	const schemas = [buildSchema("entities")];
	const options = buildOptions(schemas);
	const provider = await createOfflineDexieProvider(options);

	const entity: EntityRecord = {
		tableName: "entities",
		entityId: crypto.randomUUID(),
		entityVersion: "1.0.0",
		schemaVersion: "1.0.0",
		payload: { foo: "bar" },
		ts: new Date(),
		isDeleted: false,
		isActive: true,
	};

	await provider.batchWrites([entity]);

	const all = await readAll<EntityRecord>(options, "entities");
	const active = await readAll<EntityRecord>(
		options,
		deriveActiveTableName("entities"),
	);

	expect(all).toHaveLength(1);
	expect(all[0].entityId).toBe(entity.entityId);
	expect(active).toHaveLength(1);
	expect(active[0].entityId).toBe(entity.entityId);

	await provider.close();
	await Dexie.delete(dbName(options));
});

test("saveEntity creates and updates with bumped versions", async () => {
	const schemas = [buildSchema("entities", "1.0.0")];
	const options = buildOptions(schemas);
	const provider = await createOfflineDexieProvider(options);
	await seedSchemas(options, schemas);

	const created = await provider.saveEntity({
		tableName: "entities",
		payload: { foo: "bar" },
	});

	expect(created.entityVersion).toBe("1.0.0");
	expect(created.isDeleted).toBe(false);

	const updated = await provider.saveEntity({
		tableName: "entities",
		entityId: created.entityId,
		payload: { foo: "baz" },
	});

	expect(updated.entityVersion).toBe("1.0.1");
	expect(updated.payload).toEqual({ foo: "baz" });

	const active = await provider.getEntity<{ foo: string }>({
		tableName: "entities",
		entityId: created.entityId,
	});
	expect(active?.entityVersion).toBe("1.0.1");
	expect(active?.payload).toEqual({ foo: "baz" });

	const all = await readAll<EntityRecord>(options, "entities");
	expect(all).toHaveLength(1);
	expect(all[0].entityVersion).toBe("1.0.1");

	await provider.close();
	await Dexie.delete(dbName(options));
});

test("deleteEntity soft deletes and bumps version", async () => {
	const schemas = [buildSchema("entities", "1.0.0")];
	const options = buildOptions(schemas);
	const provider = await createOfflineDexieProvider(options);
	await seedSchemas(options, schemas);

	const created = await provider.saveEntity({
		tableName: "entities",
		payload: { foo: "bar" },
	});

	await provider.deleteEntity({
		tableName: "entities",
		entityId: created.entityId,
	});

	const active = await provider.getEntity<{ foo: string }>({
		tableName: "entities",
		entityId: created.entityId,
	});

	expect(active?.isDeleted).toBe(true);
	expect(active?.entityVersion).toBe("1.0.1");

	const all = await readAll<EntityRecord>(options, "entities");
	expect(all).toHaveLength(1);
	expect(all[0].isDeleted).toBe(true);
	expect(all[0].entityVersion).toBe("1.0.1");

	const activeRows = await readAll<EntityRecord>(
		options,
		deriveActiveTableName("entities"),
	);
	expect(activeRows).toHaveLength(1);
	expect(activeRows[0].isDeleted).toBe(true);

	await provider.close();
	await Dexie.delete(dbName(options));
});

test("setMetadata reinitializes stores for new schemas", async () => {
	const initial = [buildSchema("entities")];
	const options = buildOptions(initial);
	const provider = await createOfflineDexieProvider(options);

	const storesBefore = await new Promise<string[]>((resolve, reject) => {
		const req = indexedDB.open(dbName(options));
		req.onerror = () => reject(req.error ?? new Error("failed to open"));
		req.onsuccess = () => {
			const names = Array.from(req.result.objectStoreNames);
			req.result.close();
			resolve(names);
		};
	});
	expect(storesBefore).toEqual(expect.arrayContaining(expectedStores(initial)));

	const nextSchemas = [...initial, buildSchema("orders")];
	await provider.setMetadata(nextSchemas);

	const storesAfter = await new Promise<string[]>((resolve, reject) => {
		const req = indexedDB.open(dbName(options));
		req.onerror = () => reject(req.error ?? new Error("failed to open"));
		req.onsuccess = () => {
			const names = Array.from(req.result.objectStoreNames);
			req.result.close();
			resolve(names);
		};
	});

	expect(storesAfter).toEqual(
		expect.arrayContaining(expectedStores(nextSchemas)),
	);

	await provider.close();
	await Dexie.delete(dbName(options));
});

test("getEntity returns undefined when entity is missing", async () => {
	const schemas = [buildSchema("entities")];
	const options = buildOptions(schemas);
	const provider = await createOfflineDexieProvider(options);

	const result = await provider.getEntity({
		tableName: "entities",
		entityId: crypto.randomUUID(),
	});

	expect(result).toBeUndefined();

	await provider.close();
	await Dexie.delete(dbName(options));
});
