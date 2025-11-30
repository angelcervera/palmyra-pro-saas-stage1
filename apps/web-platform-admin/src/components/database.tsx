import type { ColumnDef, VisibilityState } from "@tanstack/react-table";
import type { ComponentProps } from "react";
import type { z } from "zod";

import {
	DataTable,
	DragHandle,
	defaultColumns,
	schema,
} from "@/components/data-table";
import { Checkbox } from "@/components/ui/checkbox";

export type DatabaseRow = z.infer<typeof schema>;

export function Database({
	columns = defaultColumns,
	defaultColumnVisibility,
	...props
}: ComponentProps<typeof DataTable>) {
	return (
		<DataTable
			columns={columns}
			defaultColumnVisibility={defaultColumnVisibility}
			{...props}
		/>
	);
}

export const databaseSchema = schema;

export const userColumns: ColumnDef<DatabaseRow>[] = [
	{
		id: "drag",
		header: () => null,
		cell: ({ row }) => <DragHandle id={row.original.id} />,
		enableHiding: false,
	},
	{
		id: "select",
		header: ({ table }) => (
			<div className="flex items-center justify-center">
				<Checkbox
					checked={
						table.getIsAllPageRowsSelected() ||
						(table.getIsSomePageRowsSelected() && "indeterminate")
					}
					onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
					aria-label="Select all"
				/>
			</div>
		),
		cell: ({ row }) => (
			<div className="flex items-center justify-center">
				<Checkbox
					checked={row.getIsSelected()}
					onCheckedChange={(value) => row.toggleSelected(!!value)}
					aria-label="Select row"
				/>
			</div>
		),
		enableSorting: false,
		enableHiding: false,
	},
	{
		accessorKey: "header",
		header: "Full name",
		cell: ({ row }) => (
			<div className="font-medium text-foreground">
				{row.original.header || "—"}
			</div>
		),
		enableHiding: false,
	},
	{
		accessorKey: "target",
		header: "Email",
		cell: ({ row }) => (
			<div className="text-muted-foreground">{row.original.target || "—"}</div>
		),
		enableHiding: false,
	},
	{
		id: "joined",
		accessorKey: "limit",
		header: "Joined",
		cell: ({ row }) => (
			<div className="text-muted-foreground">{row.original.limit || "—"}</div>
		),
		enableHiding: true,
	},
	{
		id: "userId",
		accessorKey: "status",
		header: "ID",
		cell: ({ row }) => (
			<div className="text-muted-foreground break-all">
				{String(row.original.status || "—")}
			</div>
		),
		enableHiding: true,
	},
];

export const userColumnVisibility: VisibilityState = {
	joined: false,
	userId: false,
};
