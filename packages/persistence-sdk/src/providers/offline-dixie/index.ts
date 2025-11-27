import {Dexie, type EntityTable} from 'dexie';

import {
    BatchWrite,
    DeleteEntityInput,
    EntityIdentifier,
    EntityRecord,
    JournalEntry,
    MetadataSnapshot,
    PaginatedResult,
    PaginationQuery, PersistenceClient,
    PersistenceProvider,
    SaveEntityInput,
    SchemaIdentifier
} from "../../core";


const DB_VERSION = 1
const SCHEMAS_STORE = "__schema-metadata";
const JOURNAL_STORE = "__entity-journal";

export type OfflineDixieProviderOptions = {
    tenantId: string;
    envKey: string;
    appName: string;
    initialMetadata: MetadataSnapshot;
};

export const createOfflineDixieProvider = async (
    options: OfflineDixieProviderOptions,
): Promise<OfflineDixieProvider> => new OfflineDixieProvider(initDexie(options), options);

// Helpers

const deriveDBName = (envKey: string, tenantId: string, appName: string) => `${envKey}-${tenantId}-${appName}`;

const deriveActiveTableName = (tableName: string) => `active::${tableName}`;

const dixieStoresBuilder = (metadata: MetadataSnapshot) => {

    const stores: { [tableName: string]: string | null } = {};

    // Required.
    stores[SCHEMAS_STORE] = 'tableName, schemaVersion'
    stores[deriveActiveTableName(SCHEMAS_STORE)] = 'tableName'
    stores[JOURNAL_STORE] = '++changeId'


    // Entities
    for (const [tableName] of Object.entries(metadata.tables)) {
        stores[tableName] = 'entityId, entityVersion'
        stores[deriveActiveTableName(tableName)] = 'entityId'
    }

    return stores
}

const initDexie = (options: OfflineDixieProviderOptions): Dexie => {
    const databaseName = deriveDBName(options.envKey, options.tenantId, options.appName)
    const db = new Dexie(databaseName);
    db.version(DB_VERSION).stores(dixieStoresBuilder(options.initialMetadata));
    return db;
}


// Implementation
export class OfflineDixieProvider implements PersistenceProvider {
    readonly name: string = 'Offline Dixie';
    readonly description: string = 'A persistence provider that stores data locally using Dixie';

    private readonly db: Dexie;

    constructor(private readonly dexie: Dexie, private readonly options: OfflineDixieProviderOptions) {
        this.db = dexie;
    }


    getMetadata(): Promise<MetadataSnapshot> {
        throw new Error("Method not implemented.");
    }

    setMetadata(snapshot: MetadataSnapshot): Promise<void> {
        throw new Error("Method not implemented.");
    }

    getEntity<TPayload = unknown>(ref: EntityIdentifier): Promise<EntityRecord<TPayload>> {
        throw new Error("Method not implemented.");
    }

    queryEntities<TPayload = unknown>(scope: SchemaIdentifier, pagination?: PaginationQuery): Promise<PaginatedResult<EntityRecord<TPayload>>> {
        throw new Error("Method not implemented.");
    }

    saveEntity<TPayload = unknown>(input: SaveEntityInput<TPayload>): Promise<EntityRecord<TPayload>> {
        throw new Error("Method not implemented.");
    }

    deleteEntity(input: DeleteEntityInput): Promise<void> {
        throw new Error("Method not implemented.");
    }

    batchWrites(operations: BatchWrite[]): Promise<void> {
        throw new Error("Method not implemented.");
    }

    listJournalEntries(): Promise<JournalEntry[]> {
        throw new Error("Method not implemented.");
    }

    clearJournalEntries(): Promise<void> {
        throw new Error("Method not implemented.");
    }

    close(): Promise<void> {
        throw new Error("Method not implemented.");
    }
}
