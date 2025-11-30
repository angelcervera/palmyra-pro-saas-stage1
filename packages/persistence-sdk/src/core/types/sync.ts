export interface SyncRequest {
	/**
	 * Provider that currently holds the journal to be pushed (e.g., offline).
	 */
	sourceProviderId: string;
	/**
	 * Provider that will receive outgoing changes and serve as the pull source.
	 */
	targetProviderId: string;
	/**
	 * Optional progress callback fired at key points in the sync flow.
	 */
	onProgress?: SyncProgressListener;
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

export type SyncProgressStage =
	| "push:start"
	| "push:success"
	| "push:error"
	| "journal:cleared"
	| "schemas:refreshed"
	| "clear:start"
	| "clear:success"
	| "pull:start"
	| "pull:page"
	| "pull:error"
	| "done";

export const SyncProgressStages: SyncProgressStage[] = [
	"push:start",
	"push:success",
	"push:error",
	"journal:cleared",
	"schemas:refreshed",
	"clear:start",
	"clear:success",
	"pull:start",
	"pull:page",
	"pull:error",
	"done",
];

export type SyncProgress =
	| { stage: "push:start"; journalCount: number }
	| { stage: "push:success"; journalCount: number }
	| { stage: "push:error"; error: string }
	| { stage: "journal:cleared" }
	| { stage: "schemas:refreshed"; schemaCount: number }
	| { stage: "clear:start"; tableCount: number }
	| { stage: "clear:success"; tableCount: number }
	| {
			stage: "pull:start";
			tableName: string;
			pageSize: number;
		}
	| {
			stage: "pull:page";
			tableName: string;
			page: number;
			totalPages: number;
			count: number;
		}
	| { stage: "pull:error"; tableName: string; error: string }
	| { stage: "done"; status: SyncReport["status"] };

export type SyncProgressListener = (progress: SyncProgress) => void;
