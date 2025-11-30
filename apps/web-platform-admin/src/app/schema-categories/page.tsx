import { useCallback, useEffect, useMemo, useRef } from "react";
import { useLocation } from "react-router-dom";
import { toast } from "sonner";

import { SchemaCategoriesTable } from "./schema-categories-table";
import {
	useDeleteSchemaCategory,
	useSchemaCategories,
} from "./use-schema-categories";

export default function SchemaCategoriesPage() {
	const location = useLocation();
	const hasShownFlashToast = useRef(false);
	const { data, isLoading, isError, error } = useSchemaCategories();
	const deleteMutation = useDeleteSchemaCategory();

	useEffect(() => {
		const state = location.state as
			| {
					toast?: {
						type: "success" | "error" | "info" | "warning";
						message: string;
					};
			  }
			| undefined;

		if (!state?.toast) {
			hasShownFlashToast.current = false;
			return;
		}

		if (hasShownFlashToast.current) {
			return;
		}

		hasShownFlashToast.current = true;
		const { type, message } = state.toast;

		switch (type) {
			case "success":
				toast.success(message);
				break;
			case "error":
				toast.error(message);
				break;
			case "warning":
				toast.warning(message);
				break;
			default:
				toast.info(message);
				break;
		}
	}, [location]);

	useEffect(() => {
		if (isError) {
			const message =
				error instanceof Error
					? error.message
					: "Failed to load schema categories.";
			toast.error(message);
		}
	}, [isError, error]);

	const categories = useMemo(() => data ?? [], [data]);

	const handleDelete = useCallback(
		async (categoryId: string, label: string) => {
			const confirmed = window.confirm(
				`Delete "${label}"? This action cannot be undone.`,
			);
			if (!confirmed) {
				return;
			}

			try {
				await deleteMutation.mutateAsync(categoryId);
				toast.success(`Schema category "${label}" deleted.`);
			} catch (mutationError) {
				const message =
					mutationError instanceof Error
						? mutationError.message
						: "Failed to delete schema category. Please try again.";
				toast.error(message);
			}
		},
		[deleteMutation],
	);

	return (
		<div className="flex flex-1 flex-col p-6">
			{isError ? (
				<div className="rounded-md border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
					Failed to load schema categories.{" "}
					{error instanceof Error ? error.message : "Please try again."}
				</div>
			) : (
				<SchemaCategoriesTable
					data={categories}
					isLoading={isLoading}
					deletingId={deleteMutation.variables ?? null}
					onDelete={handleDelete}
				/>
			)}
		</div>
	);
}
