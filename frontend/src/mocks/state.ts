import type {
  DegreeBucket,
  GenerateInput,
  GraphEdge,
  GraphResponse,
  GraphVertex,
  Metric,
  SearchInput,
  SearchResponse,
  SearchResultItem,
} from '@/api/schemas'

export interface MockDataset {
  id: string
  n: number
  dim: number
  preview: number[][]
  stats: { min: number[]; max: number[]; mean: number[] }
  createdAt: string
  source: string
  distribution?: string
}

export interface MockIndex {
  indexId: string
  datasetId: string
  config: {
    M: number
    efConstruction: number
    metric: Metric
    heuristic: boolean
    seed: number
  }
  startedAt: number
  durationMs: number
  status: 'building' | 'ready' | 'error'
  error: string | null
  stats: MockIndexStats | null
}

export interface MockIndexStats {
  nodes: number
  maxLevel: number
  entryPoint: number
  nodesPerLevel: number[]
  degreeDistribution: { min: number; max: number; mean: number; p50: number }
  degreeHistogram: DegreeBucket[]
  metric: Metric
  M: number
  efConstruction: number
  indexId: string
  datasetId: string
}

export const DEFAULT_BUILD_MS = 4_000

let datasets: MockDataset[] = []
let index: MockIndex | null = null
let datasetCounter = 0
let indexCounter = 0
let buildMs = DEFAULT_BUILD_MS

function round(value: number): number {
  return Math.round(value * 1000) / 1000
}

function buildPreview(n: number, dim: number, seed: number): number[][] {
  const rows = Math.min(n, 10)
  return Array.from({ length: rows }, (_, row) =>
    Array.from({ length: dim }, (_, col) =>
      round(Math.sin((row + 1) * (col + 2) * 0.37 + seed) * 1.8 + Math.cos((col + 1) * 0.11)),
    ),
  )
}

function buildStats(dim: number) {
  return {
    min: Array.from({ length: dim }, (_, col) => round(-2.4 + Math.sin(col) * 0.08)),
    max: Array.from({ length: dim }, (_, col) => round(2.4 + Math.cos(col) * 0.08)),
    mean: Array.from({ length: dim }, (_, col) => round(Math.sin(col * 1.7) * 0.05)),
  }
}

function vectorStats(vectors: number[][]) {
  const dim = vectors[0]?.length ?? 0
  const min = Array.from({ length: dim }, () => Number.POSITIVE_INFINITY)
  const max = Array.from({ length: dim }, () => Number.NEGATIVE_INFINITY)
  const sum = Array.from({ length: dim }, () => 0)
  for (const row of vectors) {
    for (let col = 0; col < dim; col += 1) {
      const value = row[col] ?? 0
      min[col] = Math.min(min[col] ?? value, value)
      max[col] = Math.max(max[col] ?? value, value)
      sum[col] = (sum[col] ?? 0) + value
    }
  }
  const n = Math.max(1, vectors.length)
  return {
    min: min.map(round),
    max: max.map(round),
    mean: sum.map((value) => round(value / n)),
  }
}

function createDataset(input: {
  n: number
  dim: number
  source: string
  distribution?: string
  preview?: number[][]
  stats?: MockDataset['stats']
  seed?: number
}): MockDataset {
  datasetCounter += 1
  const dataset: MockDataset = {
    id: `ds_${String(datasetCounter).padStart(6, '0')}`,
    n: input.n,
    dim: input.dim,
    preview: input.preview ?? buildPreview(input.n, input.dim, input.seed ?? 1),
    stats: input.stats ?? buildStats(input.dim),
    createdAt: new Date().toISOString(),
    source: input.source,
    distribution: input.distribution,
  }
  datasets = [...datasets, dataset]
  return dataset
}

export function seedMockState() {
  datasets = []
  index = null
  datasetCounter = 0
  indexCounter = 0
  buildMs = DEFAULT_BUILD_MS

  const uniform = createDataset({
    n: 2_000,
    dim: 16,
    source: 'generate',
    distribution: 'uniform',
    seed: 1,
  })
  const clusters = createDataset({
    n: 500,
    dim: 8,
    source: 'generate',
    distribution: 'clusters',
    seed: 7,
  })
  return { uniform, clusters }
}

export function resetMockState() {
  seedMockState()
}

