import { describe, expect, it } from 'vitest'
import { api } from '@/api/endpoints'
import { setMockBenchmarkDuration, setMockBuildDuration } from '@/mocks/state'
import type { IndexStatus } from '@/api/schemas'

async function waitForReady(timeoutMs = 2_000): Promise<IndexStatus> {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    const status = await api.indexStatus()
    if (status.status === 'ready') {
      return status
    }
    await new Promise((resolve) => setTimeout(resolve, 25))
  }
  throw new Error('index did not become ready')
}

describe('mock backend', () => {
  it('serves the seeded datasets', async () => {
    const { datasets } = await api.listDatasets()

    expect(datasets).toHaveLength(2)
    expect(datasets[0]).toMatchObject({ n: 2_000, dim: 16 })
    expect(datasets[1]).toMatchObject({ n: 500, dim: 8 })
  })

  it('generates a dataset and exposes its preview', async () => {
    const created = await api.generateDataset({
      n: 120,
      dim: 8,
      distribution: 'gaussian',
      seed: 42,
    })

    expect(created.n).toBe(120)
    expect(created.dim).toBe(8)
    expect(created.preview).toHaveLength(10)

    const listed = await api.listDatasets()
    expect(listed.datasets.map((dataset) => dataset.id)).toContain(created.id)

    const detail = await api.getDataset(created.id)
    expect(detail.preview[0]).toHaveLength(8)
    expect(detail.stats?.mean).toHaveLength(8)
  })

  it('builds an index and returns stats with a histogram', async () => {
    setMockBuildDuration(40)
    const created = await api.generateDataset({
      n: 300,
      dim: 6,
      distribution: 'uniform',
      seed: 3,
    })

    const started = await api.buildIndex({
      datasetId: created.id,
      M: 8,
      efConstruction: 100,
      metric: 'l2',
      heuristic: true,
      seed: 1,
    })
    expect(started.status).toBe('building')

    const building = await api.indexStatus()
    expect(building.status).toBe('building')
    expect(building.indexId).toBe(started.indexId)

    const ready = await waitForReady()
    expect(ready.progress).toBe(1)

    const stats = await api.indexStats()
    expect(stats.nodes).toBe(300)
    expect(stats.indexId).toBe(started.indexId)
    expect(stats.datasetId).toBe(created.id)
    expect(stats.degreeHistogram.length).toBeGreaterThan(0)

    const pairs = stats.nodesPerLevel.reduce((sum, value) => sum + value, 0)
    const buckets = stats.degreeHistogram.reduce((sum, bucket) => sum + bucket.count, 0)
    expect(buckets).toBe(pairs)
  })

  it('refuses to build while another build is running', async () => {
    setMockBuildDuration(5_000)
    const created = await api.generateDataset({
      n: 50,
      dim: 4,
      distribution: 'uniform',
      seed: 9,
    })
    const input = {
      datasetId: created.id,
      M: 4,
      efConstruction: 50,
      metric: 'l2' as const,
      heuristic: true,
      seed: 0,
    }

    await api.buildIndex(input)

    await expect(api.buildIndex(input)).rejects.toMatchObject({
      code: 'INDEX_NOT_READY',
      status: 409,
    })
  })

  it('runs search and returns graph data when index is ready', async () => {
    setMockBuildDuration(20)
    const created = await api.generateDataset({
      n: 100,
      dim: 4,
      distribution: 'uniform',
      seed: 5,
    })
    await api.buildIndex({
      datasetId: created.id,
      M: 8,
      efConstruction: 50,
      metric: 'l2',
      heuristic: true,
      seed: 1,
    })
    await waitForReady()

    const searchRes = await api.search({
      vectorId: '0',
      k: 5,
      efSearch: 20,
      mode: 'hnsw',
      trace: true,
    })
    expect(searchRes.results).toHaveLength(5)
    expect(searchRes.found).toBe(5)
    expect(searchRes.trace?.layers.length).toBeGreaterThan(0)

    const graphRes = await api.graph({ level: 0, limit: 50 })
    expect(graphRes.level).toBe(0)
    expect(graphRes.vertices.length).toBeGreaterThan(0)
    expect(graphRes.vertices[0]).toHaveProperty('x')
    expect(graphRes.vertices[0]).toHaveProperty('y')
  })

  it('runs a benchmark job and normalizes recall@k', async () => {
    setMockBuildDuration(0)
    setMockBenchmarkDuration(0)
    const created = await api.generateDataset({
      n: 120,
      dim: 6,
      distribution: 'uniform',
      seed: 2,
    })
    await api.buildIndex({
      datasetId: created.id,
      M: 8,
      efConstruction: 100,
      metric: 'l2',
      heuristic: true,
      seed: 1,
    })
    await waitForReady()

    const started = await api.runBenchmark({
      M: [8, 16],
      efConstruction: [100],
      efSearch: [10, 50],
      k: 5,
      queries: 20,
    })
    expect(started.status).toBe('queued')

    const job = await api.benchmarkJob(started.id)
    expect(job.status).toBe('done')
    expect(job.results).toHaveLength(4)
    expect(job.results?.[0]?.recall).toBeGreaterThan(0)
    expect(job.results?.[0]?.k).toBe(5)
  })
})
