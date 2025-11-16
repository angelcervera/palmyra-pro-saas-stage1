import type {BatchWrite, PersistenceProvider, SyncReport, SyncRequest} from './types';

/**
 * Concrete persistence client that orchestrates the configured providers.
 * It exposes high-level operations such as listing providers and syncing data.
 * Provider-specific CRUD operations live on each provider implementation.
 */
export class PersistenceClient implements PersistenceProvider {
    private readonly providers = new Map<string, PersistenceProvider>();
    private activeProvider?: PersistenceProvider;

    constructor(providers: PersistenceProvider[]) {
        for (const provider of providers) {
            this.providers.set(provider.name, provider);
        }
    }

    /**
     * Returns the descriptors of all configured providers.
     */
    getProviders(): PersistenceProvider[] {
        return Array.from(this.providers.values());
    }

    /**
     * Synchronizes data between two providers.
     * Currently unimplemented; concrete logic will wire online/offline providers.
     */
    async sync(_request: SyncRequest): Promise<SyncReport> {
        throw new Error('Not implemented');
    }

    /**
     * Internal helper for future implementations.
     */
    protected getProviderById(id: string): PersistenceProvider | undefined {
        return this.providers.get(id);
    }

}
