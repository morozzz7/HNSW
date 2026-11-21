import type { BarChartGroup, BarChartSeries } from '@/components/charts/BarChart'
import type { LineChartSeries } from '@/components/charts/LineChart'
import type { ScatterChartPoint } from '@/components/charts/ScatterChart'
import type { BenchmarkResult } from '@/api/schemas'

export interface ConfigGroup {
  id: string
  label: string
  M: number
  efConstruction: number
  results: BenchmarkResult[]
}

export function groupByConfig(results: BenchmarkResult[]): ConfigGroup[] {
  const groups = new Map<string, BenchmarkResult[]>()
  for (const row of results) {
    const key = `${row.M}/${row.efConstruction}`
    const list = groups.get(key)
    if (list) {
      list.push(row)
    } else {
      groups.set(key, [row])
    }
  }

  return [...groups.entries()]
    .map(([id, rows]) => {
      const first = rows[0] as BenchmarkResult
      return {
        id,
        label: `M=${first.M} · efC=${first.efConstruction}`,
        M: first.M,
        efConstruction: first.efConstruction,
        results: [...rows].sort((a, b) => a.efSearch - b.efSearch),
      }
    })
    .sort((a, b) => a.M - b.M || a.efConstruction - b.efConstruction)
}

export function efSearchValues(results: BenchmarkResult[]): number[] {
  return [...new Set(results.map((row) => row.efSearch))].sort((a, b) => a - b)
}

export function recallSeries(results: BenchmarkResult[]): LineChartSeries[] {
  return groupByConfig(results).map((group) => ({
    id: group.id,
    label: group.label,
    points: group.results.map((row) => ({ x: row.efSearch, y: row.recall })),
  }))
}

export function latencySeries(results: BenchmarkResult[]): LineChartSeries[] {
  return [
    {
      id: 'p50',
      label: 'p50',
      points: results.map((row) => ({ x: row.efSearch, y: row.latencyP50Ms })),
    },
    {
      id: 'p95',
      label: 'p95',
      points: results.map((row) => ({ x: row.efSearch, y: row.latencyP95Ms })),
    },
    {
      id: 'p99',
      label: 'p99',
      points: results.map((row) => ({ x: row.efSearch, y: row.latencyP99Ms })),
    },
  ]
}

export function paretoFrontier(results: BenchmarkResult[]): BenchmarkResult[] {
  const sorted = [...results].sort((a, b) => a.recall - b.recall || b.qps - a.qps)
  const frontier: BenchmarkResult[] = []
  let bestQps = Number.NEGATIVE_INFINITY
  for (const row of sorted) {
    if (row.qps > bestQps) {
      frontier.push(row)
      bestQps = row.qps
    }
  }
  return frontier
}

export function paretoPoints(results: BenchmarkResult[]): {
  points: ScatterChartPoint[]
  frontier: ScatterChartPoint[]
} {
  const frontierIds = new Set(paretoFrontier(results).map(rowKey))
  const points = results.map((row) => ({
    id: rowKey(row),
    x: row.recall,
    y: row.qps,
    label: `M=${row.M} · efC=${row.efConstruction} · efS=${row.efSearch}`,
    frontier: frontierIds.has(rowKey(row)),
  }))
  return { points, frontier: points.filter((point) => point.frontier) }
}

function rowKey(row: BenchmarkResult): string {
  return `${row.M}/${row.efConstruction}/${row.efSearch}`
}

export interface BuildTimeChart {
  groups: BarChartGroup[]
  series: BarChartSeries[]
}

export function buildTimeChart(results: BenchmarkResult[]): BuildTimeChart {
  const builds = new Map<string, BenchmarkResult>()
  for (const row of results) {
    const key = `${row.M}/${row.efConstruction}`
    if (!builds.has(key)) {
      builds.set(key, row)
    }
  }

  const mValues = [...new Set(results.map((row) => row.M))].sort((a, b) => a - b)
  const efValues = [...new Set(results.map((row) => row.efConstruction))].sort((a, b) => a - b)

  const groups = mValues.map((M) => {
    const values: Record<string, number> = {}
    for (const efConstruction of efValues) {
      const build = builds.get(`${M}/${efConstruction}`)
      if (build) {
        values[`efC=${efConstruction}`] = build.buildTimeMs
      }
    }
    return { label: `M=${M}`, values }
  })

  return {
    groups,
    series: efValues.map((efConstruction) => ({
      key: `efC=${efConstruction}`,
      label: `efC=${efConstruction}`,
    })),
  }
}

export function bestResult(results: BenchmarkResult[]): BenchmarkResult | null {
  let best: BenchmarkResult | null = null
  for (const row of results) {
    if (!best || row.recall > best.recall || (row.recall === best.recall && row.qps > best.qps)) {
      best = row
    }
  }
  return best
}

export function qpsAtRecall(results: BenchmarkResult[], target: number): BenchmarkResult | null {
  let best: BenchmarkResult | null = null
  for (const row of results) {
    if (row.recall >= target && (!best || row.qps > best.qps)) {
      best = row
    }
  }
  return best
}

export function maxSpeedup(results: BenchmarkResult[]): BenchmarkResult | null {
  let best: BenchmarkResult | null = null
  for (const row of results) {
    if (!best || row.speedup > best.speedup) {
      best = row
    }
  }
  return best
}

export function bruteforceQps(results: BenchmarkResult[]): number {
  return results[0]?.bruteforceQps ?? 0
}
