import { useEffect, useMemo, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { toast } from 'sonner'

import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'

import { SchemaRepositoryTable } from './schema-repository-table'
import { useAllSchemaVersions } from './use-schema-repository'

export default function SchemaRepositoryPage() {
  const location = useLocation()
  const [onlyActive, setOnlyActive] = useState(false)
  const { data, isLoading, isError, error } = useAllSchemaVersions(true)

  useEffect(() => {
    const state = location.state as
      | {
          toast?: {
            type: 'success' | 'error' | 'info' | 'warning'
            message: string
          }
        }
      | undefined

    if (!state?.toast) {
      return
    }

    const { type, message } = state.toast
    switch (type) {
      case 'success':
        toast.success(message)
        break
      case 'error':
        toast.error(message)
        break
      case 'warning':
        toast.warning(message)
        break
      default:
        toast.info(message)
        break
    }
  }, [location])

  useEffect(() => {
    if (isError) {
      const message =
        error instanceof Error ? error.message : 'Failed to load schema versions. Please try again.'
      toast.error(message)
    }
  }, [isError, error])

  const versions = useMemo(() => {
    const items = (data ?? []).filter((version) => (!onlyActive ? true : version.isActive))
    return [...items].sort((a, b) => {
      const tableComparison = a.tableName.localeCompare(b.tableName)
      if (tableComparison !== 0) {
        return tableComparison
      }

      return compareSemanticVersionStrings(b.schemaVersion, a.schemaVersion)
    })
  }, [data, onlyActive])

  return (
    <div className="flex flex-1 flex-col gap-6 p-6">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Schema Repository</h1>
          <p className="text-muted-foreground text-sm">
            Inspect and manage every JSON schema version stored in the persistent layer.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Checkbox id="onlyActive" checked={onlyActive} onCheckedChange={(checked) => setOnlyActive(!!checked)} />
          <Label htmlFor="onlyActive" className="text-sm font-medium">
            Only activated versions
          </Label>
        </div>
      </div>

      <div className="flex-1">
        <SchemaRepositoryTable data={versions} isLoading={isLoading} />
      </div>
    </div>
  )
}

function compareSemanticVersionStrings(a: string, b: string) {
  const [aMajor, aMinor, aPatch] = a.split('.').map(Number)
  const [bMajor, bMinor, bPatch] = b.split('.').map(Number)

  if (aMajor !== bMajor) {
    return aMajor - bMajor
  }
  if (aMinor !== bMinor) {
    return aMinor - bMinor
  }
  return aPatch - bPatch
}
