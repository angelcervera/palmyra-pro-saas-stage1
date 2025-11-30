export interface SyncRequest {
	/**
	 * Provider that currently holds the journal to be pushed (e.g., offline).
	 */
	sourceProviderId: string;
	/**
	 * Provider that will receive outgoing changes and serve as the pull source.
	 */
	targetProviderId: string;
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
