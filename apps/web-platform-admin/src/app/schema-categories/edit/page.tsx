import { useCallback } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

import {
	SchemaCategoryForm,
	type SchemaCategoryFormValues,
} from "../schema-category-form";
import {
	useSchemaCategory,
	useUpdateSchemaCategory,
} from "../use-schema-categories";

export default function SchemaCategoriesEditPage() {
	const navigate = useNavigate();
	const { categoryId } = useParams<{ categoryId: string }>();
	const { data, isLoading, isError, error } = useSchemaCategory(categoryId);
	const updateMutation = useUpdateSchemaCategory(categoryId ?? "");

	const handleSubmit = useCallback(
		async (values: SchemaCategoryFormValues) => {
			if (!categoryId) {
				return;
			}

			try {
				await updateMutation.mutateAsync(values);
				toast.success(`Schema category "${values.name}" updated.`);
				navigate("/schema-categories", {
					replace: true,
					state: {
						toast: {
							type: "success",
							message: `Schema category "${values.name}" updated.`,
						},
					},
				});
			} catch (mutationError) {
				const message =
					mutationError instanceof Error
						? mutationError.message
						: "Failed to update schema category. Please try again.";
				toast.error(message);
			}
		},
		[categoryId, updateMutation, navigate],
	);

	const heading = data?.name ?? "Schema category";

	return (
		<div className="flex flex-1 flex-col gap-6 p-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-semibold">
						{isLoading ? <Skeleton className="h-7 w-48" /> : `Edit ${heading}`}
					</h1>
					<p className="text-muted-foreground">
						Update the metadata for this schema category.
					</p>
				</div>
				<Button variant="ghost" onClick={() => navigate("/schema-categories")}>
					Cancel
				</Button>
			</div>

			{isLoading ? (
				<Skeleton className="h-[320px] w-full rounded-xl" />
			) : isError || !data || !categoryId ? (
				<div className="rounded-md border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
					Unable to load schema category.{" "}
					{error instanceof Error ? error.message : "Please try again."}
				</div>
			) : (
				<Card>
					<CardHeader>
						<CardTitle>Category details</CardTitle>
						<CardDescription>
							Adjust the slug, name, description, or parent relationship.
						</CardDescription>
					</CardHeader>
					<CardContent>
						<SchemaCategoryForm
							defaultValues={{
								name: data.name ?? "",
								slug: data.slug ?? "",
								description: data.description ?? undefined,
							}}
							isSubmitting={updateMutation.isPending}
							onSubmit={handleSubmit}
						/>
					</CardContent>
				</Card>
			)}
		</div>
	);
}
