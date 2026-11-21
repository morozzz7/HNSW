import { useMutation, useQuery } from '@tanstack/react-query'
import { api } from '../endpoints'
import { shouldRetry } from '../errors'
import { queryKeys } from '../keys'
import type { BenchmarkInput } from '../schemas'

const POLL_INTERVAL_MS = 800

export function useRunBenchmark() {
  return useMutation({
    mutationFn: (input: BenchmarkInput) => api.runBenchmark(input),
  })
}

export function useBenchmarkJob(id: string | null) {
  return useQuery({
    queryKey: queryKeys.benchmarkJob(id ?? ''),
    queryFn: ({ signal }) => api.benchmarkJob(id ?? '', signal),
    enabled: Boolean(id),
    retry: shouldRetry,
    refetchOnWindowFocus: false,
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === 'queued' || status === 'running' ? POLL_INTERVAL_MS : false
    },
  })
}
