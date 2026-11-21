import { describe, expect, it } from 'vitest'
import type { BenchmarkResult } from '@/api/schemas'
import {
  buildTimeChart,
  efSearchValues,
  groupByConfig,
  maxSpeedup,
  paretoFrontier,
  paretoPoints,
  qpsAtRecall,
  recallSeries,
} from './derive'

function row(overrides: Partial<BenchmarkResult> = {}): BenchmarkResult {
  return {
    M: 16,
    efConstruction: 200,
    efSearch: 10,
    metric: 'l2',
    heuristic: true,
    k: 10,
    recall: 0.8,
    qps: 1000,
    latencyP50Ms: 0.7,
    latencyP95Ms: 1.7,
    latencyP99Ms: 2.8,
    buildTimeMs: 200,
    bruteforceQps: 100,
    speedup: 10,
    ...overrides,
  }
}

describe('benchmark derivation', () => {
  const results = [
    row({ efSearch: 50, recall: 0.9, qps: 2500, speedup: 25 }),
    row({ efSearch: 10, recall: 0.7, qps: 2000, speedup: 20 }),
    row({ M: 32, efConstruction: 100, efSearch: 10, recall: 0.75, qps: 1500, speedup: 15 }),
    row({ M: 32, efConstruction: 100, efSearch: 50, recall: 0.95, qps: 400, speedup: 4 }),
  ]

  it('groups runs by M and efConstruction and sorts points by efSearch', () => {
    const groups = groupByConfig(results)

    expect(groups).toHaveLength(2)
    expect(groups[0]).toMatchObject({ M: 16, efConstruction: 200 })
    expect(groups[0]?.results.map((item) => item.efSearch)).toEqual([10, 50])
    expect(recallSeries(results)[0]?.points).toEqual([
      { x: 10, y: 0.7 },
      { x: 50, y: 0.9 },
    ])
    expect(efSearchValues(results)).toEqual([10, 50])
  })

  it('keeps only non-dominated points on the Pareto frontier', () => {
    const frontier = paretoFrontier(results)

    expect(frontier.map((item) => item.efSearch)).toEqual([10, 50])
    expect(frontier.some((item) => item.M === 32 && item.efSearch === 10)).toBe(false)

    const points = paretoPoints(results)
    expect(points.frontier).toHaveLength(2)
    expect(points.points.filter((point) => point.frontier)).toHaveLength(2)
  })

  it('deduplicates build time by build configuration', () => {
    const chart = buildTimeChart(results)

    expect(chart.groups).toEqual([
      { label: 'M=16', values: { 'efC=200': 200 } },
      { label: 'M=32', values: { 'efC=100': 200 } },
    ])
    expect(chart.series).toEqual([
      { key: 'efC=100', label: 'efC=100' },
      { key: 'efC=200', label: 'efC=200' },
    ])
  })

  it('selects the fastest row that reaches the target recall', () => {
    const atTarget = qpsAtRecall(results, 0.85)
    expect(atTarget?.qps).toBe(2500)
    expect(qpsAtRecall(results, 0.99)).toBeNull()
    expect(maxSpeedup(results)?.speedup).toBe(25)
  })
})
