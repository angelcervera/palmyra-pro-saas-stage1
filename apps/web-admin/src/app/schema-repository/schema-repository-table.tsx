import * as React from 'react'
import type { ColumnDef, RowSelectionState, VisibilityState } from '@tanstack/react-table'
import {
  flexRender,
  getCoreRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
  type SortingState,
} from '@tanstack/react-table'
import { IconArrowLeft, IconArrowRight, IconChevronDown, IconEye, IconPencil } from '@tabler/icons-react'
import { Link } from 'react-router-dom'

import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { SchemaRepository } from '@zengateglobal/api-sdk'

type SchemaVersion = SchemaRepository.SchemaVersion

type SchemaRepositoryTableProps = {
  data: SchemaVersion[]
  isLoading?: boolean
}

const OPTIONAL_COLUMNS = new Set(['schemaId', 'categoryId', 'createdAt', 'isSoftDeleted'])

export function SchemaRepositoryTable({ data, isLoading = false }: SchemaRepositoryTableProps) {
  const [rowSelection, setRowSelection] = React.useState<RowSelectionState>({})
  const [columnVisibility, setColumnVisibility] = React.useState<VisibilityState>({
    schemaId: false,
    categoryId: false,
    createdAt: false,
    isSoftDeleted: false,
  })
  const [sorting, setSorting] = React.useState<SortingState>([])

  const columns = React.useMemo<ColumnDef<SchemaVersion, unknown>[]>(() => {
    return [
      {
        id: 'select',
        header: ({ table }) => (
          <Checkbox
            checked={table.getIsAllPageRowsSelected() || (table.getIsSomePageRowsSelected() && 'indeterminate')}
            onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
            aria-label="Select all"
          />
        ),
        cell: ({ row }) => (
          <Checkbox
            checked={row.getIsSelected()}
            onCheckedChange={(value) => row.toggleSelected(!!value)}
            aria-label="Select row"
          />
        ),
        enableHiding: false,
        enableSorting: false,
        size: 32,
      },
      {
        accessorKey: 'tableName',
        header: 'Table Name',
        cell: ({ row }) => (
          <span className="font-mono text-sm text-foreground">{row.original.tableName}</span>
        ),
        enableHiding: false,
      },
      {
        accessorKey: 'schemaVersion',
        header: 'Version',
        cell: ({ row }) => <span className="font-medium">{row.original.schemaVersion}</span>,
        enableHiding: false,
      },
      {
        accessorKey: 'isActive',
        header: 'Status',
        cell: ({ row }) =>
          row.original.isActive ? (
            <Badge className="bg-emerald-100 text-emerald-900 hover:bg-emerald-200">Active</Badge>
          ) : (
            <Badge variant="outline" className="bg-muted text-muted-foreground">
              Inactive
            </Badge>
          ),
        enableHiding: false,
      },
      {
        accessorKey: 'slug',
        header: 'Slug',
        cell: ({ row }) => row.original.slug,
        enableHiding: false,
      },
      {
        accessorKey: 'schemaId',
        header: 'Schema ID',
        cell: ({ row }) => (
          <span className="break-all text-xs text-muted-foreground">{row.original.schemaId}</span>
        ),
        enableHiding: true,
      },
      {
        accessorKey: 'categoryId',
        header: 'Category ID',
        cell: ({ row }) => (
          <span className="break-all text-xs text-muted-foreground">{row.original.categoryId}</span>
        ),
        enableHiding: true,
      },
      {
        accessorKey: 'createdAt',
        header: 'Created',
        cell: ({ row }) => formatDate(row.original.createdAt),
        enableHiding: true,
      },
      {
        accessorKey: 'isSoftDeleted',
        header: 'Soft deleted',
        cell: ({ row }) =>
          row.original.isSoftDeleted ? (
            <Badge variant="destructive" className="bg-red-50 text-red-900 hover:bg-red-100">
              Hidden
            </Badge>
          ) : (
            <Badge variant="outline" className="text-muted-foreground">
              Visible
            </Badge>
          ),
        enableHiding: true,
      },
      {
        id: 'actions',
        enableHiding: false,
        enableSorting: false,
        cell: ({ row }) => {
          const schemaVersion = row.original
          return (
            <div className="flex items-center justify-end gap-2">
              <Button asChild variant="ghost" size="sm">
                <Link to={`/schema-repository/${schemaVersion.schemaId}/versions/${schemaVersion.schemaVersion}`}>
                  <IconEye className="mr-1 size-4" />
                  View
                </Link>
              </Button>
              <Button asChild variant="ghost" size="sm">
                <Link to={`/schema-repository/${schemaVersion.schemaId}/versions/${schemaVersion.schemaVersion}/edit`}>
                  <IconPencil className="mr-1 size-4" />
                  Edit
                </Link>
              </Button>
            </div>
          )
        },
      },
    ]
  }, [])

  const table = useReactTable({
    data,
    columns,
    state: {
      rowSelection,
      columnVisibility,
      sorting,
    },
    enableRowSelection: true,
    onRowSelectionChange: setRowSelection,
    onColumnVisibilityChange: setColumnVisibility,
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
  })

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="text-sm text-muted-foreground">
          {data.length > 0 ? `${data.length} version(s)` : 'No versions loaded yet.'}
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm" className="gap-2">
                <IconChevronDown className="size-4" />
                Customize columns
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-56">
              <DropdownMenuItem className="pointer-events-none flex items-center gap-2 text-xs uppercase tracking-wide text-muted-foreground">
                Visible columns
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              {table
                .getAllColumns()
                .filter((column) => column.getCanHide() && OPTIONAL_COLUMNS.has(column.id ?? ''))
                .map((column) => (
                  <DropdownMenuCheckboxItem
                    key={column.id}
                    className="capitalize"
                    checked={column.getIsVisible()}
                    onCheckedChange={(value) => column.toggleVisibility(!!value)}
                  >
                    {column.id}
                  </DropdownMenuCheckboxItem>
                ))}
            </DropdownMenuContent>
          </DropdownMenu>
          <Button asChild size="sm">
            <Link to="/schema-repository/new">Add schema version</Link>
          </Button>
        </div>
      </div>

      <div className="overflow-hidden rounded-lg border">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead key={header.id} className="align-middle text-sm font-medium">
                    {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {isLoading ? (
              Array.from({ length: 3 }).map((_, index) => (
                <TableRow key={`skeleton-${index}`}>
                  {table.getAllLeafColumns().map((column) => (
                    <TableCell key={`skeleton-cell-${column.id}`} className="animate-pulse">
                      <div className="h-4 w-full rounded-md bg-muted" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : table.getRowModel().rows.length ? (
              table.getRowModel().rows.map((row) => (
                <TableRow key={row.id} data-state={row.getIsSelected() && 'selected'}>
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={columns.length} className="h-24 text-center text-muted-foreground">
                  No schema versions found for this schema.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="text-sm text-muted-foreground">
          {table.getFilteredSelectedRowModel().rows.length} of {table.getFilteredRowModel().rows.length} row(s) selected
        </div>
       <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            className="gap-2"
            onClick={() => table.previousPage()}
            disabled={!table.getCanPreviousPage()}
          >
            <IconArrowLeft className="size-4" />
            Previous
          </Button>
          <span className="text-sm font-medium">
            Page {table.getState().pagination.pageIndex + 1} of {Math.max(table.getPageCount(), 1)}
          </span>
          <Button
            variant="outline"
            size="sm"
            className="gap-2"
            onClick={() => table.nextPage()}
            disabled={!table.getCanNextPage()}
          >
            Next
            <IconArrowRight className="size-4" />
          </Button>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">Rows per page</span>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm" className="w-16">
                {table.getState().pagination.pageSize}
                <IconChevronDown className="ml-1 size-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-32">
              {[10, 20, 30, 40, 50].map((pageSize) => (
                <DropdownMenuItem
                  key={pageSize}
                  onClick={() => table.setPageSize(pageSize)}
                  className={cn(table.getState().pagination.pageSize === pageSize && 'font-semibold')}
                >
                  {pageSize}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </div>
  )
}

function formatDate(value?: string | null) {
  if (!value) {
    return '—'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}
