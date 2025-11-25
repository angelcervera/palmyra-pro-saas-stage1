import { useMemo } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Users } from '@zengateglobal/api-sdk'
import { usersClient } from '@/lib/api'

export type UsersListQuery = NonNullable<Users.UsersListData['query']>

const DEFAULT_QUERY: UsersListQuery = { page: 1, pageSize: 10, sort: "-createdAt" }

export function useUsersList(query?: UsersListQuery) {
  const params = useMemo(
    () => ({ ...DEFAULT_QUERY, ...(query ?? {}) }),
    [query],
  )

  const client = useMemo(() => usersClient(), [])

  return useQuery<Users.UsersListResponses[200] | undefined>({
    queryKey: ['users', params],
    queryFn: async () => {
      const data: unknown = await Users.usersList({
        client,
        responseStyle: 'data',
        throwOnError: true,
        query: params,
      })

      return data as Users.UsersListResponses[200]
    },
    refetchOnMount: 'always',
  })
}

type CreateUserInput = Users.UsersCreateData['body']

export function useCreateUser() {
  const client = useMemo(() => usersClient(), [])
  const queryClient = useQueryClient()

  return useMutation<Users.UsersCreateResponses[201], unknown, CreateUserInput>({
    mutationFn: async (payload) => {
      const data: unknown = await Users.usersCreate({
        client,
        responseStyle: 'data',
        throwOnError: true,
        body: payload,
      })

      return data as Users.UsersCreateResponses[201]
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['users'] })
    },
  })
}

type DeleteUserPath = Users.UsersDeleteData['path']

export function useDeleteUser() {
  const client = useMemo(() => usersClient(), [])
  const queryClient = useQueryClient()

  return useMutation<void, unknown, string>({
    mutationFn: async (userId) => {
      if (!userId) {
        throw new Error('User id is required')
      }

      const path: DeleteUserPath = { userId }

      await Users.usersDelete({
        client,
        responseStyle: 'data',
        throwOnError: true,
        path,
      })
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['users'] })
    },
  })
}