export function setMockBuildDuration(ms: number) {
  buildMs = ms
}

export function listDatasets() {
  return datasets.map(({ id, n, dim, createdAt }) => ({ id, n, dim, createdAt }))
}

export function findDataset(id: string) {
  return datasets.find((dataset) => dataset.id === id)
}

export function addGeneratedDataset(input: GenerateInput) {
  return createDataset({
    n: input.n,
    dim: input.dim,
    source: 'generate',
    distribution: input.distribution,
    seed: input.seed,
  })
}

export function addUploadedDataset(vectors: number[][], source: string) {
  const dim = vectors[0]?.length ?? 0
  return createDataset({
    n: vectors.length,
    dim,
    source,
    preview: vectors.slice(0, 10),
    stats: vectorStats(vectors),
  })
}

function histogram(total: number, minDegree: number, maxDegree: number, mean: number) {
  const degrees: number[] = []
  const weights: number[] = []
  const sigma = Math.max(1, (maxDegree - minDegree) / 4)
  for (let degree = minDegree; degree <= maxDegree; degree += 1) {
    degrees.push(degree)
    weights.push(Math.exp(-((degree - mean) ** 2) / (2 * sigma * sigma)))
  }
  const weightSum = weights.reduce((sum, value) => sum + value, 0)
  let assigned = 0
  let heaviest = 0
  const buckets: DegreeBucket[] = degrees.map((degree, i) => {
    const count = Math.round((total * (weights[i] ?? 0)) / weightSum)
    if ((weights[i] ?? 0) > (weights[heaviest] ?? 0)) {
      heaviest = i
    }
    assigned += count
    return { degree, count }
  })
  const remainder = total - assigned
  const target = buckets[heaviest]
  if (target) {
    target.count = Math.max(0, target.count + remainder)
  }
  return buckets.filter((bucket) => bucket.count > 0)
}

function computeIndexStats(dataset: MockDataset, config: MockIndex['config']): MockIndexStats {
  const nodes = dataset.n
  const m = Math.max(2, config.M)
  const maxLevel = Math.min(5, Math.max(0, Math.floor(Math.log(Math.max(2, nodes)) / Math.log(m))))
  const nodesPerLevel: number[] = []
  let current = nodes
  for (let level = 0; level <= maxLevel; level += 1) {
    nodesPerLevel.push(current)
    current = Math.max(0, Math.round(current / m))
  }
  const pairs = nodesPerLevel.reduce((sum, value) => sum + value, 0)
  const minDegree = Math.max(0, Math.floor(m / 2))
  const maxDegree = 2 * m
  const mean = m * 1.55

  return {
    nodes,
    maxLevel,
    entryPoint: nodes > 1 ? Math.floor(nodes / 3) : 0,
    nodesPerLevel,
    degreeDistribution: { min: minDegree, max: maxDegree, mean, p50: m },
    degreeHistogram: histogram(pairs, minDegree, maxDegree, mean),
    metric: config.metric,
    M: config.M,
    efConstruction: config.efConstruction,
    indexId: '',
    datasetId: dataset.id,
  }
}

export function startMockBuild(dataset: MockDataset, config: MockIndex['config']) {
  indexCounter += 1
  const indexId = `idx_${String(indexCounter).padStart(8, '0')}`
  index = {
    indexId,
    datasetId: dataset.id,
    config,
    startedAt: Date.now(),
    durationMs: buildMs,
    status: 'building',
    error: null,
    stats: null,
  }
  return index
}

export function currentIndex() {
  return index
}

export function refreshIndex() {
  if (!index) {
    return null
  }
  const elapsed = Date.now() - index.startedAt
  if (index.status === 'building' && elapsed >= index.durationMs) {
    const dataset = findDataset(index.datasetId)
    if (dataset) {
      index.status = 'ready'
      index.stats = {
        ...computeIndexStats(dataset, index.config),
        indexId: index.indexId,
      }
    } else {
      index.status = 'error'
      index.error = 'Датасет индекса не найден'
    }
  }
  return index
}

