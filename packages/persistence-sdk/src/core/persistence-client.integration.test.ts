import { describe, expect, test } from "vitest";
import { PersistenceClient } from "./client";
import type {
	BatchWrite,
	DeleteEntityInput,
	EntityIdentifier,
	EntityRecord,
	JournalEntry,
	PaginatedResult,
	PersistenceProvider,
	QueryOptions,
	SaveEntityInput,
	Schema,
	SchemaIdentifier,
} from "./types";

const now = () => new Date();

const sampleSchema = (tableName: string): Schema => ({
	tableName,
	schemaVersion: "1.0.0",
	schemaDefinition: { type: "object" },
	categoryId: "cat",
	createdAt: now(),
	isDeleted: false,
	isActive: true,
});

class InMemoryProvider implements PersistenceProvider {
	readonly name: string;
	readonly description = "in-memory test provider";

	private metadata: Schema[] = [];
	private journal: JournalEntry[] = [];
	private changeId = 1;
	private store = new Map<
		string,
		Map<
			string,
			{
				version: number;
				record: EntityRecord;
			}
		>
	>();

	constructor(name: string, initialSchemas: Schema[]) {
		this.name = name;
		this.metadata = initialSchemas;
	}

	async getMetadata(): Promise<Schema[]> {
		return this.metadata;
	}

	async setMetadata(snapshot: Schema[]): Promise<void> {
		this.metadata = snapshot;
	}

	async batchWrites(
		operations: BatchWrite,
		writeInJournal = false,
	): Promise<void> {
		for (const op of operations) {
			this.applyWrite(op);
			if (writeInJournal) {
				this.journal.push(this.toJournalEntry(op));
			}
		}
	}

	async saveEntity<TPayload = unknown>(
		input: SaveEntityInput<TPayload>,
	): Promise<EntityRecord<TPayload>> {
		const table = this.ensureTable(input.tableName);
		const entityId = input.entityId ?? crypto.randomUUID();
		const current = table.get(entityId);
		const nextVersion = current ? current.version + 1 : 1;
		const record: EntityRecord<TPayload> = {
			tableName: input.tableName,
			entityId,
			entityVersion: `${nextVersion}`,
			schemaVersion: "1.0.0",
			payload: input.payload,
			createdAt: now(),
			isDeleted: false,
			isActive: true,
		};
		if (current) {
			current.record.isActive = false;
			table.set(entityId, { version: current.version, record: current.record });
		}
		table.set(entityId, { version: nextVersion, record });
		this.journal.push(this.toJournalEntry(record));
		return record;
	}

	async getEntity<TPayload = unknown>(
		ref: EntityIdentifier,
	): Promise<EntityRecord<TPayload> | undefined> {
		const table = this.store.get(ref.tableName);
		const entry = table?.get(ref.entityId);
		return entry?.record as EntityRecord<TPayload> | undefined;
	}

	async queryEntities<TPayload = unknown>(
		scope: SchemaIdentifier,
		options?: QueryOptions,
	): Promise<PaginatedResult<EntityRecord<TPayload>>> {
		const table = this.store.get(scope.tableName);
		const page =
			options?.pagination?.page && options.pagination.page > 0
				? options.pagination.page
				: 1;
		const pageSize =
			options?.pagination?.pageSize && options.pagination.pageSize > 0
				? options.pagination.pageSize
				: 20;

		if (!table) {
			return {
				items: [],
				page,
				pageSize,
				totalItems: 0,
				totalPages: 0,
			};
		}

		const records = Array.from(table.values()).map(
			(entry) => entry.record as EntityRecord<TPayload>,
		);
		const totalItems = records.length;
		const offset = (page - 1) * pageSize;
		const items = records.slice(offset, offset + pageSize);
		const totalPages = totalItems === 0 ? 0 : Math.ceil(totalItems / pageSize);

		return { items, page, pageSize, totalItems, totalPages };
	}

