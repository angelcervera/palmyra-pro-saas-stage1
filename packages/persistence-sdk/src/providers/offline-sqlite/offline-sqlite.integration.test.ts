import { describe, expect, test } from "vitest";
import {
	createOfflineSqliteProvider,
	type MetadataSnapshot,
	type SchemaDefinition,
} from "../../index";
import { createNodeSqlitePromiser } from "./node-sqlite-promiser";

function buildMetadata(): MetadataSnapshot {
	const definition: SchemaDefinition = {
		type: "object",
		properties: {
			foo: { type: "string" },
		},
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
		]),
		fetchedAt: new Date(),
	};
}

describe("offline sqlite provider (node sqlite-wasm)", () => {
	test("persists entities across provider instances", async () => {
		const metadata = buildMetadata();
		const promiser = await createNodeSqlitePromiser();
		const promiserFactory = async () => promiser;

		const provider = createOfflineSqliteProvider({
			databaseName: "/integration-proof.db",
			promiserFactory,
		});
		await provider.replaceMetadata(metadata);

		const saved = await provider.saveEntity({
			tableName: "entities",
			payload: { foo: "bar" },
		});

		const fetched = await provider.getEntity({
			tableName: "entities",
			entityId: saved.entityId,
		});
		expect(fetched.payload).toEqual({ foo: "bar" });

		const page = await provider.queryEntities({ tableName: "entities" });
		expect(page.items.length).toBe(1);
		await provider.close();

		const reopened = createOfflineSqliteProvider({
			databaseName: "/integration-proof.db",
			promiserFactory,
		});
		await reopened.replaceMetadata(metadata);
		const persisted = await reopened.getEntity({
			tableName: "entities",
			entityId: saved.entityId,
		});
		expect(persisted.payload).toEqual({ foo: "bar" });

		await reopened.close();
	}, 10000);
});
