import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import {
	BrowserRouter,
	Link,
	Route,
	Routes,
	useNavigate,
	useParams,
} from "react-router-dom";

import { ToastHost } from "./components/toast";
import { PersonForm } from "./domains/person/components/PersonForm";
import { PersonTable } from "./domains/person/components/PersonTable";
import { runWithClient } from "./domains/persistence/helpers";
import {
	useCreatePerson,
	useDeletePerson,
	usePerson,
	usePersonList,
	useUpdatePerson,
} from "./domains/person/use-persons";
import { SyncPage } from "./SyncPage";
import { TopNav } from "./components/TopNav";

const queryClient = new QueryClient();
const PAGE_SIZE = 5;

function ListPage() {
	const [page, setPage] = useState(1);
	const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
	const [schemasReady, setSchemasReady] = useState<boolean | null>(null);
	const { data, isLoading, refetch } = usePersonList(
		schemasReady ? { page, pageSize: PAGE_SIZE } : null,
	);
	const deleteMutation = useDeletePerson();

	useEffect(() => {
		refetch();
	}, [refetch]);

	useEffect(() => {
		void (async () => {
			try {
				const schemas = await runWithClient("Load schemas", (c) =>
					c.getMetadata(),
				);
				const hasPersons = schemas.some(
					(s) => s.tableName === "persons" && !s.isDeleted,
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
			<TopNav active="persons" />
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

function CreatePage() {
	const navigate = useNavigate();
	const mutation = useCreatePerson();

	return (
		<div className="app-shell">
			<Link className="link" to="/">
				← Back to list
			</Link>
			<h1>Create person</h1>
			<PersonForm
				onSubmit={async (values) => {
					await mutation.mutateAsync(values);
					navigate("/");
				}}
				submitLabel={mutation.isPending ? "Saving..." : "Save person"}
				cancelTo="/"
				isSubmitting={mutation.isPending}
			/>
		</div>
	);
}

function EditPage() {
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
			<Link className="link" to="/">
				← Back to list
			</Link>
			<h1>Edit person</h1>
			<PersonForm
				defaultValues={personQuery.data.payload}
				onSubmit={async (values) => {
					await mutation.mutateAsync(values);
					navigate("/");
				}}
				submitLabel={mutation.isPending ? "Saving..." : "Save changes"}
				cancelTo="/"
				isSubmitting={mutation.isPending}
			/>
		</div>
	);
}

export default function App() {
	return (
		<QueryClientProvider client={queryClient}>
			<BrowserRouter>
				<Routes>
					<Route path="/" element={<SyncPage />} />
					<Route path="/persons" element={<ListPage />} />
					<Route path="/new" element={<CreatePage />} />
					<Route path=":entityId/edit" element={<EditPage />} />
					<Route path="/sync" element={<SyncPage />} />
				</Routes>
			</BrowserRouter>
			<ToastHost />
		</QueryClientProvider>
	);
}
