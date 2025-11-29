import { Link } from "react-router-dom";
import * as React from "react";

import type { JournalEntry, Schema } from "@zengateglobal/persistence-sdk";

import { runWithClient } from "./domains/persistence/helpers";

async function fetchSchemas(): Promise<Schema[]> {
	return runWithClient("Load schemas", (c) => c.getMetadata());
}

async function fetchJournal(): Promise<JournalEntry[]> {
	return runWithClient("Load journal", (c) => c.listJournalEntries());
}

type JournalGroup = {
	tableName: string;
	schemaVersion: string;
	count: number;
	firstSeen?: string;
	lastSeen?: string;
};

function groupJournal(entries: JournalEntry[]): JournalGroup[] {
	const map = new Map<string, JournalGroup>();
	for (const entry of entries) {
		const key = `${entry.tableName}:${entry.schemaVersion}`;
		const createdAt =
			entry.createdAt instanceof Date
				? entry.createdAt
				: new Date(entry.createdAt);
		const existing = map.get(key);
		if (!existing) {
			map.set(key, {
				tableName: entry.tableName,
				schemaVersion: entry.schemaVersion,
				count: 1,
				firstSeen: createdAt.toISOString(),
				lastSeen: createdAt.toISOString(),
			});
		} else {
			existing.count += 1;
			if (existing.firstSeen && createdAt.toISOString() < existing.firstSeen) {
				existing.firstSeen = createdAt.toISOString();
			}
			if (existing.lastSeen && createdAt.toISOString() > existing.lastSeen) {
				existing.lastSeen = createdAt.toISOString();
			}
		}
	}
	return Array.from(map.values()).sort((a, b) =>
		a.tableName.localeCompare(b.tableName),
	);
}

export function SyncPage() {
	const [schemas, setSchemas] = React.useState<Schema[]>([]);
	const [groups, setGroups] = React.useState<JournalGroup[]>([]);
	const [loading, setLoading] = React.useState(true);

	React.useEffect(() => {
		void (async () => {
			try {
				const [s, j] = await Promise.all([fetchSchemas(), fetchJournal()]);
				setSchemas(s);
				setGroups(groupJournal(j));
			} finally {
				setLoading(false);
			}
		})();
	}, []);

	return (
		<div className="app-shell">
			<div className="toolbar" style={{ gap: 12, alignItems: "center" }}>
				<Link className="link" to="/sync">
					Sync
				</Link>
				<span aria-hidden="true">|</span>
				<Link className="link" to="/persons">
					Persons
				</Link>
			</div>
			<h1 style={{ marginTop: 8 }}>Sync status</h1>
			<p style={{ color: "#475569", maxWidth: 720 }}>
				Preview of local journal entries grouped by schema/table. Use this to
				verify what would sync when connectivity is available.
			</p>
			{loading ? <p>Loading…</p> : null}
			<div className="card">
				<h2 style={{ marginTop: 0 }}>Schemas</h2>
				{schemas.length === 0 ? (
					<p>No schemas loaded.</p>
				) : (
					<ul>
						{schemas.map((s) => (
							<li key={`${s.tableName}:${s.schemaVersion}`}>
								<strong>{s.tableName}</strong> v{s.schemaVersion}
								{s.isActive ? " (active)" : ""}
							</li>
						))}
					</ul>
				)}
			</div>

			<div className="card">
				<h2 style={{ marginTop: 0 }}>Journal</h2>
				{groups.length === 0 ? (
					<p>No pending journal entries.</p>
				) : (
					<table>
						<thead>
							<tr>
								<th>Table</th>
								<th>Schema</th>
								<th>Count</th>
								<th>First seen</th>
								<th>Last seen</th>
							</tr>
						</thead>
						<tbody>
							{groups.map((g) => (
								<tr key={`${g.tableName}:${g.schemaVersion}`}>
									<td>{g.tableName}</td>
									<td>{g.schemaVersion}</td>
									<td>{g.count}</td>
									<td>{g.firstSeen?.replace("T", " ").replace("Z", "")}</td>
									<td>{g.lastSeen?.replace("T", " ").replace("Z", "")}</td>
								</tr>
							))}
						</tbody>
					</table>
				)}
			</div>
		</div>
	);
}
