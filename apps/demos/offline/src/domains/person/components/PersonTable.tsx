import { Link } from "react-router-dom";

import type { PersonRecord } from "../persistence";

type Props = {
	items: PersonRecord[];
	isLoading?: boolean;
	onDelete(id: string): void;
	onSelectPage(delta: number): void;
	page: number;
	totalPages: number;
};

export function PersonTable({
	items,
	isLoading,
	onDelete,
	onSelectPage,
	page,
	totalPages,
}: Props) {
	return (
		<div className="card">
			<div className="toolbar" style={{ justifyContent: "space-between" }}>
				<div className="toolbar" style={{ gap: 8 }}>
					<Link className="btn primary" to="/persons/new">
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
								>
									<td>{idx + 1}</td>
									<td>
										<img
											src={p.payload.photo}
											alt={p.payload.name}
											className="photo-thumb"
										/>
									</td>
									<td>{p.payload.name}</td>
									<td>{p.payload.surname}</td>
									<td>{p.payload.age}</td>
									<td>{p.payload.dob}</td>
									<td>{p.payload.phoneNumber}</td>
									<td>-</td>
									<td style={{ textAlign: "right" }}>
										<Link
											className="link"
											to={`/persons/${p.entityId}/edit`}
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
