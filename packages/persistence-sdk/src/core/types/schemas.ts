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

export interface SchemaMetadata extends SchemaIdentifier {
	// version -> jsonSchema
	versions: Map<string, SchemaDefinition>;
	activeVersion: string;
}

export interface MetadataSnapshot {
	// tableName -> SchemaMetadata
	tables: Map<string, SchemaMetadata>;
	fetchedAt: Date;
}
