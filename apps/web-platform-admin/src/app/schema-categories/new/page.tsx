import { useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";

import {
	SchemaCategoryForm,
	type SchemaCategoryFormValues,
} from "../schema-category-form";
import { useCreateSchemaCategory } from "../use-schema-categories";

export default function SchemaCategoriesCreatePage() {
	const navigate = useNavigate();
	const createMutation = useCreateSchemaCategory();

	const handleSubmit = useCallback(
		async (values: SchemaCategoryFormValues) => {
			try {
				await createMutation.mutateAsync(values);
				toast.success(`Schema category "${values.name}" created.`);
				navigate("/schema-categories", {
					replace: true,
					state: {
						toast: {
							type: "success",
							message: `Schema category "${values.name}" created.`,
						},
					},
				});
			} catch (error) {
				const message =
					error instanceof Error
						? error.message
						: "Failed to create schema category. Please try again.";
				toast.error(message);
			}
		},
		[createMutation, navigate],
	);

	return (
		<div className="flex flex-1 flex-col gap-6 p-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-semibold">Create schema category</h1>
					<p className="text-muted-foreground">
						Define a new category to classify persisted schemas.
					</p>
				</div>
				<Button variant="ghost" onClick={() => navigate("/schema-categories")}>
					Cancel
				</Button>
			</div>

			<Card>
				<CardHeader>
					<CardTitle>Category details</CardTitle>
					<CardDescription>
						Provide the metadata that identifies this schema category.
					</CardDescription>
				</CardHeader>
				<CardContent>
					<SchemaCategoryForm
						isSubmitting={createMutation.isPending}
						onSubmit={handleSubmit}
					/>
				</CardContent>
			</Card>
		</div>
	);
}
