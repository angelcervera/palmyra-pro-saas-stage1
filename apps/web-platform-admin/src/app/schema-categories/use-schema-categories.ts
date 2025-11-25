import { useMemo } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { SchemaCategories } from '@zengateglobal/api-sdk'

import { schemaCategoriesClient } from '@/lib/api'

type ListOptions = SchemaCategories.ListSchemaCategoriesData['query']
type CreatePayload = SchemaCategories.CreateSchemaCategoryData['body']
type UpdatePayload = SchemaCategories.UpdateSchemaCategoryData['body']

export function useSchemaCategories(options?: ListOptions) {
  const queryKey = ['schema-categories', options?.includeDeleted ?? false] as const
  const client = useMemo(() => schemaCategoriesClient(), [])

  return useQuery({
    queryKey,
    queryFn: async () => {
      const response: unknown = await SchemaCategories.listSchemaCategories({
        client,
        responseStyle: 'data',
        throwOnError: true,
        query: options,
      })

      const payload = response as SchemaCategories.ListSchemaCategoriesResponses[200]
      return payload?.items ?? []
    },
    refetchOnMount: 'always',
  })
}

export function useSchemaCategory(categoryId?: string) {
  const client = useMemo(() => schemaCategoriesClient(), [])

  return useQuery({
    queryKey: ['schema-category', categoryId],
    enabled: Boolean(categoryId),
    queryFn: async () => {
      if (!categoryId) {
        return undefined
      }

      const response: unknown = await SchemaCategories.getSchemaCategory({
        client,
        responseStyle: 'data',
        throwOnError: true,
        path: { categoryId },
      })

      return response as SchemaCategories.GetSchemaCategoryResponses[200]
    },
  })
}

export function useCreateSchemaCategory() {
  const queryClient = useQueryClient()
  const client = useMemo(() => schemaCategoriesClient(), [])

  return useMutation({
    mutationFn: async (payload: CreatePayload) => {
      const response: unknown = await SchemaCategories.createSchemaCategory({
        client,
        responseStyle: 'data',
        throwOnError: true,
        body: payload,
      })

      return response as SchemaCategories.CreateSchemaCategoryResponses[201]
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['schema-categories'] })
    },
  })
}

export function useUpdateSchemaCategory(categoryId: string) {
  const queryClient = useQueryClient()
  const client = useMemo(() => schemaCategoriesClient(), [])

  return useMutation({
    mutationFn: async (payload: UpdatePayload) => {
      const response: unknown = await SchemaCategories.updateSchemaCategory({
        client,
        responseStyle: 'data',
        throwOnError: true,
        path: { categoryId },
        body: payload,
      })

      return response as SchemaCategories.UpdateSchemaCategoryResponses[200]
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['schema-categories'] })
      void queryClient.invalidateQueries({ queryKey: ['schema-category', categoryId] })
    },
  })
}

export function useDeleteSchemaCategory() {
  const queryClient = useQueryClient()
  const client = useMemo(() => schemaCategoriesClient(), [])

  return useMutation({
    mutationFn: async (categoryId: string) => {
      await SchemaCategories.deleteSchemaCategory({
        client,
        responseStyle: 'data',
        throwOnError: true,
        path: { categoryId },
      })
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['schema-categories'] })
    },
  })
}
