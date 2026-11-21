import { useEffect, useState } from 'react'
import type { BenchmarkJob } from '@/api/schemas'
import { Badge } from '@/components/ui/Badge'
import { Card } from '@/components/ui/Card'
import { Progress } from '@/components/ui/Progress'
import { formatMs, formatPercent } from '@/lib/format'
import { BENCHMARK_STATUS_LABELS, BENCHMARK_STATUS_TONE } from './labels'

interface BenchmarkProgressProps {
  job: BenchmarkJob
  startedAt: number
  combinations: number
}

function useElapsed(startedAt: number, active: boolean): number {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    if (!active) {
      return
    }
    const timer = setInterval(() => setNow(Date.now()), 500)
    return () => clearInterval(timer)
  }, [active])
  return active ? Math.max(0, now - startedAt) : 0
}

export function BenchmarkProgress({ job, startedAt, combinations }: BenchmarkProgressProps) {
  const active = job.status === 'queued' || job.status === 'running'
  const elapsed = useElapsed(startedAt, active)

  return (
    <Card
      title="Ход прогона"
      actions={
        <Badge tone={BENCHMARK_STATUS_TONE[job.status]}>
          {BENCHMARK_STATUS_LABELS[job.status]}
        </Badge>
      }
    >
      <div className="flex flex-col gap-3">
        <div className="flex items-baseline justify-between gap-3">
          <span className="font-mono text-3xl tabular-nums text-ink">
            {formatPercent(job.progress, 0)}
          </span>
          <span className="font-mono text-[12px] tabular-nums text-ink-3">
            {combinations > 0 ? `${combinations} комбинаций` : job.id}
          </span>
        </div>
        <Progress value={job.progress} label="Прогресс бенчмарка" />
        <div className="flex flex-wrap items-center justify-between gap-2 text-[12px] text-ink-3">
          <span>{active ? `Прошло ${formatMs(elapsed)}` : 'Прогон завершён'}</span>
          <span className="font-mono text-[11px]">{job.id}</span>
        </div>
      </div>
    </Card>
  )
}
