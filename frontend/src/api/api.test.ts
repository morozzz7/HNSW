import { afterEach, describe, expect, it, vi } from 'vitest'
import { request } from './client'
import { ApiError, errorMessage, shouldRetry } from './errors'
import {
  benchmarkInputSchema,
  benchmarkResultSchema,
  buildIndexInputSchema,
  datasetListSchema,
  generateInputSchema,
  healthSchema,
  indexStatsSchema,
} from './schemas'

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('schemas', () => {
  it('parses a dataset list', () => {
    const parsed = datasetListSchema.parse({
      datasets: [{ id: 'ds_1', n: 1000, dim: 16, createdAt: '2026-11-08T10:00:00Z' }],
    })
    expect(parsed.datasets[0]?.id).toBe('ds_1')
  })

  it('rejects a dataset list without datasets', () => {
    expect(datasetListSchema.safeParse({ items: [] }).success).toBe(false)
  })

  it('enforces backend index parameters', () => {
    expect(
      buildIndexInputSchema.safeParse({
        datasetId: 'ds_1',
        M: 1,
        efConstruction: 200,
        metric: 'l2',
        heuristic: true,
        seed: 0,
      }).success,
    ).toBe(false)
    expect(
      buildIndexInputSchema.safeParse({
        datasetId: 'ds_1',
        M: 16,
        efConstruction: 0,
        metric: 'l2',
        heuristic: true,
        seed: 0,
      }).success,
    ).toBe(false)
    expect(
      buildIndexInputSchema.safeParse({
        datasetId: '',
        M: 16,
        efConstruction: 200,
        metric: 'l2',
        heuristic: true,
        seed: 0,
      }).success,
    ).toBe(false)
    expect(
      buildIndexInputSchema.safeParse({
        datasetId: 'ds_1',
        M: 16,
        efConstruction: 200,
        metric: 'l2',
        heuristic: true,
        seed: 0,
      }).success,
    ).toBe(true)
  })

  it('rejects generation beyond the component budget', () => {
    const result = generateInputSchema.safeParse({
      n: 1_000_000,
      dim: 8192,
      distribution: 'uniform',
      seed: 1,
    })
    expect(result.success).toBe(false)
  })

  it('validates benchmark grid size and limits', () => {
    const base = {
      M: [16],
      efConstruction: [100],
      efSearch: [10, 50],
      k: 10,
      queries: 100,
    }
    expect(benchmarkInputSchema.safeParse(base).success).toBe(true)
    expect(
      benchmarkInputSchema.safeParse({
        ...base,
        efSearch: Array.from({ length: 257 }, (_, index) => index + 1),
      }).success,
    ).toBe(false)
    expect(benchmarkInputSchema.safeParse({ ...base, M: [1] }).success).toBe(false)
    expect(benchmarkInputSchema.safeParse({ ...base, queries: 1 }).success).toBe(false)
  })

  it('normalizes the dynamic recall@k field from benchmark rows', () => {
    const parsed = benchmarkResultSchema.parse({
      M: 16,
      efConstruction: 200,
      efSearch: 100,
      metric: 'l2',
      heuristic: true,
      k: 10,
      'recall@10': 0.968,
      qps: 3420.5,
      latencyP50Ms: 0.12,
      latencyP95Ms: 0.31,
      latencyP99Ms: 0.58,
      buildTimeMs: 412,
      bruteforceQps: 12.4,
      speedup: 275.8,
    })
    expect(parsed.recall).toBe(0.968)
  })

  it('parses index stats with a histogram', () => {
    const parsed = indexStatsSchema.parse({
      nodes: 100,
      maxLevel: 2,
      entryPoint: 7,
      nodesPerLevel: [100, 20, 2],
      degreeDistribution: { min: 2, max: 32, mean: 16.4, p50: 16 },
      degreeHistogram: [
        { degree: 2, count: 4 },
        { degree: 3, count: 10 },
      ],
      metric: 'l2',
      M: 16,
      efConstruction: 200,
    })
    expect(parsed.degreeHistogram.reduce((sum, bucket) => sum + bucket.count, 0)).toBe(14)
  })
})

describe('errors', () => {
  it('maps API codes to Russian messages', () => {
    expect(errorMessage(new ApiError('INDEX_NOT_READY', 'building'))).toBe(
      'Индекс недоступен: идёт сборка или произошла ошибка',
    )
  })

  it('retries only network and server failures', () => {
    expect(shouldRetry(0, new ApiError('NETWORK_ERROR', 'down'))).toBe(true)
    expect(shouldRetry(0, new ApiError('INVALID_REQUEST', 'bad', 400))).toBe(false)
    expect(shouldRetry(2, new ApiError('NETWORK_ERROR', 'down'))).toBe(false)
  })
})

describe('request', () => {
  it('returns validated data', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ status: 'ok' })))
    const result = await request('/health', healthSchema)
    expect(result).toEqual({ status: 'ok' })
  })

  it('throws a typed error from the error envelope', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValue(
          jsonResponse(
            { error: { code: 'DATASET_NOT_FOUND', message: 'dataset "ds_9" not found' } },
            404,
          ),
        ),
    )
    await expect(request('/datasets/ds_9', datasetListSchema)).rejects.toMatchObject({
      code: 'DATASET_NOT_FOUND',
      status: 404,
    })
  })

  it('rejects a response that does not match the schema', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ datasets: 'nope' })))
    await expect(request('/datasets', datasetListSchema)).rejects.toMatchObject({
      code: 'INVALID_RESPONSE',
    })
  })

  it('reports network failures', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))
    await expect(request('/health', datasetListSchema)).rejects.toMatchObject({
      code: 'NETWORK_ERROR',
    })
  })
})
