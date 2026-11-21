import { http, HttpResponse } from 'msw'
import { sse } from 'msw/sse'
import {
  MAX_DIMENSION,
  MAX_UPLOAD_BYTES,
  MAX_VALUES,
  MAX_VECTORS,
  benchmarkInputSchema,
  buildIndexInputSchema,
  generateInputSchema,
  searchInputSchema,
} from '@/api/schemas'
import type {
  BenchmarkResult,
  BuildIndexInput,
  IndexDoneEvent,
  IndexErrorEvent,
  IndexProgressEvent,
} from '@/api/schemas'
import {
  addGeneratedDataset,
  addUploadedDataset,
  currentIndex,
  findDataset,
  listDatasets,
  mockGraph,
  mockIndexStats,
  mockIndexStatus,
  mockSearch,
  refreshBenchmarkJob,
  startMockBenchmark,
  startMockBuild,
} from './state'

const base = '*/api/v1'

function fail(status: number, code: string, message: string) {
  return HttpResponse.json({ error: { code, message } }, { status })
}

function validationMessage(result: {
  success: boolean
  error?: { issues: { message: string }[] }
}) {
  return result.error?.issues[0]?.message ?? 'Некорректные параметры'
}

function toBenchmarkWire(row: BenchmarkResult) {
  const { recall, ...rest } = row
  return { ...rest, [`recall@${row.k}`]: recall }
}

function parseUpload(text: string, name: string): number[][] {
  if (name.endsWith('.json')) {
    const parsed: unknown = JSON.parse(text)
    if (!Array.isArray(parsed)) {
      throw new Error('JSON должен содержать массив векторов')
    }
    return parsed.map((row) => {
      if (!Array.isArray(row)) {
        throw new Error('Каждый элемент JSON должен быть массивом чисел')
      }
      return row.map(Number)
    })
  }

  const rows = text
    .split(/\r?\n/)
    .map((row) => row.trim())
    .filter(Boolean)
  return rows.map((row) => row.split(/[,;\t]/).map((cell) => Number(cell.trim())))
}

function validateVectors(vectors: number[][]): { n: number; dim: number } {
  if (vectors.length === 0) {
    throw new Error('Датасет пуст')
  }
  const first = vectors[0]
  if (!first || first.length === 0) {
    throw new Error('Нулевая размерность вектора')
  }
  if (first.length > MAX_DIMENSION) {
    throw new Error(`Размерность больше ${MAX_DIMENSION}`)
  }
  if (vectors.length > MAX_VECTORS) {
    throw new Error(`Больше ${MAX_VECTORS} векторов`)
  }
  for (const row of vectors) {
    if (row.length !== first.length) {
      throw new Error('Векторы разной размерности')
    }
    if (row.some((value) => !Number.isFinite(value))) {
      throw new Error('В векторах есть NaN или Inf')
    }
  }
  if (vectors.length * first.length > MAX_VALUES) {
    throw new Error('Слишком много значений в датасете')
  }
  return { n: vectors.length, dim: first.length }
}

