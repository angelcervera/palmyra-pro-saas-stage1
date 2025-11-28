import type { EntityRecord } from "./entities";

export interface JournalEntry extends EntityRecord {
	changeId: number;
	changeDate: Date;
}
