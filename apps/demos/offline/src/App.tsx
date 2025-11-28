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
import {
	useCreatePerson,
	useDeletePerson,
	usePerson,
	usePersonList,
	useUpdatePerson,
} from "./domains/person/use-persons";

const queryClient = new QueryClient();
const PAGE_SIZE = 5;

function ListPage() {
	const [page, setPage] = useState(1);
	const [queuedOnly, setQueuedOnly] = useState(false);
	const { data, isLoading, refetch } = usePersonList({
		page,
		pageSize: PAGE_SIZE,
		queuedOnly,
	});
	const deleteMutation = useDeletePerson();

	useEffect(() => {
		refetch();
	}, [refetch]);

	const handleDelete = async (id: string) => {
		if (!window.confirm("Delete this person?")) return;
		await deleteMutation.mutateAsync(id);
	};

	return (
		<div className="app-shell">
			<h1 style={{ marginTop: 0 }}>Person Demo</h1>
			<p style={{ color: "#475569", maxWidth: 720 }}>
				Local-only CRUD using the persistence-sdk demo provider. Create, update,
				delete, and filter queued records while offline.
			</p>
			<PersonTable
				items={data?.items ?? []}
				isLoading={isLoading}
				queuedOnly={queuedOnly}
				onToggleQueuedOnly={(v) => {
					setQueuedOnly(v);
					setPage(1);
				}}
				onDelete={handleDelete}
				onSelectPage={(delta) =>
					setPage((p) =>
						Math.max(1, Math.min(data?.totalPages ?? 1, p + delta)),
					)
				}
				page={page}
				totalPages={data?.totalPages ?? 1}
			/>
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
				defaultValues={personQuery.data.entity}
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
					<Route path="/" element={<ListPage />} />
					<Route path="/new" element={<CreatePage />} />
					<Route path=":entityId/edit" element={<EditPage />} />
				</Routes>
			</BrowserRouter>
			<ToastHost />
		</QueryClientProvider>
	);
}
