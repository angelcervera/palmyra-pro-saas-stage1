/**
 * Schema/table metadata types.
 */
export interface SchemaIdentifier {
	tableName: string;
}

export interface SchemaIdentifierVersioned extends SchemaIdentifier {
	schemaVersion: string;
}

export type SchemaDefinition = {
	[key: string]: unknown;
};

export interface Schema extends SchemaIdentifierVersioned {
	schemaDefinition: SchemaDefinition;
	categoryId: string;
	createdAt: Date;
	isDeleted: boolean;
	isActive: boolean;
}
