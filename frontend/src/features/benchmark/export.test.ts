import { describe, expect, it } from 'vitest'
import type { BenchmarkResult } from '@/api/schemas'
import { benchmarkResultsToCsv } from './export'

const result: BenchmarkResult = {
  M: 16,
  efConstruction: 200,
  efSearch: 50,
  metric: 'cosine',
  heuristic: true,
  k: 10,
  recall: 0.968,
  qps: 3420.5,
  latencyP50Ms: 0.12,
  latencyP95Ms: 0.31,
  latencyP99Ms: 0.58,
  buildTimeMs: 412,
  bruteforceQps: 12.4,
  speedup: 275.8,
}

describe('benchmark CSV export', () => {
  it('uses a stable recall column independent of k', () => {
    const csv = benchmarkResultsToCsv([result])
    const [header, row] = csv.split('\n')

    expect(header).toBe(
      'M,efConstruction,efSearch,metric,heuristic,k,recall,qps,latencyP50Ms,latencyP95Ms,latencyP99Ms,buildTimeMs,bruteforceQps,speedup',
    )
    expect(row).toBe('16,200,50,cosine,true,10,0.968,3420.5,0.12,0.31,0.58,412,12.4,275.8')
  })

  it('writes one row per result', () => {
    const csv = benchmarkResultsToCsv([result, { ...result, efSearch: 100 }])
    expect(csv.split('\n')).toHaveLength(3)
  })
})
