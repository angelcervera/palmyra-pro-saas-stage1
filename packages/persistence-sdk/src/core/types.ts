export type JsonPrimitive = string | number | boolean | null;
export type JsonValue =
  | JsonPrimitive
  | JsonValue[]
  | {
      [key: string]: JsonValue;
    };

export type ProviderKind = 'online' | 'offline';

export interface TableIdentifier {
  tableName: string; // unique per tenant
}

export interface EntityIdentifier extends TableIdentifier {
  entityId: string; // UUID + table name
}

export interface SchemaVersionInfo {
  version: string;
  createdAt: string;
  description?: string;
}

export interface TableMetadata {
  tableName: string;
  title?: string;
  versions: SchemaVersionInfo[];
}

export interface MetadataSnapshot {
  tables: TableMetadata[];
  fetchedAt: string;
}

export interface EntityRecord<TPayload extends JsonValue = JsonValue> {
  id: string;
  tableName: string;
  payload: TPayload;
  schemaVersion: string;
  createdAt: string;
  updatedAt: string;
  deletedAt?: string | null;
}

export interface PaginationQuery {
  page?: number;
  pageSize?: number;
}

export interface PaginatedResult<T> {
  items: T[];
  page: number;
  pageSize: number;
  totalItems: number;
  totalPages: number;
}

export interface SaveEntityInput<TPayload extends JsonValue = JsonValue> extends TableIdentifier {
  entityId?: string;
  payload: TPayload;
  correlationId?: string;
}

export interface DeleteEntityInput extends EntityIdentifier {
  reason?: string;
}

export type BatchWrite =
  | { type: 'save'; data: SaveEntityInput }
  | { type: 'delete'; data: DeleteEntityInput };

export interface BatchWriteResult {
  success: boolean;
  entityId?: string;
  error?: string;
}

export interface ProviderSummary {
  id: string;
  kind: ProviderKind;
  description?: string;
  isDefault?: boolean;
}

export interface SyncRequest {
  sourceProviderId: string;
  targetProviderId: string;
  tables?: string[]; // defaults to all
}

export interface SyncReport {
  startedAt: string;
  finishedAt: string;
  status: 'success' | 'partial' | 'error';
  details: {
    tableName: string;
    entitiesSynced: number;
    errors?: string[];
  }[];
}