export function mockIndexStatus() {
  const active = refreshIndex()
  if (!active) {
    return { status: 'idle' as const, progress: 0, elapsedMs: 0, indexId: '', error: null }
  }
  const elapsed = Date.now() - active.startedAt
  const progress =
    active.status === 'ready'
      ? 1
      : Math.min(0.99, Math.max(0, elapsed / Math.max(1, active.durationMs)))
  return {
    status: active.status,
    progress,
    elapsedMs: elapsed,
    indexId: active.indexId,
    error: active.error,
  }
}

export function mockIndexStats() {
  const active = refreshIndex()
  if (!active || active.status !== 'ready' || !active.stats) {
    return null
  }
  return active.stats
}

export function failMockBuild(message: string) {
  if (!index) {
    return
  }
  index.status = 'error'
  index.error = message
}

export function mockGraph(level = 0, limit = 500, around?: string): GraphResponse | null {
  const active = refreshIndex()
  if (!active || active.status !== 'ready' || !active.stats) {
    return null
  }
  const dataset = findDataset(active.datasetId)
  if (!dataset) {
    return null
  }
  if (level < 0 || level > active.stats.maxLevel) {
    return null
  }
  const total = active.stats.nodesPerLevel[level] ?? 0
  const count = Math.min(limit, total, 500)
  const vertices: GraphVertex[] = []
  const edges: GraphEdge[] = []

  let aroundId: number | null = null
  if (around !== undefined && around !== '') {
    const parsed = parseInt(around, 10)
    if (!Number.isNaN(parsed) && parsed >= 0 && parsed < dataset.n) {
      aroundId = parsed
    }
  }

  const selectedIds: number[] = []
  if (aroundId !== null) {
    selectedIds.push(aroundId)
    for (let i = 1; i < count; i += 1) {
      const nid = (aroundId + i * 3) % dataset.n
      selectedIds.push(nid)
    }
  } else {
    for (let i = 0; i < count; i += 1) {
      selectedIds.push(i)
    }
  }

  const idSet = new Set(selectedIds)

  for (let idx = 0; idx < selectedIds.length; idx += 1) {
    const id = selectedIds[idx]!
    const angle = id * 2.39996
    const radius = 10 * Math.sqrt(id + 1)
    const x = Math.cos(angle) * radius * 0.1
    const y = Math.sin(angle) * radius * 0.1
    const nodeLevel = id % 7 === 0 ? Math.min(active.stats.maxLevel, 2) : 0
    vertices.push({
      id,
      vectorId: String(id),
      level: nodeLevel,
      x,
      y,
      isEntryPoint: id === active.stats.entryPoint,
    })
  }

  for (let idx = 0; idx < selectedIds.length; idx += 1) {
    const u = selectedIds[idx]!
    const deg = (u % 4) + 1
    for (let j = 1; j <= deg; j += 1) {
      const v = selectedIds[(idx + j * 2) % selectedIds.length]!
      if (u < v && idSet.has(v)) {
        edges.push({ from: u, to: v, level })
      }
    }
  }

  return {
    level,
    limit,
    totalVertices: total,
    vertices,
    edges,
  }
}

export function mockSearch(input: SearchInput): SearchResponse | null {
  const active = refreshIndex()
  if (!active || active.status !== 'ready' || !active.stats) {
    return null
  }
  const dataset = findDataset(active.datasetId)
  if (!dataset) {
    return null
  }
  const k = Math.min(input.k, dataset.n)
  const results: SearchResultItem[] = []
  for (let i = 0; i < k; i += 1) {
    const id = (i * 7 + 3) % dataset.n
    results.push({
      id,
      vectorId: String(id),
      distance: Math.round((0.15 + i * 0.12) * 1000) / 1000,
      vector:
        input.includeVectors !== false ? dataset.preview[i % dataset.preview.length] : undefined,
    })
  }

  let trace: SearchResponse['trace'] = undefined
  if (input.trace && input.mode !== 'bruteforce') {
    const layers = []
    for (let l = active.stats.maxLevel; l >= 0; l -= 1) {
      const visited = [
        active.stats.entryPoint,
        (active.stats.entryPoint + l * 2 + 1) % dataset.n,
        results[0]?.id ?? 0,
      ]
      layers.push({
        level: l,
        visited,
        closest: l === 0 ? (results[0]?.id ?? 0) : visited[visited.length - 1]!,
      })
    }
    trace = { layers }
  }

  return {
    results,
    found: results.length,
    trace,
  }
}

resetMockState()