export const handlers = [
  http.get(`${base}/health`, () => HttpResponse.json({ status: 'ok' })),

  http.get(`${base}/datasets`, () => HttpResponse.json({ datasets: listDatasets() })),

  http.post(`${base}/datasets`, async ({ request }) => {
    const contentType = request.headers.get('content-type') ?? ''

    if (contentType.includes('multipart/form-data')) {
      const form = await request.formData()
      const file = form.get('file')
      if (!(file instanceof File)) {
        return fail(400, 'INVALID_REQUEST', 'В форме нет поля "file"')
      }
      if (file.size > MAX_UPLOAD_BYTES) {
        return fail(413, 'FILE_TOO_LARGE', `Файл больше ${MAX_UPLOAD_BYTES} байт`)
      }
      const name = file.name.toLowerCase()
      if (!name.endsWith('.csv') && !name.endsWith('.json')) {
        return fail(415, 'UNSUPPORTED_FILE_TYPE', 'Допустимы только .csv и .json')
      }
      try {
        const vectors = parseUpload(await file.text(), name)
        validateVectors(vectors)
        const dataset = addUploadedDataset(vectors, name.endsWith('.json') ? 'json' : 'csv')
        return HttpResponse.json(
          { id: dataset.id, n: dataset.n, dim: dataset.dim, preview: dataset.preview },
          { status: 201 },
        )
      } catch (error) {
        return fail(
          400,
          'INVALID_REQUEST',
          error instanceof Error ? error.message : 'Не удалось разобрать файл',
        )
      }
    }

    const body: unknown = await request.json()
    const input = { ...(body as Record<string, unknown>) }
    if (input.action !== 'generate') {
      return fail(400, 'INVALID_REQUEST', 'action должен быть "generate"')
    }
    delete input.action
    const parsed = generateInputSchema.safeParse(input)
    if (!parsed.success) {
      return fail(400, 'INVALID_REQUEST', validationMessage(parsed))
    }
    const dataset = addGeneratedDataset(parsed.data)
    return HttpResponse.json(
      { id: dataset.id, n: dataset.n, dim: dataset.dim, preview: dataset.preview },
      { status: 201 },
    )
  }),

  http.get(`${base}/datasets/:id`, ({ params }) => {
    const dataset = findDataset(String(params.id))
    if (!dataset) {
      return fail(404, 'DATASET_NOT_FOUND', `Датасет "${String(params.id)}" не найден`)
    }
    return HttpResponse.json({
      id: dataset.id,
      n: dataset.n,
      dim: dataset.dim,
      preview: dataset.preview,
      stats: dataset.stats,
      createdAt: dataset.createdAt,
      source: dataset.source,
      distribution: dataset.distribution,
    })
  }),

  http.post(`${base}/index`, async ({ request }) => {
    const body = (await request.json()) as Partial<BuildIndexInput>
    const requested = {
      datasetId: body.datasetId ?? '',
      M: body.M ?? 16,
      efConstruction: body.efConstruction ?? 200,
      metric: body.metric ?? 'l2',
      heuristic: body.heuristic ?? true,
      seed: body.seed ?? 0,
    }
    const parsed = buildIndexInputSchema.safeParse(requested)
    if (!parsed.success) {
      return fail(400, 'INVALID_REQUEST', validationMessage(parsed))
    }
    const dataset = findDataset(parsed.data.datasetId)
    if (!dataset) {
      return fail(404, 'DATASET_NOT_FOUND', `Датасет "${parsed.data.datasetId}" не найден`)
    }
    const active = currentIndex()
    if (active && active.status === 'building') {
      return fail(409, 'INDEX_NOT_READY', 'Сборка индекса уже идёт')
    }
    const build = startMockBuild(dataset, {
      M: parsed.data.M,
      efConstruction: parsed.data.efConstruction,
      metric: parsed.data.metric,
      heuristic: parsed.data.heuristic,
      seed: parsed.data.seed,
    })
    return HttpResponse.json({ indexId: build.indexId, status: 'building' }, { status: 202 })
  }),

  http.get(`${base}/index/status`, () => HttpResponse.json(mockIndexStatus())),

  http.get(`${base}/index/stats`, () => {
    const stats = mockIndexStats()
    if (stats) {
      return HttpResponse.json(stats)
    }
    const status = mockIndexStatus()
    if (status.status === 'building') {
      return fail(409, 'INDEX_NOT_READY', 'Индекс ещё строится')
    }
    if (status.status === 'error') {
      return fail(500, 'BUILD_FAILED', status.error ?? 'Сборка индекса завершилась ошибкой')
    }
    return fail(404, 'INDEX_NOT_FOUND', 'Индекс ещё не построен')
  }),

  http.post(`${base}/search`, async ({ request }) => {
    let raw: unknown
    try {
      raw = await request.json()
    } catch {
      return fail(400, 'INVALID_REQUEST', 'Тело запроса должно быть JSON')
    }
    const parsed = searchInputSchema.safeParse(raw)
    if (!parsed.success) {
      return fail(400, 'INVALID_REQUEST', validationMessage(parsed))
    }
    const status = mockIndexStatus()
    if (status.status === 'building') {
      return fail(409, 'INDEX_NOT_READY', 'Индекс ещё строится')
    }
    if (status.status !== 'ready') {
      return fail(404, 'INDEX_NOT_FOUND', 'Индекс ещё не построен')
    }
    const searchRes = mockSearch(parsed.data)
    if (!searchRes) {
      return fail(500, 'SEARCH_FAILED', 'Ошибка выполнения поиска')
    }
    return HttpResponse.json(searchRes)
  }),

  http.get(`${base}/graph`, ({ request }) => {
    const url = new URL(request.url)
    const levelStr = url.searchParams.get('level')
    const limitStr = url.searchParams.get('limit')
    const around = url.searchParams.get('around') ?? undefined

    const level = levelStr ? parseInt(levelStr, 10) : 0
    const limit = limitStr ? parseInt(limitStr, 10) : 500

    if (Number.isNaN(level) || level < 0) {
      return fail(400, 'INVALID_REQUEST', 'level must be an integer >= 0')
    }
    if (Number.isNaN(limit) || limit < 1 || limit > 5000) {
      return fail(400, 'INVALID_REQUEST', 'limit must be between 1 and 5000')
    }

    const status = mockIndexStatus()
    if (status.status === 'building') {
      return fail(409, 'INDEX_NOT_READY', 'Индекс ещё строится')
    }
    if (status.status !== 'ready') {
      return fail(404, 'INDEX_NOT_FOUND', 'Индекс ещё не построен')
    }

    const graphRes = mockGraph(level, limit, around)
    if (!graphRes) {
      return fail(400, 'INVALID_REQUEST', 'Не удалось получить граф для уровня')
    }
    return HttpResponse.json(graphRes)
  }),

  http.post(`${base}/benchmark`, async ({ request }) => {
    let raw: unknown
    try {
      raw = await request.json()
    } catch {
      return fail(400, 'INVALID_REQUEST', 'Тело запроса должно быть JSON')
    }
    const parsed = benchmarkInputSchema.safeParse(raw)
    if (!parsed.success) {
      return fail(400, 'INVALID_REQUEST', validationMessage(parsed))
    }
    const status = mockIndexStatus()
    if (status.status === 'building') {
      return fail(409, 'INDEX_NOT_READY', 'Индекс ещё строится')
    }
    if (status.status !== 'ready') {
      return fail(404, 'INDEX_NOT_FOUND', 'Индекс ещё не построен')
    }
    const job = startMockBenchmark(parsed.data)
    if (!job) {
      return fail(500, 'INTERNAL_ERROR', 'Не удалось запустить бенчмарк')
    }
    return HttpResponse.json({ id: job.id, status: 'queued' }, { status: 202 })
  }),

  http.get(`${base}/benchmark/:id`, ({ params }) => {
    const job = refreshBenchmarkJob(String(params.id))
    if (!job) {
      return fail(404, 'NOT_FOUND', `Задача "${String(params.id)}" не найдена`)
    }
    return HttpResponse.json({
      id: job.id,
      status: job.status,
      progress: job.progress,
      ...(job.status === 'error' ? { error: job.error ?? 'Ошибка бенчмарка' } : {}),
      ...(job.status === 'done' ? { results: job.results.map(toBenchmarkWire) } : {}),
    })
  }),

  sse<{ progress: IndexProgressEvent; done: IndexDoneEvent; error: IndexErrorEvent }>(
    `${base}/index/events`,
    ({ client, request }) => {
      const sendSnapshot = (): boolean => {
        const status = mockIndexStatus()
        if (status.status === 'ready') {
          client.send({ event: 'done', data: { indexId: status.indexId } })
          client.close()
          return true
        }
        if (status.status === 'error') {
          client.send({
            event: 'error',
            data: { message: status.error ?? 'Сборка индекса завершилась ошибкой' },
          })
          client.close()
          return true
        }
        if (status.status === 'building') {
          const active = currentIndex()
          const total = active ? (findDataset(active.datasetId)?.n ?? 0) : 0
          client.send({
            event: 'progress',
            data: {
              progress: status.progress,
              processed: Math.round(status.progress * total),
              total,
            },
          })
        }
        return false
      }

      if (sendSnapshot()) {
        return
      }

      const timer = setInterval(() => {
        if (sendSnapshot()) {
          clearInterval(timer)
        }
      }, 250)

      request.signal.addEventListener('abort', () => clearInterval(timer))
    },
  ),
]
