import {
	createEntitiesClient,
	createSchemaRepositoryClient,
	type Entities,
	type EntitiesClient,
	type SchemaRepository,
	type SchemaRepositoryClient,
} from "@zengateglobal/api-sdk";
import type {
	BatchWrite,
	BatchWriteProgressListener,
	DeleteEntityInput,
	EntityIdentifier,
	EntityRecord,
	JournalEntry,
	PaginatedResult,
	PaginationQuery,
	PersistenceProvider,
	QueryOptions,
	SaveEntityInput,
	Schema,
	SchemaIdentifier,
} from "../../core";
import { wrapProviderError } from "../../shared/errors";
import { fromWireJson, type JsonValue, toJsonObject } from "../../shared/json";

const BEARER_SECURITY = Object.freeze([
	{
		in: "header" as const,
		name: "Authorization",
		scheme: "bearer" as const,
		type: "http" as const,
	},
]);

export type TokenSupplier = () =>
	| Promise<string | undefined>
	| string
	| undefined;

export interface OnlineApiSdkProviderOptions {
	/**
	 * Base URL for the API server (e.g. https://api.example.com/api/v1).
	 * Defaults to the generated client's /api/v1 relative path.
	 */
	readonly baseUrl?: string;
	/**
	 * Optional custom fetch implementation (SSR/testing override).
	 */
	readonly fetch?: typeof fetch;
	/**
	 * Supplies the JWT used for bearer authentication.
	 */
	readonly getToken?: TokenSupplier;
}

export function createOnlineApiSdkProvider(
	options: OnlineApiSdkProviderOptions = {},
): PersistenceProvider {
	return new OnlineApiSdkProvider(options);
}

class OnlineApiSdkProvider implements PersistenceProvider {
	readonly name = "online-apisdk";
	readonly description =
		"Online provider backed by @zengateglobal/api-sdk and the persistence API";

	private readonly entitiesClient: EntitiesClient;
	private readonly schemaRepositoryClient: SchemaRepositoryClient;
	private readonly tokenSupplier?: TokenSupplier;

	constructor(options: OnlineApiSdkProviderOptions) {
		this.tokenSupplier = options.getToken;
		const sharedConfig = {
			baseUrl: options.baseUrl,
			fetch: options.fetch,
			responseStyle: "data" as const,
			throwOnError: true as const,
			auth: async () => this.resolveToken(),
		};
		this.entitiesClient = createEntitiesClient(sharedConfig);
		this.schemaRepositoryClient = createSchemaRepositoryClient(sharedConfig);
	}

	async getMetadata(): Promise<Schema[]> {
		try {
			const response = await this.schemaRepositoryClient.get<
				SchemaRepository.ListAllSchemaVersionsResponses,
				SchemaRepository.ListAllSchemaVersionsErrors,
				true,
				"data"
			>({
				url: "/schema-repository/schemas",
				query: { includeInactive: true },
				security: BEARER_SECURITY,
			});

			return response.items.map((item) => ({
				tableName: item.tableName,
				schemaVersion: item.schemaVersion,
				schemaDefinition: item.schemaDefinition,
				categoryId: item.categoryId,
				createdAt: new Date((item as any).createdAt ?? 0), // FIXME: Why this is a string and not a Date?
				isDeleted: item.isDeleted,
				isActive: item.isActive,
			}));
		} catch (error) {
			throw wrapProviderError("Failed to load schema metadata", error);
		}
	}

	async setMetadata(_snapshot: Schema[]): Promise<void> {
		throw new Error("setMetadata is only supported by offline providers");
	}

	async listJournalEntries(): Promise<JournalEntry[]> {
		return [];
	}

	async clearJournalEntries(): Promise<void> {
		return Promise.resolve();
	}

	async clear(table: SchemaIdentifier): Promise<void> {
		throw new Error(
			`clear(${table.tableName}) is not supported by the online provider`,
		);
	}

	async close(): Promise<void> {
		return Promise.resolve();
	}

	async getEntity<TPayload>(
		ref: EntityIdentifier,
	): Promise<EntityRecord<TPayload>> {
		try {
			const document = await this.entitiesClient.get<
				Entities.GetDocumentResponses,
				Entities.GetDocumentErrors,
				true,
				"data"
			>({
				url: "/entities/{tableName}/documents/{entityId}",
				path: {
					entityId: ref.entityId,
					tableName: ref.tableName,
				},
				security: BEARER_SECURITY,
			});

			return this.toEntityRecord<TPayload>(ref.tableName, document);
		} catch (error) {
			throw wrapProviderError(
				`Failed to fetch entity ${ref.entityId} from ${ref.tableName}`,
				error,
			);
		}
	}

