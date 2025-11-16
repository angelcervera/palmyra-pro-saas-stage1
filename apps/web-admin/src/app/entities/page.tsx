import { Entities, SchemaRepository } from "@zengateglobal/api-sdk"
import { useCallback, useEffect, useMemo, useState } from "react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { entitiesClient, schemaRepositoryClient } from "@/lib/api"
import EntitiesGrid from "./entities-grid"

// Demo page for an Excel-like JSON entities editor using Glide Data Grid.
// Contract-first note: there is no Entities API in contracts yet. This page
// demonstrates editing on the client. Wire up to the future endpoints once
// the domain/paths exist.

type Row = Record<string, unknown>
type EntityDocument = Entities.EntityDocument

const METADATA_KEYS = ["entityId", "schemaId", "schemaVersion", "createdAt", "updatedAt", "deletedAt", "isActive"]

const READ_ONLY_KEYS = [...METADATA_KEYS]

const hasPayloadData = (payload: Record<string, unknown>) =>
  Object.values(payload).some((value) => value !== undefined && value !== null && value !== "")

const extractPayload = (row: Row) => {
  const payload: Record<string, unknown> = {}
  for (const [key, value] of Object.entries(row)) {
    if (!METADATA_KEYS.includes(key)) {
      payload[key] = value
    }
  }
  return payload
}

const payloadsEqual = (a: Record<string, unknown>, b: Record<string, unknown>) => {
  return JSON.stringify(a) === JSON.stringify(b)
}

