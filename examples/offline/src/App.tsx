import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";

import { TopNav } from "./components/TopNav";
import {
	CreatePersonPage,
	EditPersonPage,
	PersonsPage,
} from "./domains/person/PersonsPages";
import { SyncPage } from "./domains/sync/SyncPage";
import { ToastHost } from "./components/toast";

export default function App() {
	return (
		<BrowserRouter>
			<TopNav />
			<Routes>
				<Route path="/" element={<Navigate to="/sync" replace />} />
				<Route path="/sync" element={<SyncPage />} />
				<Route path="/persons" element={<PersonsPage />} />
				<Route path="/persons/new" element={<CreatePersonPage />} />
				<Route path="/persons/:entityId/edit" element={<EditPersonPage />} />
			</Routes>
			<ToastHost />
		</BrowserRouter>
	);
}
