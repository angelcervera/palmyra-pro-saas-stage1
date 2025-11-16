import type { SchemaIdentifier } from "./schemas";

// This is a temporary interface until we have a proper sync contract defined.

export interface SyncRequest {
	sourceProviderId: string;
	targetProviderId: string;
	schemas?: SchemaIdentifier[];
}

export interface SyncReport {
	startedAt: Date;
	finishedAt: Date;
	status: "success" | "partial" | "error";
	details: Array<{
		tableName: string;
		entitiesSynced: number;
		errors?: string[];
	}>;
}