export default function EntitiesPage() {
  const [rows, setRows] = useState<Row[]>([])
  const [baselineRows, setBaselineRows] = useState<Row[]>([])
  const [tableName, setTableName] = useState<string>("")
  const [tableOptions, setTableOptions] = useState<string[]>([])
  const [tablesLoading, setTablesLoading] = useState(false)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)

  const primaryKey = useMemo(() => "entityId", [])

  const baselineMap = useMemo(() => {
    const map = new Map<string, Row>()
    baselineRows.forEach((row) => {
      const entityId = row.entityId
      if (typeof entityId === "string") {
        map.set(entityId, row)
      }
    })
    return map
  }, [baselineRows])

  const fetchTables = useCallback(async () => {
    setTablesLoading(true)
    try {
      const client = schemaRepositoryClient()
      const raw: unknown = await SchemaRepository.listAllSchemaVersions({
        client,
        responseStyle: 'data',
        throwOnError: true,
      })
      const payload = raw as SchemaRepository.ListAllSchemaVersionsResponses[200]
      const activeTables = Array.from(
        new Set(
          (payload?.items ?? [])
            .filter((item) => item.isActive)
            .map((item) => item.tableName)
            .filter((value): value is string => Boolean(value)),
        ),
      ).sort((a, b) => a.localeCompare(b))

      setTableOptions(activeTables)
      setTableName((prev) => prev || activeTables[0] || '')
    } catch (error) {
      console.error(error)
      toast.error('Failed to load table options')
    } finally {
      setTablesLoading(false)
    }
  }, [])

  const load = useCallback(async () => {
    if (!tableName.trim()) {
      return
    }
    setLoading(true)
    try {
      const client = entitiesClient()
      const raw: unknown = await Entities.listDocuments({
        client,
        responseStyle: 'data',
        throwOnError: true,
        path: { tableName },
        query: { page: 1, pageSize: 20 },
      })
      const result = raw as Entities.ListDocumentsResponses[200]
      const items: EntityDocument[] = result?.items ?? []
      // Flatten to grid: show payload fields at top-level but keep metadata
      const flattened = items.map((item) => {
        const deletedAt =
          "deletedAt" in item && typeof (item as { deletedAt?: unknown }).deletedAt === "string"
            ? (item as { deletedAt?: string }).deletedAt
            : undefined
        const updatedAt =
          "updatedAt" in item && typeof (item as { updatedAt?: unknown }).updatedAt === "string"
            ? (item as { updatedAt?: string }).updatedAt
            : undefined
        return {
          entityId: item.entityId,
          schemaId: item.schemaId,
          schemaVersion: item.schemaVersion,
          createdAt: item.createdAt,
          updatedAt,
          deletedAt,
          isActive: item.isActive,
          ...item.payload,
        }
      })
      setRows(flattened)
      setBaselineRows(flattened)
    } catch (e) {
      console.error(e)
      toast.error("Failed to load entities")
    } finally {
      setLoading(false)
    }
  }, [tableName])

  useEffect(() => {
    void fetchTables()
  }, [fetchTables])

  useEffect(() => {
    if (tableName) {
      void load()
    }
  }, [load, tableName])

  const hasChanges = useMemo(() => {
    return rows.some((row) => {
      const entityId = typeof row.entityId === "string" ? row.entityId : undefined
      const payload = extractPayload(row)

      if (!entityId) {
        return hasPayloadData(payload)
      }

      const baseline = baselineMap.get(entityId)
      if (!baseline) return true
      return !payloadsEqual(payload, extractPayload(baseline))
    })
  }, [rows, baselineMap])

  const handleSave = async () => {
    if (!tableName) {
      toast.error('Select a table before saving.')
      return
    }

    if (!hasChanges) {
      toast.info("No changes to save.")
      return
    }

    setSaving(true)
    try {
      const client = entitiesClient()
      const createPromises: Promise<unknown>[] = []
      const updatePromises: Promise<unknown>[] = []

      for (const row of rows) {
        const payload = extractPayload(row)
        if (!hasPayloadData(payload)) {
          continue
        }

        const entityId = typeof row.entityId === "string" ? row.entityId : undefined
        if (!entityId) {
          createPromises.push(
            Entities.createDocument({
              client,
              responseStyle: 'data',
              throwOnError: true,
              path: { tableName },
              body: { payload },
            }),
          )
          continue
        }

        const baseline = baselineMap.get(entityId)
        if (baseline && payloadsEqual(payload, extractPayload(baseline))) {
          continue
        }

        updatePromises.push(
          Entities.updateDocument({
            client,
            responseStyle: 'data',
            throwOnError: true,
            path: { tableName, entityId },
            body: { payload },
          }),
        )
      }

      await Promise.all([...createPromises, ...updatePromises])
      toast.success("Changes saved")
      await load()
    } catch (error) {
      console.error(error)
      toast.error("Failed to save changes")
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="flex flex-1 flex-col gap-6 p-6">
      <div>
        <h1 className="text-2xl font-semibold">Entities</h1>
        <p className="text-muted-foreground text-sm">
          Inspect and edit JSON documents stored per schema table.
        </p>
      </div>
      <Card>
        <CardHeader className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
          <div>
            <CardTitle>Entities Grid</CardTitle>
            <CardDescription>
              Choose an active table to load documents, then edit inline and save.
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            <Select
              value={tableName || undefined}
              onValueChange={(value) => setTableName(value)}
              disabled={tablesLoading || tableOptions.length === 0}
            >
              <SelectTrigger className="w-64">
                <SelectValue placeholder={tablesLoading ? "Loading tables..." : "Select a table"} />
              </SelectTrigger>
              <SelectContent>
                {tableOptions.map((option) => (
                  <SelectItem key={option} value={option}>
                    {option}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button size="sm" variant="outline" onClick={load} disabled={loading || !tableName}>
              {loading ? "Loading…" : "Reload"}
            </Button>
            <Button size="sm" onClick={handleSave} disabled={saving || !hasChanges || !tableName}>
              {saving ? "Saving…" : "Save"}
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <EntitiesGrid
            rows={rows}
            onChange={setRows}
            primaryKey={primaryKey}
            readOnlyKeys={READ_ONLY_KEYS}
          />
        </CardContent>
      </Card>
    </div>
  )
}
