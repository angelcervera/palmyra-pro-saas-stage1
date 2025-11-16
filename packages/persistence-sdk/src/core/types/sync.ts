
export interface SyncRequest {
    sourceProviderId: string;
    targetProviderId: string;
    tables?: string[];
}

export interface SyncReport {
    startedAt: Date;
    finishedAt: Date;
    status: 'success' | 'partial' | 'error';
    details: Array<{
        tableName: string;
        entitiesSynced: number;
        errors?: string[];
    }>;
}
