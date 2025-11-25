// Root component for the admin app. Currently renders the route tree only,
// leaving space to layer cross-cutting UI (toasters, error boundaries, Suspense, devtools, error boundaries)
// without touching main.tsx.
import { lazy, Suspense } from "react";
import { Toaster } from "@/components/ui/sonner";
import { AdminRoutes } from "@/routes";

const ReactQueryDevtools = lazy(() =>
	import("@tanstack/react-query-devtools").then((m) => ({
		default: m.ReactQueryDevtools,
	})),
);

export default function App() {
	return (
		<>
			<AdminRoutes />
			<Toaster richColors closeButton />
			{import.meta.env.DEV && (
				<Suspense fallback={null}>
					<ReactQueryDevtools
						initialIsOpen={false}
						position="right"
						buttonPosition="bottom-right"
					/>
				</Suspense>
			)}
		</>
	);
}
