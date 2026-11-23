import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { api } from '../endpoints'
import { shouldRetry } from '../errors'
import { queryKeys } from '../keys'
import type { GraphResponse } from '../schemas'

export interface UseGraphOptions {
  level: number
  limit?: number
  around?: string
  enabled?: boolean
}

export function useGraphQuery({ level, limit = 500, around, enabled = true }: UseGraphOptions) {
  return useQuery<GraphResponse>({
    queryKey: queryKeys.graph(level, limit, around),
    queryFn: ({ signal }) => api.graph({ level, limit, around }, signal),
    enabled,
    retry: shouldRetry,
    staleTime: 60_000,
    placeholderData: keepPreviousData,
  })
}
