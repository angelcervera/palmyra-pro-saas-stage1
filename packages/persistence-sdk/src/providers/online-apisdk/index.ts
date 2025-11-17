import {
	createEntitiesClient,
	createSchemaRepositoryClient,
	type Entities,
	type EntitiesClient,
	type SchemaRepository,
	type SchemaRepositoryClient,
} from "@zengateglobal/api-sdk";
import {
	type BatchWrite,
	BatchWriteError,
	type DeleteEntityInput,
	type EntityIdentifier,
	type EntityRecord,
	type MetadataSnapshot,
	type PaginatedResult,
	type PaginationQuery,
	type PersistenceProvider,
	type SaveEntityInput,
	type SchemaDefinition,
	type SchemaIdentifier,
	type SchemaMetadata,
} from "../../core";
import { describeProviderError, wrapProviderError } from "../../shared/errors";
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

	async getMetadata(): Promise<MetadataSnapshot> {
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

			return this.buildMetadataSnapshot(response);
		} catch (error) {
			throw wrapProviderError("Failed to load schema metadata", error);
		}
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

	async queryEntities<TPayload>(
		scope: SchemaIdentifier,
		pagination?: PaginationQuery,
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
				query: this.toPaginationQuery(pagination),
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

	async batchWrites(operations: BatchWrite[]): Promise<void> {
		for (const operation of operations) {
			try {
				if (operation.type === "save") {
					await this.saveEntity(operation.data);
					continue;
				}
				await this.deleteEntity(operation.data);
			} catch (error) {
				throw new BatchWriteError({
					tableName: operation.data.tableName,
					entityId: operation.data.entityId,
					reason: describeProviderError(error),
				});
			}
		}
	}

	private async resolveToken(): Promise<string | undefined> {
		if (!this.tokenSupplier) {
			return undefined;
		}
		const token = this.tokenSupplier();
		return token instanceof Promise ? await token : token;
	}

	private buildMetadataSnapshot(
		schemas: SchemaRepository.SchemaVersionList,
	): MetadataSnapshot {
		const tables = new Map<string, SchemaMetadata>();
		for (const schema of schemas.items ?? []) {
			let entry = tables.get(schema.tableName);
			if (!entry) {
				entry = {
					tableName: schema.tableName,
					versions: new Map<string, SchemaDefinition>(),
					activeVersion: schema.schemaVersion,
				};
				tables.set(schema.tableName, entry);
			}
			entry.versions.set(schema.schemaVersion, schema.schemaDefinition ?? {});
			if (schema.isActive) {
				entry.activeVersion = schema.schemaVersion;
			}
		}

		return {
			tables,
			fetchedAt: new Date(),
		};
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
			tableName,
			entityId: document.entityId,
			entityVersion: document.entityVersion,
			schemaVersion: document.schemaVersion,
			payload: fromWireJson<TPayload>(document.payload as JsonValue),
			ts: new Date(document.createdAt),
			isDeleted: Boolean(document.isSoftDeleted),
		};
	}
}

export function createOnlineOpenAPIPersistenceProvider(
	options: OnlineApiSdkProviderOptions = {},
): PersistenceProvider {
	return createOnlineApiSdkProvider(options);
}
