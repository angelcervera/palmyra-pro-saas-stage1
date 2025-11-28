import type { EntityRecord } from "./entities";

export type JournalChangeType = "create" | "update" | "delete";

export interface JournalEntry extends EntityRecord {
	changeId: number;
	changeDate: Date;
	changeType: JournalChangeType;
}
