import type {
	BatchWrite,
	BatchWriteProgressListener,
	DeleteEntityInput,
	EntityIdentifier,
	EntityRecord,
	PaginatedResult,
	PersistenceProvider,
	QueryOptions,
	SaveEntityInput,
	Schema,
	SchemaIdentifier,
	SyncProgressListener,
	SyncReport,
	SyncRequest,
} from "./types";
import type { JournalEntry } from "./types/journal";

/**
 * Concrete persistence client that orchestrates the configured providers.
 * It exposes high-level operations such as listing providers and syncing data.
 * Provider-specific CRUD operations live on each provider implementation.
 */
export class PersistenceClient implements PersistenceProvider {
	private readonly providers = new Map<string, PersistenceProvider>();
	private activeProvider?: PersistenceProvider;

	constructor(providers: PersistenceProvider[]) {
		if (providers.length === 0) {
			throw new Error("At least one provider must be registered.");
		}
		for (const provider of providers) {
			this.providers.set(provider.name, provider);
		}
		this.activeProvider = providers[0];
	}

	setActiveProvider(providerName: string): void {
		const provider = this.providers.get(providerName);
		if (!provider) {
			throw new Error(`Provider "${providerName}" not found.`);
		}
		this.activeProvider = provider;
	}

	getProviders(): PersistenceProvider[] {
		return Array.from(this.providers.values());
	}

	/**
	 * Planned sync flow (implementation WIP):
	 * 1) If the source provider has journal entries, push them to the target via batchWrites.
	 *    - On failure, abort and surface the error.
	 * 2) Clear the source provider journal after a successful push.
	 * 3) Pull schemas from the target (getMetadata) and apply them to the source (setMetadata).
	 * 4) Clear all entity tables in the source provider (WIP in provider contract).
	 * 5) For each schema/table from the target:
	 *    - queryEntities on the target
	 *    - batchWrites into the source
	 *
	 * Note: This MVP approach is intentionally simple and downloads all data, so it is not optimal for large datasets.
	 * Future iterations will optimize the sync strategy; this version prioritizes correctness and speed of delivery.
	 * Even it could be fully implemented in the backend async.
	 *
	 */
	async sync(request: SyncRequest): Promise<SyncReport> {
		const startedAt = new Date();
		const details: SyncReport["details"] = [];
		let status: SyncReport["status"] = "success";

		const source = this.providers.get(request.sourceProviderId);
		const target = this.providers.get(request.targetProviderId);

		if (!source) {
			throw new Error(
				`Source provider "${request.sourceProviderId}" not found`,
			);
		}
		if (!target) {
			throw new Error(
				`Target provider "${request.targetProviderId}" not found`,
			);
		}

		const emit = (event: Parameters<SyncProgressListener>[0]) =>
			request.onProgress?.(event);

		try {
			// Step 1: push journal from source to target.
			const journal = await source.listJournalEntries();
			emit({ stage: "push:start", journalCount: journal.length });
			if (journal.length > 0) {
				// Drop changeId when sending to batchWrites.
				const operations = journal.map(({ changeId, ...entity }) => entity);
				let lastProgress = 0;
				await target.batchWrites(operations, false, ({ written, total }) => {
					if (written !== lastProgress) {
						lastProgress = written;
						emit({
							stage: "push:progress",
							written,
							total,
						} as any);
					}
				});
			}
			emit({ stage: "push:success", journalCount: journal.length });

			// Step 2: clear source journal.
			await source.clearJournalEntries();
			emit({ stage: "journal:cleared" });

			// Step 3: refresh schemas in source from target.
			const schemas = await target.getMetadata();
			await source.setMetadata(schemas);
			emit({ stage: "schemas:refreshed", schemaCount: schemas.length });

			// Step 4: clear source entity tables (per schema).
			const tableNames = Array.from(
				new Set(schemas.map((schema) => schema.tableName)),
			);
			emit({ stage: "clear:start", tableCount: tableNames.length });
			for (const tableName of tableNames) {
				await source.clear({ tableName });
			}
			emit({ stage: "clear:success", tableCount: tableNames.length });

			// Step 5: pull all data per table from target into source.
			const PAGE_SIZE = 100;
			for (const tableName of tableNames) {
				let entitiesSynced = 0;
				let page = 1;
				let totalPages = 1;
				const tableErrors: string[] = [];
				emit({ stage: "pull:start", tableName, pageSize: PAGE_SIZE });

				do {
					try {
						const pageResult = await target.queryEntities(
							{ tableName },
							{
								pagination: { page, pageSize: PAGE_SIZE },
								includeDeleted: true,
								onlyActive: false,
							},
						);
						totalPages = pageResult.totalPages || 0;
						if (pageResult.items.length > 0) {
							let lastProgress = 0;
							await source.batchWrites(
								pageResult.items,
								false,
								({ written, total }) => {
									if (written !== lastProgress) {
										lastProgress = written;
										emit({
											stage: "pull:progress",
											tableName,
											page,
											totalPages,
											written,
											total,
										} as any);
									}
								},
							);
							entitiesSynced += pageResult.items.length;
						}
						emit({
							stage: "pull:page",
							tableName,
							page,
							totalPages,
							count: pageResult.items.length,
						});
						page += 1;

						// TODO: Remove the success page from the journal.
					} catch (error) {
						console.error(`Sync failed for table: ${tableName}`, error);
						tableErrors.push(
							error instanceof Error ? error.message : String(error),
						);
						status = "partial";
						emit({
							stage: "pull:error",
							tableName,
							error: error instanceof Error ? error.message : String(error),
						});
						break;
					}
				} while (page <= totalPages);

				details.push({
					tableName,
					entitiesSynced,
					errors: tableErrors.length > 0 ? tableErrors : undefined,
				});
			}
		} catch (error) {
			console.error("Sync failed:", error);
			status = "error";
			details.push({
				tableName: "*",
				entitiesSynced: 0,
				errors: [error instanceof Error ? error.message : String(error)],
			});
		}

		return {
			startedAt,
			finishedAt: new Date(),
			status,
			details,
		};
	}

