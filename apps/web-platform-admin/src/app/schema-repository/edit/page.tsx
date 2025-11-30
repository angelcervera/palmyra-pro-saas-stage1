import { useCallback, useMemo } from "react";
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
import { useSchemaCategories } from "../../schema-categories/use-schema-categories";
import {
	SchemaVersionForm,
	type SchemaVersionFormValues,
	type SchemaVersionSubmitPayload,
} from "../schema-version-form";
import {
	useCreateSchemaVersion,
	useSchemaVersion,
} from "../use-schema-repository";

export default function SchemaVersionEditPage() {
	const navigate = useNavigate();
	const params = useParams();
	const schemaId = params.schemaId ?? "";
	const schemaVersion = params.schemaVersion ?? "";

	const { data, isLoading } = useSchemaVersion(schemaId, schemaVersion);
	const { data: categoriesData } = useSchemaCategories();
	const createMutation = useCreateSchemaVersion();

	const categories = useMemo(
		() =>
			(categoriesData ?? []).map((category) => ({
				id: category.categoryId,
				name: category.name ?? category.slug,
			})),
		[categoriesData],
	);

	const defaultValues = useMemo<Partial<SchemaVersionFormValues>>(() => {
		if (!data) {
			return {};
		}
		return {
			tableName: data.tableName,
			slug: data.slug,
			categoryId: data.categoryId,
			schemaDefinition: JSON.stringify(data.schemaDefinition ?? {}, null, 2),
		};
	}, [data]);

	const handleSubmit = useCallback(
		async (values: SchemaVersionSubmitPayload) => {
			if (!data) {
				return;
			}

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
		[createMutation, data, navigate],
	);

	return (
		<div className="flex flex-1 flex-col gap-6 p-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-semibold">Edit schema definition</h1>
					<p className="text-muted-foreground text-sm">
						A new version will be created and activated automatically.
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
						Review immutable attributes and update the JSON schema definition as
						needed.
					</CardDescription>
				</CardHeader>
				<CardContent>
					<SchemaVersionForm
						defaultValues={defaultValues}
						categories={categories}
						readOnlyFields={{ tableName: true, slug: true, category: true }}
						isSubmitting={createMutation.isPending}
						onSubmit={handleSubmit}
						className="max-w-4xl"
						submitLabel="Create new version"
					/>
					{isLoading && !data && (
						<p className="text-muted-foreground mt-4 text-sm">
							Loading schema version…
						</p>
					)}
				</CardContent>
			</Card>
		</div>
	);
}
