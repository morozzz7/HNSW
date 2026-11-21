import { ArrowDown, ArrowUp, Download } from 'lucide-react'
import { useMemo, useState, type ReactNode } from 'react'
import type { BenchmarkResult } from '@/api/schemas'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Table, Td, Th } from '@/components/ui/Table'
import { formatDecimal, formatInt, formatMs, formatPercent } from '@/lib/format'
import { benchmarkResultsToCsv, downloadCsv } from './export'

type SortKey =
  | 'M'
  | 'efConstruction'
  | 'efSearch'
  | 'recall'
  | 'qps'
  | 'latencyP50Ms'
  | 'latencyP95Ms'
  | 'latencyP99Ms'
  | 'buildTimeMs'
  | 'bruteforceQps'
  | 'speedup'

const COLUMNS: { key: SortKey; label: string }[] = [
  { key: 'M', label: 'M' },
  { key: 'efConstruction', label: 'efC' },
  { key: 'efSearch', label: 'efS' },
  { key: 'recall', label: 'recall@k' },
  { key: 'qps', label: 'QPS' },
  { key: 'latencyP50Ms', label: 'p50' },
  { key: 'latencyP95Ms', label: 'p95' },
  { key: 'latencyP99Ms', label: 'p99' },
  { key: 'buildTimeMs', label: 'Сборка' },
  { key: 'bruteforceQps', label: 'QPS перебора' },
  { key: 'speedup', label: 'Ускорение' },
]

function renderValue(row: BenchmarkResult, key: SortKey): ReactNode {
  switch (key) {
    case 'recall':
      return formatPercent(row.recall, 1)
    case 'qps':
      return formatInt(row.qps)
    case 'latencyP50Ms':
    case 'latencyP95Ms':
    case 'latencyP99Ms':
      return formatDecimal(row[key], 3)
    case 'buildTimeMs':
      return formatMs(row.buildTimeMs)
    case 'bruteforceQps':
      return formatInt(row.bruteforceQps)
    case 'speedup':
      return `${formatDecimal(row.speedup, 1)}x`
    default:
      return formatInt(row[key])
  }
}

interface BenchmarkTableProps {
  results: BenchmarkResult[]
  jobId: string
}

export function BenchmarkTable({ results, jobId }: BenchmarkTableProps) {
  const [sortKey, setSortKey] = useState<SortKey>('recall')
  const [direction, setDirection] = useState<'asc' | 'desc'>('desc')

  const sorted = useMemo(() => {
    const rows = [...results]
    rows.sort((a, b) => {
      const left = a[sortKey]
      const right = b[sortKey]
      if (left === right) {
        return a.M - b.M || a.efConstruction - b.efConstruction || a.efSearch - b.efSearch
      }
      return direction === 'asc' ? left - right : right - left
    })
    return rows
  }, [direction, results, sortKey])

  const toggleSort = (key: SortKey) => {
    if (key === sortKey) {
      setDirection((previous) => (previous === 'asc' ? 'desc' : 'asc'))
      return
    }
    setSortKey(key)
    setDirection(key === 'M' || key === 'efConstruction' || key === 'efSearch' ? 'asc' : 'desc')
  }

  return (
    <Card
      title="Результаты сетки"
      description={`${formatInt(results.length)} прогонов · k = ${results[0]?.k ?? 0} · ${results[0]?.metric ?? ''}`}
      actions={
        <Button
          type="button"
          variant="secondary"
          size="sm"
          onClick={() =>
            downloadCsv(`benchmark_${jobId}_results.csv`, benchmarkResultsToCsv(sorted))
          }
        >
          <Download aria-hidden className="size-3.5" />
          CSV
        </Button>
      }
    >
      <Table>
        <thead>
          <tr>
            {COLUMNS.map((column) => {
              const active = column.key === sortKey
              return (
                <Th
                  key={column.key}
                  numeric
                  aria-sort={active ? (direction === 'asc' ? 'ascending' : 'descending') : 'none'}
                >
                  <button
                    type="button"
                    onClick={() => toggleSort(column.key)}
                    className={`inline-flex items-center gap-1 transition-colors hover:text-ink ${
                      active ? 'text-ink' : ''
                    }`}
                  >
                    {column.label}
                    {active ? (
                      direction === 'asc' ? (
                        <ArrowUp aria-hidden className="size-3" />
                      ) : (
                        <ArrowDown aria-hidden className="size-3" />
                      )
                    ) : null}
                  </button>
                </Th>
              )
            })}
          </tr>
        </thead>
        <tbody>
          {sorted.map((row) => (
            <tr key={`${row.M}/${row.efConstruction}/${row.efSearch}`}>
              {COLUMNS.map((column) => (
                <Td key={column.key} numeric className="whitespace-nowrap">
                  {renderValue(row, column.key)}
                </Td>
              ))}
            </tr>
          ))}
        </tbody>
      </Table>
    </Card>
  )
}
