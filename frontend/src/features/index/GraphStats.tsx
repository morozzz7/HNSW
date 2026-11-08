import { BarChart3 } from 'lucide-react'
import type { IndexStats } from '@/api/schemas'
import { Badge } from '@/components/ui/Badge'
import { Card } from '@/components/ui/Card'
import { EmptyState } from '@/components/ui/EmptyState'
import { Stat } from '@/components/ui/Stat'
import { formatDecimal, formatInt } from '@/lib/format'
import { METRIC_LABELS } from './labels'

function LevelBars({ nodesPerLevel }: { nodesPerLevel: number[] }) {
  const max = Math.max(1, ...nodesPerLevel)
  return (
    <div className="flex flex-col gap-2">
      {nodesPerLevel.map((count, level) => (
        <div
          key={level}
          className="grid grid-cols-[2.5rem_minmax(0,1fr)_4.5rem] items-center gap-2"
          title={`Уровень ${level}: ${formatInt(count)} узлов`}
        >
          <span className="font-mono text-[11px] text-ink-3">L{level}</span>
          <div className="h-2 overflow-hidden rounded-full bg-surface-3">
            <div
              className="h-full rounded-full bg-accent"
              style={{ width: `${(count / max) * 100}%` }}
            />
          </div>
          <span className="text-right font-mono text-[11px] tabular-nums text-ink-2">
            {formatInt(count)}
          </span>
        </div>
      ))}
    </div>
  )
}

function DegreeHistogram({ buckets }: { buckets: IndexStats['degreeHistogram'] }) {
  if (buckets.length === 0) {
    return (
      <EmptyState
        icon={BarChart3}
        title="Гистограмма пуста"
        description="Сервер не вернул степени."
      />
    )
  }

  const max = Math.max(1, ...buckets.map((bucket) => bucket.count))
  const first = buckets[0]
  const middle = buckets[Math.floor(buckets.length / 2)]
  const last = buckets[buckets.length - 1]

  return (
    <div className="flex flex-col gap-2">
      <div
        role="img"
        aria-label={`Гистограмма степеней от ${first?.degree ?? 0} до ${last?.degree ?? 0}`}
        className="flex h-28 items-end gap-[2px] border-b border-line pb-px"
      >
        {buckets.map((bucket) => (
          <div
            key={bucket.degree}
            title={`Степень ${bucket.degree}: ${formatInt(bucket.count)}`}
            className="min-w-px flex-1 rounded-t-sm bg-accent/75 transition-colors hover:bg-accent"
            style={{ height: `${Math.max(2, (bucket.count / max) * 100)}%` }}
          />
        ))}
      </div>
      <div className="flex justify-between font-mono text-[11px] text-ink-3">
        <span>d{first?.degree ?? 0}</span>
        <span>d{middle?.degree ?? 0}</span>
        <span>d{last?.degree ?? 0}</span>
      </div>
    </div>
  )
}

export function GraphStats({ stats }: { stats: IndexStats }) {
  return (
    <>
      <Card
        title="Статистика графа"
        description="Снимок готового индекса: размер, уровни и распределение степеней."
        actions={<Badge tone="accent">{METRIC_LABELS[stats.metric]}</Badge>}
      >
        <div className="flex flex-col gap-5">
          <div className="grid gap-4 sm:grid-cols-3 lg:grid-cols-5">
            <Stat label="Узлов" value={formatInt(stats.nodes)} />
            <Stat
              label="Уровней"
              value={formatInt(stats.maxLevel + 1)}
              hint={`maxLevel ${stats.maxLevel}`}
            />
            <Stat label="Входная точка" value={stats.entryPoint} />
            <Stat label="M" value={stats.M} />
            <Stat label="efConstruction" value={formatInt(stats.efConstruction)} />
          </div>

          <div className="rounded-lg border border-line bg-surface-2 px-4 py-3">
            <div className="mb-2 flex items-baseline justify-between gap-3">
              <span className="text-[12px] text-ink-3">Степени узлов</span>
              <span className="font-mono text-[11px] text-ink-3">min / p50 / mean / max</span>
            </div>
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
              <Stat label="min" value={stats.degreeDistribution.min} />
              <Stat label="p50" value={stats.degreeDistribution.p50} />
              <Stat label="mean" value={formatDecimal(stats.degreeDistribution.mean, 2)} />
              <Stat label="max" value={stats.degreeDistribution.max} />
            </div>
          </div>
        </div>
      </Card>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card
          title="Узлы по уровням"
          description="Сколько узлов присутствует на каждом уровне графа."
        >
          <LevelBars nodesPerLevel={stats.nodesPerLevel} />
        </Card>

        <Card
          title="Гистограмма степеней"
          description="Число пар (узел, уровень) с каждой исходящей степенью."
        >
          <DegreeHistogram buckets={stats.degreeHistogram} />
        </Card>
      </div>
    </>
  )
}
