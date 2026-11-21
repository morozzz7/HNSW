import { request, uploadFile } from './client'
import type { UploadOptions } from './client'
import {
  benchmarkJobSchema,
  benchmarkRunResponseSchema,
  buildIndexResponseSchema,
  createdDatasetSchema,
  datasetListSchema,
  datasetSchema,
  graphResponseSchema,
  healthSchema,
  indexStatsSchema,
  indexStatusSchema,
  searchResponseSchema,
} from './schemas'
import type {
  BenchmarkInput,
  BenchmarkJob,
  BenchmarkRunResponse,
  BuildIndexInput,
  GenerateInput,
  GraphResponse,
  SearchInput,
  SearchResponse,
} from './schemas'

export const api = {
  health(signal?: AbortSignal) {
    return request('/health', healthSchema, { signal })
  },

  listDatasets(signal?: AbortSignal) {
    return request('/datasets', datasetListSchema, { signal })
  },

  getDataset(id: string, signal?: AbortSignal) {
    return request(`/datasets/${encodeURIComponent(id)}`, datasetSchema, { signal })
  },

  generateDataset(input: GenerateInput, signal?: AbortSignal) {
    return request('/datasets', createdDatasetSchema, {
      method: 'POST',
      body: { action: 'generate', ...input },
      signal,
    })
  },

  uploadDataset(file: File, options?: UploadOptions) {
    return uploadFile('/datasets', file, createdDatasetSchema, options)
  },

  buildIndex(input: BuildIndexInput, signal?: AbortSignal) {
    return request('/index', buildIndexResponseSchema, {
      method: 'POST',
      body: input,
      signal,
    })
  },

  indexStatus(signal?: AbortSignal) {
    return request('/index/status', indexStatusSchema, { signal })
  },

  indexStats(signal?: AbortSignal) {
    return request('/index/stats', indexStatsSchema, { signal })
  },

  search(input: SearchInput, signal?: AbortSignal): Promise<SearchResponse> {
    return request('/search', searchResponseSchema, {
      method: 'POST',
      body: input,
      signal,
    })
  },

  graph(
    params: { level?: number; limit?: number; around?: string } = {},
    signal?: AbortSignal,
  ): Promise<GraphResponse> {
    const searchParams = new URLSearchParams()
    if (params.level !== undefined) {
      searchParams.set('level', String(params.level))
    }
    if (params.limit !== undefined) {
      searchParams.set('limit', String(params.limit))
    }
    if (params.around !== undefined && params.around !== '') {
      searchParams.set('around', params.around)
    }
    const query = searchParams.toString()
    const path = query.length > 0 ? `/graph?${query}` : '/graph'
    return request(path, graphResponseSchema, { signal })
  },

  runBenchmark(input: BenchmarkInput, signal?: AbortSignal): Promise<BenchmarkRunResponse> {
    return request('/benchmark', benchmarkRunResponseSchema, {
      method: 'POST',
      body: input,
      signal,
    })
  },

  benchmarkJob(id: string, signal?: AbortSignal): Promise<BenchmarkJob> {
    return request(`/benchmark/${encodeURIComponent(id)}`, benchmarkJobSchema, { signal })
  },
}
