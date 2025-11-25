import {
  createSchemaCategoriesClient,
  createSchemaRepositoryClient,
  createUsersClient,
  createEntitiesClient,
} from '@zengateglobal/api-sdk'

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

export function usersClient() {
  const client = createUsersClient({
    baseUrl: BASE_URL,
    responseStyle: 'data',
  })

  client.interceptors.request.use((request: Request) => {
    const token = sessionStorage.getItem('jwt')
    if (!token) {
      return request
    }

    const headers = new Headers(request.headers)
    headers.set('Authorization', `Bearer ${token}`)
    return new Request(request, { headers })
  })

  return client
}

export function schemaCategoriesClient() {
  const client = createSchemaCategoriesClient({
    baseUrl: BASE_URL,
    responseStyle: 'data',
  })

  client.interceptors.request.use((request) => {
    const token = sessionStorage.getItem('jwt')
    if (!token) {
      return request
    }

    const headers = new Headers(request.headers)
    headers.set('Authorization', `Bearer ${token}`)
    return new Request(request, { headers })
  })

  return client
}

export function schemaRepositoryClient() {
  const client = createSchemaRepositoryClient({
    baseUrl: BASE_URL,
    responseStyle: 'data',
  })

  client.interceptors.request.use((request) => {
    const token = sessionStorage.getItem('jwt')
    if (!token) {
      return request
    }

    const headers = new Headers(request.headers)
    headers.set('Authorization', `Bearer ${token}`)
    return new Request(request, { headers })
  })

  return client
}

export function entitiesClient() {
  const client = createEntitiesClient({
    baseUrl: BASE_URL,
    responseStyle: 'data',
  })

  client.interceptors.request.use((request) => {
    const token = sessionStorage.getItem('jwt')
    if (!token) {
      return request
    }

    const headers = new Headers(request.headers)
    headers.set('Authorization', `Bearer ${token}`)
    return new Request(request, { headers })
  })

  return client
}
