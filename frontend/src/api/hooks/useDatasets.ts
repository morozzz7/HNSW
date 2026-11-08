import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../endpoints'
import { shouldRetry } from '../errors'
import { queryKeys } from '../keys'
import type { GenerateInput } from '../schemas'

export function useDatasets() {
  return useQuery({
    queryKey: queryKeys.datasets,
    queryFn: ({ signal }) => api.listDatasets(signal),
    retry: shouldRetry,
  })
}

export function useDataset(id: string | null) {
  return useQuery({
    queryKey: queryKeys.dataset(id ?? ''),
    queryFn: ({ signal }) => api.getDataset(id ?? '', signal),
    enabled: Boolean(id),
    retry: shouldRetry,
  })
}

export function useGenerateDataset() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: GenerateInput) => api.generateDataset(input),
    onSuccess: (dataset) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.datasets })
      void queryClient.invalidateQueries({ queryKey: queryKeys.dataset(dataset.id) })
    },
  })
}

export interface UploadDatasetVariables {
  file: File
  onProgress?: (fraction: number) => void
}

export function useUploadDataset() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ file, onProgress }: UploadDatasetVariables) =>
      api.uploadDataset(file, { onProgress }),
    onSuccess: (dataset) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.datasets })
      void queryClient.invalidateQueries({ queryKey: queryKeys.dataset(dataset.id) })
    },
  })
}
