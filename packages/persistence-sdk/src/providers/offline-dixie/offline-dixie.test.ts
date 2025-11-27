import "fake-indexeddb/auto";

import { Dexie } from "dexie";
import { describe, expect, test } from "vitest";

import type {
	EntityRecord,
	MetadataSnapshot,
	SchemaDefinition,
} from "../../core";
import {
	createOfflineDixieProvider,
	type OfflineDixieProviderOptions,
} from "./index";

const SCHEMAS_STORE = "__schema-metadata";
const JOURNAL_STORE = "__entity-journal";
const deriveActiveTableName = (tableName: string) => `active::${tableName}`;

const buildMetadata = (): MetadataSnapshot => {
	const definition: SchemaDefinition = {
		type: "object",
		properties: { foo: { type: "string" } },
		required: ["foo"],
	};

	return {
		tables: new Map([
			[
				"entities",
				{
					tableName: "entities",
					activeVersion: "v1",
					versions: new Map([["v1", definition]]),
				},
			],
			[
				"orders",
				{
					tableName: "orders",
					activeVersion: "v1",
					versions: new Map([["v1", definition]]),
				},
			],
		]),
	};
};

const buildOptions = (
	metadata: MetadataSnapshot,
): OfflineDixieProviderOptions => {
	const envKey = `env-${crypto.randomUUID()}`;
	const tenantId = `tenant-${crypto.randomUUID()}`;
	const appName = `app-${crypto.randomUUID()}`;

	return {
		envKey,
		tenantId,
		appName,
		initialMetadata: metadata,
	};
};

const expectedStores = (metadata: MetadataSnapshot): string[] => {
	const names = [
		SCHEMAS_STORE,
		deriveActiveTableName(SCHEMAS_STORE),
		JOURNAL_STORE,
	];

	for (const [tableName] of metadata.tables) {
		names.push(tableName, deriveActiveTableName(tableName));
	}

	return names;
};

const toDbName = (options: OfflineDixieProviderOptions): string =>
	`${options.envKey}-${options.tenantId}-${options.appName}`;

const getObjectStoreNames = async (
	options: OfflineDixieProviderOptions,
): Promise<string[]> =>
	new Promise((resolve, reject) => {
		const request = indexedDB.open(toDbName(options));
		request.onerror = () =>
			reject(request.error ?? new Error("failed to open"));
		request.onsuccess = () => {
			const stores = Array.from(request.result.objectStoreNames);
			request.result.close();
			resolve(stores);
		};
	});

const readAllFromStore = async (
	options: OfflineDixieProviderOptions,
	storeName: string,
): Promise<unknown[]> =>
	new Promise((resolve, reject) => {
		const request = indexedDB.open(toDbName(options));
		request.onerror = () =>
			reject(request.error ?? new Error("failed to open"));
		request.onsuccess = () => {
			const db = request.result;
			const tx = db.transaction(storeName, "readonly");
			const store = tx.objectStore(storeName);
			const getAll = store.getAll();
			getAll.onerror = () => reject(getAll.error ?? new Error("getAll failed"));
			getAll.onsuccess = () => {
				db.close();
				resolve(getAll.result as unknown[]);
			};
		};
	});

const buildEntity = (overrides?: Partial<EntityRecord>): EntityRecord => ({
	tableName: "entities",
	entityId: crypto.randomUUID(),
	entityVersion: "v1",
	schemaVersion: "v1",
	payload: { foo: "bar" },
	ts: new Date(),
	isDeleted: false,
	isActive: true,
	...overrides,
});

describe("offline-dixie provider", () => {
	test("creates stores on first instantiation", async () => {
		const metadata = buildMetadata();
		const options = buildOptions(metadata);

		const provider = await createOfflineDixieProvider(options);

		const storeNames = await getObjectStoreNames(provider.options);
		const expected = expectedStores(metadata);

		expect(storeNames.length).toBe(expected.length);
		expect(storeNames).toEqual(expect.arrayContaining(expected));

		await provider.close();
		await Dexie.delete(toDbName(provider.options));
	});

	test("retains stores across subsequent instantiations", async () => {
		const metadata = buildMetadata();
		const options = buildOptions(metadata);
		const expected = expectedStores(metadata);

		const first = await createOfflineDixieProvider(options);
		await first.close();

		const second = await createOfflineDixieProvider(options);

		const storeNames = await getObjectStoreNames(second.options);
		expect(storeNames.length).toBe(expected.length);
		expect(storeNames).toEqual(expect.arrayContaining(expected));

		await second.close();
		await Dexie.delete(toDbName(second.options));
	});

	test("batchWrites saves entity and updates active store", async () => {
		const metadata = buildMetadata();
		const options = buildOptions(metadata);
		const provider = await createOfflineDixieProvider(options);

		const entity = buildEntity();
		await provider.batchWrites([entity]);

		const rows = await readAllFromStore(options, "entities");
		const activeRows = await readAllFromStore(options, "active::entities");

		expect(rows).toHaveLength(1);
		expect((rows[0] as EntityRecord).entityId).toBe(entity.entityId);
		expect((rows[0] as EntityRecord).isDeleted).toBe(false);
		expect((rows[0] as EntityRecord).payload).toEqual({ foo: "bar" });

		expect(activeRows).toHaveLength(1);
		expect((activeRows[0] as EntityRecord).entityId).toBe(entity.entityId);

		await provider.close();
		await Dexie.delete(toDbName(options));
	});

	test("batchWrites delete marks tombstone and active store", async () => {
		const metadata = buildMetadata();
		const options = buildOptions(metadata);
		const provider = await createOfflineDixieProvider(options);

		const entity = buildEntity();
		const tombstone = buildEntity({
			entityId: entity.entityId,
			entityVersion: "v2",
			isDeleted: true,
			isActive: true,
		});
		await provider.batchWrites([entity, tombstone]);

		const rows = await readAllFromStore(options, "entities");
		const activeRows = await readAllFromStore(options, "active::entities");

		expect(rows).toHaveLength(1);
		expect((rows[0] as EntityRecord).isDeleted).toBe(true);
		expect((rows[0] as EntityRecord).entityId).toBe(entity.entityId);

		expect(activeRows).toHaveLength(1);
		expect((activeRows[0] as EntityRecord).isDeleted).toBe(true);

		await provider.close();
		await Dexie.delete(toDbName(options));
	});
});