	async deleteEntity(input: DeleteEntityInput): Promise<void> {
		const table = this.ensureTable(input.tableName);
		const current = table.get(input.entityId);
		if (!current) {
			throw new Error("not found");
		}
		current.record.isActive = false;
		table.set(input.entityId, {
			version: current.version,
			record: current.record,
		});
		const nextVersion = current.version + 1;
		const deleted: EntityRecord = {
			tableName: input.tableName,
			entityId: input.entityId,
			entityVersion: `${nextVersion}`,
			schemaVersion: current.record.schemaVersion,
			payload: current.record.payload,
			createdAt: now(),
			isDeleted: true,
			isActive: true,
		};
		table.set(input.entityId, { version: nextVersion, record: deleted });
		this.journal.push(this.toJournalEntry(deleted));
	}

	async listJournalEntries(): Promise<JournalEntry[]> {
		return this.journal;
	}

	async clearJournalEntries(): Promise<void> {
		this.journal = [];
	}

	async clear(table: SchemaIdentifier): Promise<void> {
		this.store.delete(table.tableName);
	}

	async close(): Promise<void> {
		this.store.clear();
		this.journal = [];
	}

	// Helpers
	private ensureTable(tableName: string) {
		let table = this.store.get(tableName);
		if (!table) {
			table = new Map();
			this.store.set(tableName, table);
		}
		return table;
	}

	private applyWrite(op: EntityRecord): void {
		const table = this.ensureTable(op.tableName);
		table.set(op.entityId, { version: Number(op.entityVersion), record: op });
	}

	private toJournalEntry(record: EntityRecord): JournalEntry {
		return {
			...record,
			changeId: this.changeId++,
		};
	}
}

describe("PersistenceClient with InMemoryProvider", () => {
	test("delegates to active provider and supports CRUD + journal + metadata", async () => {
		const providerA = new InMemoryProvider("provA", [sampleSchema("foo")]);
		const providerB = new InMemoryProvider("provB", [sampleSchema("bar")]);
		const client = new PersistenceClient([providerA, providerB]);

		// metadata
		expect((await client.getMetadata()).map((s) => s.tableName)).toEqual([
			"foo",
		]);
		await client.setMetadata([sampleSchema("foo"), sampleSchema("baz")]);
		expect((await client.getMetadata()).length).toBe(2);

		// CRUD
		const created = await client.saveEntity({
			tableName: "foo",
			payload: { value: 1 },
		});
		const fetched = await client.getEntity({
			tableName: "foo",
			entityId: created.entityId,
		});
		expect(fetched?.entityVersion).toBe("1");
		expect(fetched?.isDeleted).toBe(false);

		const updated = await client.saveEntity({
			tableName: "foo",
			entityId: created.entityId,
			payload: { value: 2 },
		});
		expect(updated.entityVersion).toBe("2");

		await client.deleteEntity({ tableName: "foo", entityId: created.entityId });
		const deleted = await client.getEntity({
			tableName: "foo",
			entityId: created.entityId,
		});
		expect(deleted?.isDeleted).toBe(true);

		// journal + clear
		const journal = await client.listJournalEntries();
		expect(journal).toHaveLength(3);
		await client.clearJournalEntries();
		expect(await client.listJournalEntries()).toHaveLength(0);

		// batchWrites with journal flag
		const batch: BatchWrite = [
			{
				tableName: "foo",
				entityId: "b1",
				entityVersion: "1",
				schemaVersion: "1.0.0",
				payload: { value: 3 },
				createdAt: now(),
				isDeleted: false,
				isActive: true,
			},
		];
		await client.batchWrites(batch, true);
		expect((await client.listJournalEntries()).length).toBe(1);

		// switch provider
		client.setActiveProvider("provB");
		const createdB = await client.saveEntity({
			tableName: "bar",
			payload: { value: 10 },
		});
		expect(createdB.tableName).toBe("bar");
	});
});
