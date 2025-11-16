import type {
	BatchWrite,
	DeleteEntityInput,
	EntityIdentifier,
	EntityRecord,
	MetadataSnapshot,
	PaginatedResult,
	PaginationQuery,
	PersistenceProvider,
	SaveEntityInput,
	SchemaIdentifier,
	SyncReport,
	SyncRequest,
} from "./types";

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

	async getMetadata(): Promise<MetadataSnapshot> {
		return this.resolveActiveProvider().getMetadata();
	}

	async getEntity<TPayload = unknown>(
		ref: EntityIdentifier,
	): Promise<EntityRecord<TPayload>> {
		return this.resolveActiveProvider().getEntity(ref);
	}

	async queryEntities<TPayload = unknown>(
		scope: SchemaIdentifier,
		pagination?: PaginationQuery,
	): Promise<PaginatedResult<EntityRecord<TPayload>>> {
		return this.resolveActiveProvider().queryEntities(scope, pagination);
	}

	async saveEntity<TPayload = unknown>(
		input: SaveEntityInput<TPayload>,
	): Promise<EntityRecord<TPayload>> {
		return this.resolveActiveProvider().saveEntity(input);
	}

	async deleteEntity(input: DeleteEntityInput): Promise<void> {
		return this.resolveActiveProvider().deleteEntity(input);
	}

	async batchWrites(operations: BatchWrite[]): Promise<void> {
		return this.resolveActiveProvider().batchWrites(operations);
	}
}
