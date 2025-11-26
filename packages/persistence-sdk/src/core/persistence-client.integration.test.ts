import "fake-indexeddb/auto";
import { describe, expect, test } from "vitest";
import {
	createOfflineIndexedDbProvider,
	type MetadataSnapshot,
	PersistenceClient,
	type PersistenceProvider,
	type SchemaDefinition,
} from "../index";

// ProviderHarness lets us plug multiple PersistenceProvider impls into the same
// integration flow. Add new providers by pushing to `providers` with a build()
// that returns the provider plus optional cleanup/journal verifiers.
type ProviderHarness = {
	name: string;
	build: () => Promise<{
		provider: PersistenceProvider;
		cleanup?: () => Promise<void>;
	}>;
};

const buildMetadata = (): MetadataSnapshot => {
	const farmerDefinitionV1: SchemaDefinition = {
		type: "object",
		properties: {
			name: { type: "string" },
			crop: { type: "string" },
		},
		required: ["name"],
	};
	const farmerDefinitionV2: SchemaDefinition = {
		type: "object",
		properties: {
			name: { type: "string" },
			crop: { type: "string" },
			farmSize: { type: "number" },
		},
		required: ["name"],
	};
	const orderDefinition: SchemaDefinition = {
		type: "object",
		properties: { foo: { type: "string" }, total: { type: "number" } },
		required: ["foo"],
	};
	return {
		tables: new Map([
			[
				"farmers",
				{
					tableName: "farmers",
					activeVersion: "1.0.1",
					versions: new Map([
						["1.0.0", farmerDefinitionV1],
						["1.0.1", farmerDefinitionV2],
					]),
				},
			],
			[
				"orders",
				{
					tableName: "orders",
					activeVersion: "1.0.0",
					versions: new Map([["1.0.0", orderDefinition]]),
				},
			],
		]),
		fetchedAt: new Date(),
	};
};

const providers: ProviderHarness[] = [
	{
		name: "offline-indexeddb",
		build: async () => {
			const metadata = buildMetadata();
			const provider = createOfflineIndexedDbProvider({
				tenantId: "tenant-test",
				initialMetadata: metadata,
				databaseName: `test-idb-${crypto.randomUUID()}`,
			});
			return {
				provider,
			};
		},
	},
];

describe.each(providers)("%s PersistenceClient integration", (harness) => {
	test("performs CRUD, preserves versions, and delegates to active provider", async () => {
		const { provider, cleanup } = await harness.build();
		const client = new PersistenceClient([provider]);

		try {
			// create
			const created = await client.saveEntity({
				tableName: "farmers",
				payload: { foo: "bar" },
			});
			expect(created.entityVersion).toBeTruthy();
			expect(created.isDeleted).toBe(false);

			// query (before delete) should list one active entity
			const pageBeforeDelete = await client.queryEntities({
				tableName: "farmers",
			});
			expect(pageBeforeDelete.totalItems).toBe(1);
			expect(pageBeforeDelete.items[0]?.entityId).toBe(created.entityId);

			// update -> new version
			const updated = await client.saveEntity({
				tableName: "farmers",
				entityId: created.entityId,
				payload: { foo: "baz" },
			});
			expect(updated.entityVersion).not.toBe(created.entityVersion);

			// delete -> new version marked deleted
			await client.deleteEntity({
				tableName: "farmers",
				entityId: created.entityId,
			});
			const deleted = await client.getEntity<{ foo: string }>({
				tableName: "farmers",
				entityId: created.entityId,
			});
			expect(deleted.isDeleted).toBe(true);
			expect(deleted.entityVersion).not.toBe(updated.entityVersion);

			// second table entity
			const order = await client.saveEntity({
				tableName: "orders",
				payload: { foo: "order-1" },
			});
			expect(order.tableName).toBe("orders");

			// active listing now excludes deleted entity
			const pageAfterDelete = await client.queryEntities({
				tableName: "farmers",
			});
			expect(pageAfterDelete.totalItems).toBe(0);

			const entries = await client.listJournalEntries();
			expect(entries.map((j) => j.changeType)).toEqual([
				"create",
				"update",
				"delete",
				"create",
			]);
			expect(entries.map((j) => j.tableName)).toEqual([
				"farmers",
				"farmers",
				"farmers",
				"orders",
			]);
		} finally {
			await provider.close();
			if (cleanup) {
				await cleanup();
			}
		}
	});
});
