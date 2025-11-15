import type {
  BatchWrite,
  BatchWriteResult,
  DeleteEntityInput,
  EntityIdentifier,
  EntityRecord,
  JsonValue,
  MetadataSnapshot,
  PaginatedResult,
  PaginationQuery,
  ProviderSummary,
  SaveEntityInput,
  SyncReport,
  SyncRequest,
  TableIdentifier,
} from './types';

export interface PersistenceClient {
  /**
   * Returns the list of tables (schemas) available for the tenant along with every published version.
   */
  getMetadata(): Promise<MetadataSnapshot>;

  /**
   * Retrieves the latest version of an entity by table name and UUID.
   */
  getEntity<TPayload extends JsonValue = JsonValue>(ref: EntityIdentifier): Promise<EntityRecord<TPayload>>;

  /**
   * Queries a table and returns paginated entities (latest versions only).
   * Filtering/sorting will be added later; only pagination is supported initially.
   */
  queryEntities<TPayload extends JsonValue = JsonValue>(
    scope: TableIdentifier,
    pagination?: PaginationQuery,
  ): Promise<PaginatedResult<EntityRecord<TPayload>>>;

  /**
   * Upserts an entity using the latest schema version for the selected table.
   */
  saveEntity<TPayload extends JsonValue = JsonValue>(input: SaveEntityInput<TPayload>): Promise<EntityRecord<TPayload>>;

  /**
   * Soft-deletes an entity (marks the latest version as deleted).
   */
  deleteEntity(input: DeleteEntityInput): Promise<void>;

  /**
   * Applies multiple save/delete operations in a single call.
   */
  batchWrites(operations: BatchWrite[]): Promise<BatchWriteResult[]>;

  /**
   * Returns the available providers (e.g., online API, offline cache).
   */
  getProviders(): Promise<ProviderSummary[]>;

  /**
   * Synchronizes two providers (e.g., push offline mutations to the online API).
   */
  sync(request: SyncRequest): Promise<SyncReport>;
}
