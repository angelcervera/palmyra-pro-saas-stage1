import "fake-indexeddb/auto";
import { describe, expect, test } from "vitest";
import {
	createOfflineIndexedDbProvider,
	type MetadataSnapshot,
	type SchemaDefinition,
} from "../../index";

function buildMetadata(): MetadataSnapshot {
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
		fetchedAt: new Date(),
	};
}

describe("offline-indexeddb provider", () => {
	test("creates, updates, deletes entities and records journal entries", async () => {
		const metadata = buildMetadata();
		const databaseName = `test-idb-${crypto.randomUUID()}`;
		const provider = createOfflineIndexedDbProvider({
			tenantId: "tenantA",
			initialMetadata: metadata,
			databaseName,
		});

		// create entity -> should create store and set version
		const created = await provider.saveEntity({
			tableName: "entities",
			payload: { foo: "bar" },
		});
		expect(created.tableName).toBe("entities");
		expect(created.entityVersion).toBeTruthy();
		expect(created.isDeleted).toBe(false);

		// store exists with tenant prefix
		const storeName = "tenantA::entities";
		const db = await new Promise<IDBDatabase>((resolve, reject) => {
			const open = indexedDB.open(databaseName, undefined);
			open.onsuccess = () => resolve(open.result);
			open.onerror = () => reject(open.error ?? new Error("failed to open db"));
		});
		expect(db.objectStoreNames.contains(storeName)).toBe(true);
		db.close();

		// update entity -> new version, original version untouched (version changes)
		const updated = await provider.saveEntity({
			tableName: "entities",
			entityId: created.entityId,
			payload: { foo: "baz" },
		});
		expect(updated.entityVersion).not.toBe(created.entityVersion);
		expect(updated.payload).toEqual({ foo: "baz" });

		// delete -> new version with isDeleted flag
		const beforeDeleteVersion = updated.entityVersion;
		await provider.deleteEntity({
			tableName: "entities",
			entityId: created.entityId,
		});
		const deleted = await provider.getEntity<{ foo: string }>({
			tableName: "entities",
			entityId: created.entityId,
		});
		expect(deleted.isDeleted).toBe(true);
		expect(deleted.entityVersion).not.toBe(beforeDeleteVersion);

		// second table to ensure multiple entities work
		const order = await provider.saveEntity({
			tableName: "orders",
			payload: { foo: "order-1" },
		});
		expect(order.tableName).toBe("orders");

		// journal contains all changes in order
		const journal = await provider.listJournalEntries();
		expect(journal.map((j) => j.changeType)).toEqual([
			"create",
			"update",
			"delete",
			"create",
		]);
		expect(journal.map((j) => j.tableName)).toEqual([
			"entities",
			"entities",
			"entities",
			"orders",
		]);
	});
});
