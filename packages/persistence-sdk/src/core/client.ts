import type {
	BatchWrite,
	DeleteEntityInput,
	EntityIdentifier,
	EntityRecord,
	PaginatedResult,
	PersistenceProvider,
	QueryOptions,
	SaveEntityInput,
	Schema,
	SchemaIdentifier,
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
	 */
	async sync(_request: SyncRequest): Promise<SyncReport> {
		throw new Error("Not implemented");
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
	): Promise<void> {
		return await this.resolveActiveProvider().batchWrites(
			entities,
			writeInJournal,
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

	async close(): Promise<void> {
		return await this.resolveActiveProvider().close();
	}
}
