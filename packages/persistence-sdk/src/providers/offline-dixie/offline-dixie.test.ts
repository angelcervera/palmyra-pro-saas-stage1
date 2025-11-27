import "fake-indexeddb/auto";

import { Dexie } from "dexie";
import { describe, expect, test } from "vitest";

import type { MetadataSnapshot, SchemaDefinition } from "../../core";
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

	const tables =
		metadata.tables instanceof Map
			? metadata.tables
			: new Map(Object.entries(metadata.tables ?? {}));

	for (const [tableName] of tables) {
		names.push(tableName, deriveActiveTableName(tableName));
	}

	return names;
};

const getObjectStoreNames = async (dbName: string): Promise<string[]> => {
	const db = await new Promise<IDBDatabase>((resolve, reject) => {
		const request = indexedDB.open(dbName);
		request.onerror = () =>
			reject(request.error ?? new Error("failed to open db"));
		request.onsuccess = () => resolve(request.result);
	});

	const stores = Array.from(db.objectStoreNames);
	db.close();
	return stores;
};

describe("offline-dixie provider", () => {
	test("creates stores on first instantiation", async () => {
		const metadata = buildMetadata();
		const options = buildOptions(metadata);
		const dbName = `${options.envKey}-${options.tenantId}-${options.appName}`;

		const provider = await createOfflineDixieProvider(options);
		const dexie: Dexie = provider.dexie;
		await dexie.open();

		const storeNames = await getObjectStoreNames(dbName);
		const expected = expectedStores(metadata);

		expect(storeNames.length).toBe(expected.length);
		expect(storeNames).toEqual(expect.arrayContaining(expected));

		await Dexie.delete(dbName);
	});

	test("retains stores across subsequent instantiations", async () => {
		const metadata = buildMetadata();
		const options = buildOptions(metadata);
		const dbName = `${options.envKey}-${options.tenantId}-${options.appName}`;
		const expected = expectedStores(metadata);

		const first = await createOfflineDixieProvider(options);
		const firstDexie: Dexie = first.dexie;
		await firstDexie.open();
		await firstDexie.close();

		const second = await createOfflineDixieProvider(options);
		const secondDexie: Dexie = second.dexie;
		await secondDexie.open();

		const storeNames = await getObjectStoreNames(dbName);
		expect(storeNames.length).toBe(expected.length);
		expect(storeNames).toEqual(expect.arrayContaining(expected));

		await secondDexie.close();
		await Dexie.delete(dbName);
	});
});
