import type { BenchmarkResult } from '@/api/schemas'
import { Card } from '@/components/ui/Card'
import { Select } from '@/components/ui/Select'
import { Stat } from '@/components/ui/Stat'
import { formatDecimal, formatInt, formatPercent } from '@/lib/format'
import { bestResult, bruteforceQps, maxSpeedup, qpsAtRecall } from './derive'

const TARGET_RECALLS = [0.9, 0.95, 0.99]

interface BenchmarkSummaryProps {
  results: BenchmarkResult[]
  targetRecall: number
  onTargetRecallChange: (value: number) => void
}

export function BenchmarkSummary({
  results,
  targetRecall,
  onTargetRecallChange,
}: BenchmarkSummaryProps) {
  const best = bestResult(results)
  const fastest = maxSpeedup(results)
  const atTarget = qpsAtRecall(results, targetRecall)
  const bfQps = bruteforceQps(results)

  return (
    <Card
      title="Сравнение с полным перебором"
      description="Лучшие точки сетки и ускорение относительно точного поиска."
      actions={
        <label className="flex items-center gap-2 text-[11px] text-ink-3">
          <span>Целевой recall</span>
          <Select
            value={String(targetRecall)}
            onChange={(event) => onTargetRecallChange(Number(event.target.value))}
            className="h-7 w-[84px] text-[12px]"
            aria-label="Целевой recall"
          >
            {TARGET_RECALLS.map((value) => (
              <option key={value} value={value}>
                {formatPercent(value, 0)}
              </option>
            ))}
          </Select>
        </label>
      }
    >
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Stat label="QPS перебора" value={formatInt(bfQps)} hint="точный поиск, без индекса" />
        <Stat
          label="Лучший recall"
          value={best ? formatPercent(best.recall, 1) : '—'}
          hint={
            best ? `M=${best.M} · efC=${best.efConstruction} · efS=${best.efSearch}` : undefined
          }
        />
        <Stat
          label={`QPS при recall ≥ ${formatPercent(targetRecall, 0)}`}
          value={atTarget ? formatInt(atTarget.qps) : '—'}
          hint={
            atTarget ? `M=${atTarget.M} · efS=${atTarget.efSearch}` : 'нет точки с таким recall'
          }
        />
        <Stat
          label="Максимальное ускорение"
          value={fastest ? `${formatDecimal(fastest.speedup, 1)}x` : '—'}
          hint={
            fastest
              ? `recall ${formatPercent(fastest.recall, 1)} · M=${fastest.M} · efS=${fastest.efSearch}`
              : undefined
          }
        />
      </div>
    </Card>
  )
}
