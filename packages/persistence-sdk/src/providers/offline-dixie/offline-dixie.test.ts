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

describe("offline-dixie provider", () => {
	test("creates stores on first instantiation", async () => {
		const metadata = buildMetadata();
		const options = buildOptions(metadata);

		const provider = await createOfflineDixieProvider(options);
		const dexie: Dexie = provider.dexie;
		await dexie.open();

		const storeNames = dexie.tables.map((t) => t.name);
		const expected = expectedStores(metadata);

		expect(storeNames.length).toBe(expected.length);
		expect(storeNames).toEqual(expect.arrayContaining(expected));

		dexie.close();
		await Dexie.delete(dexie.name);
	});

	test("retains stores across subsequent instantiations", async () => {
		const metadata = buildMetadata();
		const options = buildOptions(metadata);
		const expected = expectedStores(metadata);

		const first = await createOfflineDixieProvider(options);
		const firstDexie: Dexie = first.dexie;
		await firstDexie.open();
		await firstDexie.close();

		const second = await createOfflineDixieProvider(options);
		const secondDexie: Dexie = second.dexie;
		await secondDexie.open();

		const storeNames = secondDexie.tables.map((t) => t.name);
		expect(storeNames.length).toBe(expected.length);
		expect(storeNames).toEqual(expect.arrayContaining(expected));

		secondDexie.close();
		await Dexie.delete(secondDexie.name);
	});
});
