import * as React from "react";
import type {
	JournalEntry,
	Schema,
	SyncReport,
} from "@zengateglobal/persistence-sdk";

import { runWithClient } from "../persistence/helpers";
import { pushToast } from "../../components/toast";

async function fetchSchemas(): Promise<Schema[]> {
	return runWithClient("Load schemas", (c) => c.getMetadata());
}

async function fetchJournal(): Promise<JournalEntry[]> {
	return runWithClient("Load journal", (c) => c.listJournalEntries());
}

// TODO: replace with real sync wiring once backend connectivity is available.
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
	const [syncing, setSyncing] = React.useState(false);
	const [progress, setProgress] = React.useState<string | null>(null);

	const load = React.useCallback(async () => {
		setLoading(true);
		try {
			const [s, j] = await Promise.all([fetchSchemas(), fetchJournal()]);
			setSchemas(s);
			setGroups(groupJournal(j));
		} finally {
			setLoading(false);
		}
	}, []);

	const handleSyncReport = React.useCallback((report: SyncReport) => {
		if (report.status === "success") {
			pushToast({ kind: "success", title: "Sync complete" });
			return;
		}

		const firstErroredTable = report.details.find(
			(detail) => detail.errors && detail.errors.length > 0,
		);
		const firstError = firstErroredTable?.errors?.[0];
		const description =
			firstErroredTable && firstError
				? `${firstErroredTable.tableName}: ${firstError}`
				: "One or more steps failed. See progress for details.";

		pushToast({
			kind: "error",
			title: report.status === "partial" ? "Sync partially completed" : "Sync failed",
			description,
		});
	}, []);

	const handleSync = React.useCallback(async () => {
		setSyncing(true);
		setProgress(null);
		try {
			const report = await runWithClient("Sync", async (client) => {
				const providers = client.getProviders();
				if (providers.length < 2) {
					throw new Error("At least two providers are required to sync.");
				}
				const [source, target] = providers;
				return await client.sync({
					sourceProviderId: source.name,
					targetProviderId: target.name,
					onProgress: (event) => {
						switch (event.stage) {
							case "push:start":
								setProgress(`Pushing journal (${event.journalCount} changes)…`);
								break;
							case "push:progress":
								setProgress(
									`Pushing journal ${event.written}/${event.total}…`,
								);
								break;
							case "push:success":
								setProgress("Journal pushed");
								break;
							case "journal:cleared":
								setProgress("Journal cleared");
								break;
							case "schemas:refreshed":
								setProgress(`Schemas refreshed (${event.schemaCount})`);
								break;
							case "clear:start":
								setProgress(`Clearing local tables (${event.tableCount})…`);
								break;
							case "clear:success":
								setProgress("Local tables cleared");
								break;
							case "pull:start":
								setProgress(`Pulling ${event.tableName}…`);
								break;
							case "pull:progress":
								setProgress(
									`Pulling ${event.tableName} page ${event.page}/${event.totalPages} (${event.written}/${event.total})…`,
								);
								break;
							case "pull:page":
								setProgress(
									`Pulled ${event.count} from ${event.tableName} (page ${event.page}/${event.totalPages})`,
								);
								break;
							case "pull:error":
								setProgress(`Pull failed for ${event.tableName}: ${event.error}`);
								break;
							case "done":
								setProgress(`Sync ${event.status}`);
								break;
						}
					},
				});
			});
			handleSyncReport(report);
			if (report.status !== "error") {
				await load();
			}
		} catch (error) {
			pushToast({
				kind: "error",
				title: "Sync failed",
				description:
					error instanceof Error ? error.message : String(error ?? "Unknown"),
			});
		} finally {
			setSyncing(false);
		}
	}, [handleSyncReport, load]);

	React.useEffect(() => {
		void load();
	}, [load]);

	return (
		<div className="app-shell">
			<h1 style={{ marginTop: 8 }}>Sync status</h1>
			<p style={{ color: "#475569", maxWidth: 720 }}>
				Preview of local journal entries grouped by schema/table. Use this to
				verify what would sync when connectivity is available.
			</p>
			<div className="toolbar" style={{ margin: "12px 0" }}>
				<button
					type="button"
					className="btn primary"
					onClick={handleSync}
					disabled={loading || syncing}
				>
					{syncing ? "Syncing..." : "Sync now"}
				</button>
				{progress ? (
					<span style={{ color: "#475569", fontSize: 14 }}>{progress}</span>
				) : null}
			</div>
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
