import { useCallback, useEffect, useMemo, useState } from "react"
import { DataEditor, GridCellKind } from "@glideapps/glide-data-grid"
import type { GridCell, GridColumn, Item } from "@glideapps/glide-data-grid"
import "@glideapps/glide-data-grid/dist/index.css"
import { Button } from "@/components/ui/button"
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { Separator } from "@/components/ui/separator"

type RowObject = Record<string, unknown>
type DataBackedCell = Extract<GridCell, { data?: unknown }>
type TextCell = Extract<GridCell, { kind: GridCellKind.Text; data: string }>

export type EntitiesGridProps = {
  rows: RowObject[]
  onChange?: (rows: RowObject[]) => void
  primaryKey?: string
  readOnlyKeys?: string[]
}

function inferKind(value: unknown): GridCellKind {
  if (typeof value === "number") return GridCellKind.Number
  if (typeof value === "boolean") return GridCellKind.Boolean
  if (value === null || value === undefined) return GridCellKind.Text
  // Render short JSON for objects/arrays
  if (typeof value === "object") return GridCellKind.Text
  return GridCellKind.Text
}

export function EntitiesGrid({ rows: inputRows, onChange, primaryKey, readOnlyKeys = [] }: EntitiesGridProps) {
  const [rows, setRows] = useState<RowObject[]>(inputRows)

  useEffect(() => {
    setRows(inputRows)
  }, [inputRows])

  const allKeys = useMemo(() => {
    const keys = new Set<string>()
    for (const r of rows) {
      Object.keys(r).forEach((k) => {
        keys.add(k)
      })
    }
    return Array.from(keys)
  }, [rows])

  const columns: GridColumn[] = useMemo(
    () =>
      allKeys.map((k) => ({
        id: k,
        title: k,
        width: 200,
        // lock primary key or readOnlyKeys
        readonly: (primaryKey && k === primaryKey) || readOnlyKeys.includes(k),
      })),
    [allKeys, primaryKey, readOnlyKeys],
  )

  const getCellContent = useCallback(
    ([col, row]: Item): GridCell => {
      const key = columns[col]?.id as string
      const value = rows[row]?.[key]

      // Normalize complex values into short JSON strings for display/editing
      const normalize = (v: unknown) => {
        if (v === null || v === undefined) return ""
        if (typeof v === "object") return JSON.stringify(v)
        return String(v)
      }

      const kind = inferKind(value)

      switch (kind) {
        case GridCellKind.Number:
          return {
            kind: GridCellKind.Number,
            displayData: typeof value === "number" ? String(value) : normalize(value),
            data: typeof value === "number" ? value : Number(value ?? 0),
            allowOverlay: true,
          }
        case GridCellKind.Boolean:
          return {
            kind: GridCellKind.Boolean,
            data: Boolean(value),
            allowOverlay: false,
          }
        default:
          return {
            kind: GridCellKind.Text,
            data: normalize(value),
            displayData: normalize(value),
            allowOverlay: true,
          }
      }
    },
    [columns, rows],
  )

  const onCellEdited = useCallback(
    (cell: Item, newValue: GridCell) => {
      const [col, row] = cell
      const key = columns[col]?.id as string
      const next = rows.slice()
      const current = { ...(next[row] ?? {}) }

      switch (newValue.kind) {
        case GridCellKind.Number:
          current[key] = newValue.data
          break
        case GridCellKind.Boolean:
          current[key] = newValue.data
          break
        case GridCellKind.Text: {
          const textCell = newValue as TextCell
          const rawValue = textCell.data ?? ""
          try {
            current[key] = JSON.parse(rawValue)
          } catch {
            current[key] = rawValue
          }
          break
        }
        default: {
          const fallbackValue = extractCellData(newValue)
          current[key] = fallbackValue
          break
        }
      }

      next[row] = current
      setRows(next)
      onChange?.(next)
    },
    [columns, rows, onChange],
  )

  const addRow = useCallback(() => {
    const blank: RowObject = {}
    for (const c of columns) blank[c.id as string] = ""
    const next = [...rows, blank]
    setRows(next)
    onChange?.(next)
  }, [columns, rows, onChange])

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <Button size="sm" onClick={addRow} variant="outline">
          Add Row
        </Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button size="sm" variant="ghost">
              Columns
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="p-2 text-sm">
            <div className="mb-1 font-medium">Visible Columns</div>
            <Separator className="my-1" />
            <ul className="max-h-64 overflow-auto pr-2">
              {columns.map((c) => (
                <li key={c.id} className="text-muted-foreground">
                  {String(c.title)}
                </li>
              ))}
            </ul>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <div className="border rounded-md overflow-hidden">
        <DataEditor
          width="100%"
          height={520}
          columns={columns}
          getCellContent={getCellContent}
          rows={rows.length}
          onCellEdited={onCellEdited}
          smoothScrollX
          smoothScrollY
          rowMarkers={"number"}
          freezeColumns={primaryKey ? 1 : 0}
        />
      </div>
    </div>
  )
}

export default EntitiesGrid

function extractCellData(cell: GridCell) {
  if ("data" in cell) {
    const value = (cell as DataBackedCell).data
    if (typeof value === "string") {
      return value
    }
    if (value === null || value === undefined) {
      return ""
    }
    if (typeof value === "object") {
      try {
        return JSON.stringify(value)
      } catch {
        return String(value)
      }
    }
    return value
  }
  return ""
}
