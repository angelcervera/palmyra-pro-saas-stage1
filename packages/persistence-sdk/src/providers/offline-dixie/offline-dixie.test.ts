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
});
