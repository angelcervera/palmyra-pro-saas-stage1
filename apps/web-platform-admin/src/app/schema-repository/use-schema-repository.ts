import { useMemo } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { SchemaRepository } from '@zengateglobal/api-sdk'

import { schemaRepositoryClient } from '@/lib/api'

type CreatePayload = SchemaRepository.CreateSchemaVersionData['body']

const listAllKey = (includeInactive: boolean) => ['schema-repository', 'all', includeInactive] as const

export function useAllSchemaVersions(includeInactive = false) {
  const client = useMemo(() => schemaRepositoryClient(), [])

  return useQuery({
    queryKey: listAllKey(includeInactive),
    queryFn: async () => {
      const response: unknown = await SchemaRepository.listAllSchemaVersions({
        client,
        responseStyle: 'data',
        throwOnError: true,
        query: includeInactive ? { includeInactive: true } : undefined,
      })

      const payload = response as SchemaRepository.ListAllSchemaVersionsResponses[200]
      return payload?.items ?? []
    },
    refetchOnMount: 'always',
  })
}

export function useSchemaVersion(schemaId?: string, schemaVersion?: string) {
  const client = useMemo(() => schemaRepositoryClient(), [])

  return useQuery({
    queryKey: ['schema-repository', schemaId, schemaVersion],
    enabled: Boolean(schemaId && schemaVersion),
    queryFn: async () => {
      if (!schemaId || !schemaVersion) {
        return undefined
      }

      const response: unknown = await SchemaRepository.getSchemaVersion({
        client,
        responseStyle: 'data',
        throwOnError: true,
        path: { schemaId, schemaVersion },
      })

      return response as SchemaRepository.GetSchemaVersionResponses[200]
    },
  })
}

export function useCreateSchemaVersion() {
  const queryClient = useQueryClient()
  const client = useMemo(() => schemaRepositoryClient(), [])

  return useMutation({
    mutationFn: async (payload: CreatePayload) => {
      const response: unknown = await SchemaRepository.createSchemaVersion({
        client,
        responseStyle: 'data',
        throwOnError: true,
        body: payload,
      })

      return response as SchemaRepository.CreateSchemaVersionResponses[201]
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: listAllKey(false) })
      void queryClient.invalidateQueries({ queryKey: listAllKey(true) })
    },
  })
}
