import { useEffect, useSyncExternalStore } from "react";

export type ToastKind = "info" | "success" | "error";

export type Toast = {
	id: string;
	kind: ToastKind;
	title: string;
	description?: string;
};

type Listener = (toasts: Toast[]) => void;

const listeners = new Set<Listener>();
let toasts: Toast[] = [];

function emit() {
	for (const listener of listeners) listener(toasts);
}

function addToast(toast: Toast) {
	toasts = [...toasts, toast];
	emit();
	setTimeout(() => removeToast(toast.id), 4_000);
}

function removeToast(id: string) {
	toasts = toasts.filter((t) => t.id !== id);
	emit();
}

export function pushToast(params: {
	kind?: ToastKind;
	title: string;
	description?: string;
}) {
	const id =
		typeof crypto !== "undefined" && crypto.randomUUID
			? crypto.randomUUID()
			: Math.random().toString(36).slice(2, 10);
	addToast({
		id,
		kind: params.kind ?? "info",
		title: params.title,
		description: params.description,
	});
}

export function ToastHost() {
	const snapshot = useSyncExternalStore(
		(listener) => {
			listeners.add(listener);
			return () => listeners.delete(listener);
		},
		() => toasts,
	);

	useEffect(() => {
		const onError = (event: ErrorEvent) => {
			pushToast({
				kind: "error",
				title: "Unexpected error",
				description: event.message,
			});
		};
		const onRejection = (event: PromiseRejectionEvent) => {
			const message =
				event.reason instanceof Error
					? event.reason.message
					: typeof event.reason === "string"
						? event.reason
						: JSON.stringify(event.reason);
			pushToast({
				kind: "error",
				title: "Unhandled error",
				description: message,
			});
		};
		window.addEventListener("error", onError);
		window.addEventListener("unhandledrejection", onRejection);
		return () => {
			window.removeEventListener("error", onError);
			window.removeEventListener("unhandledrejection", onRejection);
			listeners.clear();
		};
	}, []);

	return (
		<div
			style={{
				position: "fixed",
				bottom: 16,
				right: 16,
				display: "flex",
				flexDirection: "column",
				gap: 8,
				zIndex: 50,
			}}
		>
			{snapshot.map((toast) => (
				<div
					key={toast.id}
					style={{
						minWidth: 260,
						maxWidth: 360,
						padding: 12,
						borderRadius: 8,
						boxShadow: "0 8px 24px rgba(0,0,0,0.12)",
						background: toast.kind === "error" ? "#fef2f2" : "#f8fafc",
						border: `1px solid ${toast.kind === "error" ? "#fecdd3" : "#e2e8f0"}`,
					}}
				>
					<div
						style={{
							display: "flex",
							justifyContent: "space-between",
							alignItems: "center",
							gap: 12,
						}}
					>
						<strong style={{ color: "#0f172a", fontSize: 14 }}>
							{toast.title}
						</strong>
						<button
							type="button"
							onClick={() => removeToast(toast.id)}
							style={{
								background: "transparent",
								border: "none",
								cursor: "pointer",
								color: "#475569",
							}}
							aria-label="Dismiss"
						>
							×
						</button>
					</div>
					{toast.description ? (
						<p style={{ margin: "6px 0 0", color: "#475569", fontSize: 13 }}>
							{toast.description}
						</p>
					) : null}
				</div>
			))}
		</div>
	);
}
