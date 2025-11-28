import type {
	BatchWrite,
	DeleteEntityInput,
	EntityIdentifier,
	EntityRecord,
	PersistenceProvider,
	SaveEntityInput,
	Schema,
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

	async batchWrites(entities: BatchWrite): Promise<void> {
		return await this.resolveActiveProvider().batchWrites(entities);
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

	// async queryEntities<TPayload = unknown>(
	// 	scope: SchemaIdentifier,
	// 	pagination?: PaginationQuery,
	// ): Promise<PaginatedResult<EntityRecord<TPayload>>> {
	// 	return this.resolveActiveProvider().queryEntities(scope, pagination);

	// }

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
