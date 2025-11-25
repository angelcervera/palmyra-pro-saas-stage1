import { Link } from "react-router-dom";

import type { PersonRecord } from "../persistence";

type Props = {
	items: PersonRecord[];
	isLoading?: boolean;
	queuedOnly: boolean;
	onToggleQueuedOnly(value: boolean): void;
	onDelete(id: string): void;
	onSyncAll(): void;
	onSelectPage(delta: number): void;
	page: number;
	totalPages: number;
};

export function PersonTable({
	items,
	isLoading,
	queuedOnly,
	onToggleQueuedOnly,
	onDelete,
	onSyncAll,
	onSelectPage,
	page,
	totalPages,
}: Props) {
	return (
		<div className="card">
			<div className="toolbar" style={{ justifyContent: "space-between" }}>
				<div className="toolbar" style={{ gap: 8 }}>
					<input
						id="queued-filter"
						type="checkbox"
						checked={queuedOnly}
						onChange={(e) => onToggleQueuedOnly(e.target.checked)}
					/>
					<label htmlFor="queued-filter">Show only queued for sync</label>
				</div>
				<div className="toolbar" style={{ gap: 8 }}>
					<button
						className="btn"
						type="button"
						onClick={onSyncAll}
						disabled={isLoading}
					>
						Sync all
					</button>
					<Link className="btn primary" to="/new">
						New person
					</Link>
				</div>
			</div>

			<div style={{ marginTop: 16, overflowX: "auto" }}>
				<table>
					<thead>
						<tr>
							<th>#</th>
							<th>Photo</th>
							<th>Name</th>
							<th>Surname</th>
							<th>Age</th>
							<th>DOB</th>
							<th>Phone</th>
							<th>Sync</th>
							<th style={{ textAlign: "right" }}>Actions</th>
						</tr>
					</thead>
					<tbody>
						{isLoading ? (
							<tr>
								<td colSpan={9} style={{ textAlign: "center", padding: 16 }}>
									Loading...
								</td>
							</tr>
						) : items.length === 0 ? (
							<tr>
								<td colSpan={9} style={{ textAlign: "center", padding: 16 }}>
									No persons yet. Create one to get started.
								</td>
							</tr>
						) : (
							items.map((p, idx) => (
								<tr
									key={p.entityId}
									style={{ opacity: p.queuedForSync ? 1 : 0.9 }}
								>
									<td>{idx + 1}</td>
									<td>
										<img
											src={p.entity.photo}
											alt={p.entity.name}
											className="photo-thumb"
										/>
									</td>
									<td>{p.entity.name}</td>
									<td>{p.entity.surname}</td>
									<td>{p.entity.age}</td>
									<td>{p.entity.dob}</td>
									<td>{p.entity.phoneNumber}</td>
									<td>
										{p.queuedForSync ? (
											<span className="badge">Queued</span>
										) : (
											<span className="badge secondary">Synced</span>
										)}
									</td>
									<td style={{ textAlign: "right" }}>
										<Link
											className="link"
											to={`/${p.entityId}/edit`}
											style={{ marginRight: 12 }}
										>
											Edit
										</Link>
										<button
											type="button"
											className="btn"
											onClick={() => onDelete(p.entityId)}
										>
											Delete
										</button>
									</td>
								</tr>
							))
						)}
					</tbody>
				</table>
			</div>

			<div className="table-meta">
				<span>
					Page {page} of {totalPages}
				</span>
				<div className="toolbar" style={{ gap: 8 }}>
					<button
						type="button"
						className="btn"
						disabled={page <= 1}
						onClick={() => onSelectPage(-1)}
					>
						Prev
					</button>
					<button
						type="button"
						className="btn"
						disabled={page >= totalPages}
						onClick={() => onSelectPage(1)}
					>
						Next
					</button>
				</div>
			</div>
		</div>
	);
}
