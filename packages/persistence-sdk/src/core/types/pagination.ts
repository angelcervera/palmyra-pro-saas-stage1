/**
 * Pagination helpers used by provider/client contracts.
 */
export interface PaginationQuery {
	page?: number;
	pageSize?: number;
}

export interface PaginatedResult<T> {
	items: T[];
	page: number;
	pageSize: number;
	totalItems: number;
	totalPages: number;
}

// Future-proof container for query-time options (pagination, filters, flags).
export interface QueryOptions {
	pagination?: PaginationQuery;
	onlyActive?: boolean;
	includeDeleted?: boolean;
}
