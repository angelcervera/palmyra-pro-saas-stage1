import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import * as React from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import type { Schema } from "@zengateglobal/persistence-sdk";

import { PersonForm } from "./components/PersonForm";
import { PersonTable } from "./components/PersonTable";
import {
	useCreatePerson,
	useDeletePerson,
	usePerson,
	usePersonList,
	useUpdatePerson,
} from "./use-persons";
import { runWithClient } from "../persistence/helpers";

const queryClient = new QueryClient();
const PAGE_SIZE = 5;
const loadSchemas = async () =>
	runWithClient("Load schemas", (c) => c.getMetadata());

function PersonsListPage() {
	const [page, setPage] = React.useState(1);
	const [confirmDeleteId, setConfirmDeleteId] = React.useState<string | null>(
		null,
	);
	const [schemasReady, setSchemasReady] = React.useState<boolean | null>(null);
	const { data, isLoading, refetch } = usePersonList(
		schemasReady ? { page, pageSize: PAGE_SIZE } : null,
	);
	const deleteMutation = useDeletePerson();

	React.useEffect(() => {
		refetch();
	}, [refetch]);

	React.useEffect(() => {
		void (async () => {
				try {
					const schemas = await loadSchemas();
					const hasPersons = schemas.some(
						(s: Schema) => s.tableName === "persons" && !s.isDeleted,
					);
					setSchemasReady(hasPersons);
				} catch {
					setSchemasReady(false);
				}
			})();
		}, []);

	const handleDelete = async (id: string) => {
		await deleteMutation.mutateAsync(id);
		setConfirmDeleteId(null);
	};

	return (
		<div className="app-shell">
			<h1 style={{ marginTop: 8 }}>Person Demo</h1>
			<p style={{ color: "#475569", maxWidth: 720 }}>
				Local-only CRUD using the persistence-sdk demo provider. Create, update,
				delete, and filter queued records while offline.
			</p>
			{schemasReady === null ? (
				<p>Checking schemas…</p>
			) : schemasReady === false ? (
				<div className="card">
					<p>
						The persons schema isn&apos;t loaded yet. Please open the sync page
						to load metadata and retry. Data will not load until schemas are
						available.
					</p>
					<Link className="btn primary" to="/sync">
						Go to sync
					</Link>
				</div>
			) : null}

			{schemasReady ? (
				<>
					<PersonTable
						items={data?.items ?? []}
						isLoading={isLoading}
						onDelete={(id) => setConfirmDeleteId(id)}
						onSelectPage={(delta) =>
							setPage((p) =>
								Math.max(1, Math.min(data?.totalPages ?? 1, p + delta)),
							)
						}
						page={page}
						totalPages={data?.totalPages ?? 1}
					/>
					{confirmDeleteId && (
						<div className="modal">
							<div className="modal__content">
								<p>Delete this person?</p>
								<div className="toolbar" style={{ gap: 8 }}>
									<button
										type="button"
										className="btn"
										onClick={() => setConfirmDeleteId(null)}
									>
										Cancel
									</button>
									<button
										type="button"
										className="btn primary"
										onClick={() => handleDelete(confirmDeleteId)}
										disabled={deleteMutation.isPending}
									>
										{deleteMutation.isPending ? "Deleting..." : "Yes, delete"}
									</button>
								</div>
							</div>
						</div>
					)}
				</>
			) : null}
		</div>
	);
}

function CreatePersonPageInner() {
	const navigate = useNavigate();
	const mutation = useCreatePerson();

	return (
		<div className="app-shell">
			<Link className="link" to="/persons">
				← Back to list
			</Link>
			<h1>Create person</h1>
			<PersonForm
				onSubmit={async (values) => {
					await mutation.mutateAsync(values);
					navigate("/persons");
				}}
				submitLabel={mutation.isPending ? "Saving..." : "Save person"}
				cancelTo="/persons"
				isSubmitting={mutation.isPending}
			/>
		</div>
	);
}

function EditPersonPageInner() {
	const { entityId } = useParams();
	const navigate = useNavigate();
	const personQuery = usePerson(entityId);
	const mutation = useUpdatePerson(entityId ?? "");

	if (!entityId) return <div className="app-shell">Missing id</div>;
	if (personQuery.isLoading) return <div className="app-shell">Loading...</div>;
	if (personQuery.isError || !personQuery.data)
		return <div className="app-shell">Unable to load person</div>;

	return (
		<div className="app-shell">
			<Link className="link" to="/persons">
				← Back to list
			</Link>
			<h1>Edit person</h1>
			<PersonForm
				defaultValues={personQuery.data.payload}
				onSubmit={async (values) => {
					await mutation.mutateAsync(values);
					navigate("/persons");
				}}
				submitLabel={mutation.isPending ? "Saving..." : "Save changes"}
				cancelTo="/persons"
				isSubmitting={mutation.isPending}
			/>
		</div>
	);
}

export function PersonsPage() {
	return (
		<QueryClientProvider client={queryClient}>
			<PersonsListPage />
		</QueryClientProvider>
	);
}

export function CreatePersonPage() {
	return (
		<QueryClientProvider client={queryClient}>
			<CreatePersonPageInner />
		</QueryClientProvider>
	);
}

export function EditPersonPage() {
	return (
		<QueryClientProvider client={queryClient}>
			<EditPersonPageInner />
		</QueryClientProvider>
	);
}
