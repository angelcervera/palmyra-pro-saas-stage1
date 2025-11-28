import type { SchemaIdentifier, SchemaIdentifierVersioned } from "./schemas";

export interface EntityIdentifier extends SchemaIdentifier {
	entityId: string;
}

export interface EntityIdentifierVersioned extends EntityIdentifier {
	entityVersion: string;
}

export interface EntityRecord<TPayload = unknown>
	extends SchemaIdentifierVersioned,
		EntityIdentifierVersioned {
	payload: TPayload;
	ts: Date;
	isDeleted: boolean;
	isActive: boolean;
}

export type BatchWrite = EntityRecord[];

// Represent an entity following the active schema version.
// When the `entityId` is not present, it means that it is new.
export interface SaveEntityInput<TPayload = unknown> extends SchemaIdentifier {
	entityId?: string;
	payload: TPayload;
}

export interface DeleteEntityInput extends EntityIdentifier {}

export class BatchWriteError extends Error {
	readonly tableName: string;
	readonly entityId: string;

	constructor(params: {
		tableName: string;
		entityId: string;
		reason: string;
	}) {
		super(params.reason);
		this.tableName = params.tableName;
		this.entityId = params.entityId;
	}
}
