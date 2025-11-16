import { useEffect, useMemo } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

import { useSchemaCategories } from '../../schema-categories/use-schema-categories'
import { useSchemaVersion } from '../use-schema-repository'

export default function SchemaVersionDetailPage() {
  const navigate = useNavigate()
  const params = useParams()
  const schemaId = params.schemaId ?? ''
  const schemaVersion = params.schemaVersion ?? ''

  const { data, isLoading, isError, error } = useSchemaVersion(schemaId, schemaVersion)
  const { data: categories } = useSchemaCategories()

  useEffect(() => {
    if (isError) {
      const message =
        error instanceof Error ? error.message : 'Unable to load the schema version. Please try again.'
      toast.error(message)
    }
  }, [isError, error])

  const categoryLabel = useMemo(() => {
    if (!data || !categories) {
      return data?.categoryId ?? '—'
    }
    const match = categories.find((category) => category.categoryId === data.categoryId)
    return match?.name ?? match?.slug ?? data.categoryId
  }, [categories, data])

  const jsonBody = useMemo(() => {
    if (!data) {
      return '{ }'
    }
    return JSON.stringify(data.schemaDefinition, null, 2)
  }, [data])

  return (
    <div className="flex flex-1 flex-col gap-6 p-6">
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold">Schema version details</h1>
          <p className="text-muted-foreground">
            Inspect the stored JSON schema and metadata for this version.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="ghost" onClick={() => navigate(-1)}>
            Back
          </Button>
          <Button asChild variant="outline" disabled={!data || isLoading}>
            <Link to={`/schema-repository/${schemaId}/versions/${schemaVersion}/edit`}>Edit definition</Link>
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader className="gap-2">
          <CardTitle>
            Version {schemaVersion}{' '}
            {data?.isActive ? (
              <Badge className="ml-2 bg-emerald-100 text-emerald-900 hover:bg-emerald-200">Active</Badge>
            ) : (
              <Badge variant="outline" className="ml-2 bg-muted text-muted-foreground">
                Inactive
              </Badge>
            )}
          </CardTitle>
          <CardDescription>Metadata persisted in the schema repository.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {isLoading && <p className="text-muted-foreground text-sm">Loading schema version…</p>}
          {!isLoading && !data && (
            <p className="text-muted-foreground text-sm">
              Schema version not found. It may have been deleted.
            </p>
          )}
          {data && (
            <div className="space-y-4">
              <dl className="grid gap-4 sm:grid-cols-2">
                <MetadataItem label="Schema ID" value={data.schemaId} />
                <MetadataItem label="Table name" value={data.tableName} monospace />
                <MetadataItem label="Slug" value={data.slug} />
                <MetadataItem label="Category" value={categoryLabel} />
                <MetadataItem label="Created at" value={formatDateTime(data.createdAt)} />
                <MetadataItem label="Active" value={formatBoolean(data.isActive)} />
                <MetadataItem label="Soft deleted" value={formatBoolean(data.isSoftDeleted)} />
              </dl>
              <div className="space-y-2">
                <h2 className="text-lg font-semibold">Schema definition</h2>
                <pre className="rounded-lg border bg-muted p-4 text-sm leading-relaxed">
                  <code>{jsonBody}</code>
                </pre>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function MetadataItem({
  label,
  value,
  monospace = false,
}: {
  label: string
  value: string | undefined | null
  monospace?: boolean
}) {
  return (
    <div className="space-y-1">
      <dt className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{label}</dt>
      <dd className={monospace ? 'font-mono text-sm text-foreground' : 'text-sm text-foreground'}>
        {value && value !== 'Invalid Date' ? value : '—'}
      </dd>
    </div>
  )
}

function formatDateTime(value?: string | null) {
  if (!value) {
    return '—'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatBoolean(value?: boolean | null) {
  if (value === undefined || value === null) {
    return '—'
  }
  return value ? 'Yes' : 'No'
}
