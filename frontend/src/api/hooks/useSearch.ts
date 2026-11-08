import { useMutation } from '@tanstack/react-query'
import { api } from '../endpoints'
import type { SearchInput, SearchResponse, UISearchMode } from '../schemas'

export interface ComparisonMetrics {
  recall: number
  intersectionCount: number
  intersectionIds: Set<number>
  hnswDurationMs: number
  bfDurationMs: number
  speedup: number
}

export interface SearchOutcome {
  mode: UISearchMode
  hnsw?: SearchResponse
  bruteforce?: SearchResponse
  comparison?: ComparisonMetrics
}

export interface SearchMutationVariables {
  input: SearchInput
  uiMode: UISearchMode
}

export function useSearchMutation() {
  return useMutation({
    mutationFn: async ({ input, uiMode }: SearchMutationVariables): Promise<SearchOutcome> => {
      if (uiMode === 'both') {
        const t0Hnsw = performance.now()
        const hnswPromise = api
          .search({ ...input, mode: 'hnsw' })
          .then((res) => ({ res, durationMs: performance.now() - t0Hnsw }))

        const t0Bf = performance.now()
        const bfPromise = api
          .search({ ...input, mode: 'bruteforce', trace: false })
          .then((res) => ({ res, durationMs: performance.now() - t0Bf }))

        const [hnswResult, bfResult] = await Promise.all([hnswPromise, bfPromise])

        const hnswIds = new Set(hnswResult.res.results.map((r) => r.id))
        const bfIds = new Set(bfResult.res.results.map((r) => r.id))
        const intersectionIds = new Set<number>()
        for (const id of hnswIds) {
          if (bfIds.has(id)) {
            intersectionIds.add(id)
          }
        }

        const k = input.k
        const recall = k > 0 ? intersectionIds.size / k : 0
        const speedup =
          hnswResult.durationMs > 0
            ? bfResult.durationMs / Math.max(0.01, hnswResult.durationMs)
            : 1

        return {
          mode: 'both',
          hnsw: hnswResult.res,
          bruteforce: bfResult.res,
          comparison: {
            recall,
            intersectionCount: intersectionIds.size,
            intersectionIds,
            hnswDurationMs: hnswResult.durationMs,
            bfDurationMs: bfResult.durationMs,
            speedup,
          },
        }
      }

      if (uiMode === 'bruteforce') {
        const res = await api.search({ ...input, mode: 'bruteforce', trace: false })
        return { mode: 'bruteforce', bruteforce: res }
      }

      const res = await api.search({ ...input, mode: 'hnsw' })
      return { mode: 'hnsw', hnsw: res }
    },
  })
}
