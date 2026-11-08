import { request, uploadFile } from './client'
import type { UploadOptions } from './client'
import {
  buildIndexResponseSchema,
  createdDatasetSchema,
  datasetListSchema,
  datasetSchema,
  healthSchema,
  indexStatsSchema,
  indexStatusSchema,
} from './schemas'
import type { BuildIndexInput, GenerateInput } from './schemas'

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
}