	protected resolveActiveProvider(): PersistenceProvider {
		if (!this.activeProvider) {
			throw new Error("No active provider configured.");
		}
		return this.activeProvider;
	}

	// PersistenceProvider Implementation.

	get name(): string {
		return this.resolveActiveProvider().name;
	}

	get description(): string {
		return this.resolveActiveProvider().description;
	}

	async getMetadata(): Promise<Schema[]> {
		return await this.resolveActiveProvider().getMetadata();
	}

	async setMetadata(snapshot: Schema[]): Promise<void> {
		return await this.resolveActiveProvider().setMetadata(snapshot);
	}

	async batchWrites(
		entities: BatchWrite,
		writeInJournal: boolean = false,
		onProgress?: BatchWriteProgressListener,
	): Promise<void> {
		return await this.resolveActiveProvider().batchWrites(
			entities,
			writeInJournal,
			onProgress,
		);
	}

	async saveEntity<TPayload = unknown>(
		input: SaveEntityInput<TPayload>,
	): Promise<EntityRecord<TPayload>> {
		return await this.resolveActiveProvider().saveEntity(input);
	}

	async getEntity<TPayload = unknown>(
		ref: EntityIdentifier,
	): Promise<EntityRecord<TPayload> | undefined> {
		return await this.resolveActiveProvider().getEntity(ref);
	}

	async deleteEntity(input: DeleteEntityInput): Promise<void> {
		return await this.resolveActiveProvider().deleteEntity(input);
	}

	async queryEntities<TPayload = unknown>(
		tableName: SchemaIdentifier,
		options?: QueryOptions,
	): Promise<PaginatedResult<EntityRecord<TPayload>>> {
		return this.resolveActiveProvider().queryEntities(tableName, options);
	}

	async listJournalEntries(): Promise<JournalEntry[]> {
		return await this.resolveActiveProvider().listJournalEntries();
	}

	async clearJournalEntries(): Promise<void> {
		return await this.resolveActiveProvider().clearJournalEntries();
	}

	async clear(table: SchemaIdentifier): Promise<void> {
		return await this.resolveActiveProvider().clear(table);
	}

	async close(): Promise<void> {
		return await this.resolveActiveProvider().close();
	}
}