	async queryEntities<TPayload = unknown>(
		scope: SchemaIdentifier,
		options?: QueryOptions,
	): Promise<PaginatedResult<EntityRecord<TPayload>>> {
		try {
			const response = await this.entitiesClient.get<
				Entities.ListDocumentsResponses,
				Entities.ListDocumentsErrors,
				true,
				"data"
			>({
				url: "/entities/{tableName}/documents",
				path: { tableName: scope.tableName },
				// TODO: includeDeleted is ignored until the backend listDocuments endpoint supports it.
				query: this.toPaginationQuery(options?.pagination),
				security: BEARER_SECURITY,
			});

			return {
				items: response.items.map((item: Entities.EntityDocument) =>
					this.toEntityRecord<TPayload>(scope.tableName, item),
				),
				page: response.page,
				pageSize: response.pageSize,
				totalItems: response.totalItems,
				totalPages: response.totalPages,
			};
		} catch (error) {
			throw wrapProviderError(
				`Failed to query entities for ${scope.tableName}`,
				error,
			);
		}
	}

	async saveEntity<TPayload>(
		input: SaveEntityInput<TPayload>,
	): Promise<EntityRecord<TPayload>> {
		try {
			const payload = toJsonObject(input.payload);
			if (input.entityId) {
				try {
					const updated = await this.entitiesClient.patch<
						Entities.UpdateDocumentResponses,
						Entities.UpdateDocumentErrors,
						true,
						"data"
					>({
						url: "/entities/{tableName}/documents/{entityId}",
						path: {
							entityId: input.entityId,
							tableName: input.tableName,
						},
						body: { payload },
						headers: { "Content-Type": "application/json" },
						security: BEARER_SECURITY,
					});

					return this.toEntityRecord<TPayload>(input.tableName, updated);
				} catch (error) {
					if (!this.isNotFoundError(error)) {
						throw error;
					}
					const created = await this.entitiesClient.post<
						Entities.CreateDocumentResponses,
						Entities.CreateDocumentErrors,
						true,
						"data"
					>({
						url: "/entities/{tableName}/documents",
						path: { tableName: input.tableName },
						body: { entityId: input.entityId, payload },
						headers: { "Content-Type": "application/json" },
						security: BEARER_SECURITY,
					});

					return this.toEntityRecord<TPayload>(input.tableName, created);
				}
			}

			const created = await this.entitiesClient.post<
				Entities.CreateDocumentResponses,
				Entities.CreateDocumentErrors,
				true,
				"data"
			>({
				url: "/entities/{tableName}/documents",
				path: { tableName: input.tableName },
				body: { payload },
				headers: { "Content-Type": "application/json" },
				security: BEARER_SECURITY,
			});

			return this.toEntityRecord<TPayload>(input.tableName, created);
		} catch (error) {
			throw wrapProviderError(
				`Failed to save entity in ${input.tableName}`,
				error,
			);
		}
	}

	async deleteEntity(input: DeleteEntityInput): Promise<void> {
		try {
			await this.entitiesClient.delete<
				Entities.DeleteDocumentResponses,
				Entities.DeleteDocumentErrors,
				true,
				"data"
			>({
				url: "/entities/{tableName}/documents/{entityId}",
				path: {
					entityId: input.entityId,
					tableName: input.tableName,
				},
				security: BEARER_SECURITY,
			});
		} catch (error) {
			throw wrapProviderError(
				`Failed to delete entity ${input.entityId} from ${input.tableName}`,
				error,
			);
		}
	}

	async batchWrites(
		operations: BatchWrite,
		writeInJournal: boolean = false,
		_onProgress?: BatchWriteProgressListener,
	): Promise<void> {
		throw new Error("batchWrites is not implementaed. WIP");
	}

	private async resolveToken(): Promise<string | undefined> {
		if (!this.tokenSupplier) {
			return undefined;
		}
		const token = this.tokenSupplier();
		return token instanceof Promise ? await token : token;
	}

	private toPaginationQuery(
		pagination?: PaginationQuery,
	): Record<string, number> | undefined {
		if (!pagination) {
			return undefined;
		}
		const query: Record<string, number> = {};
		if (typeof pagination.page === "number") {
			query.page = pagination.page;
		}
		if (typeof pagination.pageSize === "number") {
			query.pageSize = pagination.pageSize;
		}
		return Object.keys(query).length > 0 ? query : undefined;
	}

	private toEntityRecord<TPayload>(
		tableName: string,
		document: Entities.EntityDocument,
	): EntityRecord<TPayload> {
		return {
			entityId: document.entityId,
			entityVersion: document.entityVersion,
			tableName,
			schemaVersion: document.schemaVersion,
			payload: fromWireJson<TPayload>(document.payload as JsonValue),
			createdAt: new Date(document.createdAt),
			isDeleted: Boolean(document.isDeleted),
			isActive: Boolean(document.isActive),
		};
	}

	private isNotFoundError(error: unknown): boolean {
		if (!error || typeof error !== "object") {
			return false;
		}
		const candidate = error as { status?: unknown; cause?: unknown };
		if (typeof candidate.status === "number") {
			return candidate.status === 404;
		}
		if (candidate.cause) {
			return this.isNotFoundError(candidate.cause);
		}
		return false;
	}
}
