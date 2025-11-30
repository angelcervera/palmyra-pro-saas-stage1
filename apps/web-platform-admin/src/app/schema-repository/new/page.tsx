import { useCallback, useMemo } from "react";
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
import { useSchemaCategories } from "../../schema-categories/use-schema-categories";
import {
	SchemaVersionForm,
	type SchemaVersionSubmitPayload,
} from "../schema-version-form";
import { useCreateSchemaVersion } from "../use-schema-repository";

export default function SchemaRepositoryCreatePage() {
	const navigate = useNavigate();
	const createMutation = useCreateSchemaVersion();
	const { data: categoryData } = useSchemaCategories();

	const categories = useMemo(
		() =>
			(categoryData ?? []).map((category) => ({
				id: category.categoryId,
				name: category.name ?? category.slug,
			})),
		[categoryData],
	);

	const handleSubmit = useCallback(
		async (values: SchemaVersionSubmitPayload) => {
			try {
				const result = await createMutation.mutateAsync({
					tableName: values.tableName,
					slug: values.slug,
					categoryId: values.categoryId,
					schemaDefinition: values.schemaDefinition,
				});

				toast.success(`Schema version ${result.schemaVersion} created.`);
				navigate(
					{
						pathname: "/schema-repository",
						search: `?schemaId=${encodeURIComponent(result.schemaId)}`,
					},
					{
						replace: true,
						state: {
							toast: {
								type: "success",
								message: `Schema version ${result.schemaVersion} created.`,
							},
						},
					},
				);
			} catch (mutationError) {
				const message =
					mutationError instanceof Error
						? mutationError.message
						: "Failed to create schema version. Please try again.";
				toast.error(message);
			}
		},
		[createMutation, navigate],
	);

	return (
		<div className="flex flex-1 flex-col gap-6 p-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-semibold">Create schema version</h1>
					<p className="text-muted-foreground">
						Register a new JSON schema definition and classify it under the
						right category.
					</p>
				</div>
				<Button variant="ghost" onClick={() => navigate(-1)}>
					Cancel
				</Button>
			</div>

			<Card>
				<CardHeader>
					<CardTitle>Schema metadata</CardTitle>
					<CardDescription>
						Provide the identifiers and JSON schema contents that describe this
						version.
					</CardDescription>
				</CardHeader>
				<CardContent>
					<SchemaVersionForm
						categories={categories}
						isSubmitting={createMutation.isPending}
						onSubmit={handleSubmit}
						submitLabel="Create schema version"
						className="max-w-4xl"
					/>
				</CardContent>
			</Card>
		</div>
	);
}
