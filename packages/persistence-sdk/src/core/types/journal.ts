export type JournalChangeType = "create" | "update" | "delete";

export interface JournalEntry {
	changeId: number;
	tableName: string;
	entityId: string;
	entityVersion: string;
	schemaVersion: string;
	changeType: JournalChangeType;
	payload?: unknown;
}
