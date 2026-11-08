import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../endpoints'
import { shouldRetry } from '../errors'
import { queryKeys } from '../keys'
import type { BuildIndexInput } from '../schemas'

export function useIndexStatus() {
  return useQuery({
    queryKey: queryKeys.indexStatus,
    queryFn: ({ signal }) => api.indexStatus(signal),
    retry: shouldRetry,
    refetchInterval: (query) => (query.state.data?.status === 'building' ? 1_200 : false),
  })
}

export function useIndexStats(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.indexStats,
    queryFn: ({ signal }) => api.indexStats(signal),
    enabled,
    retry: shouldRetry,
  })
}

export function useBuildIndex() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: BuildIndexInput) => api.buildIndex(input),
    onSuccess: () => {
      queryClient.removeQueries({ queryKey: queryKeys.indexStats })
      void queryClient.invalidateQueries({ queryKey: queryKeys.indexStatus })
    },
  })
}

export function useHealth() {
  return useQuery({
    queryKey: queryKeys.health,
    queryFn: ({ signal }) => api.health(signal),
    refetchInterval: 30_000,
    retry: false,
    staleTime: 15_000,
  })
}
