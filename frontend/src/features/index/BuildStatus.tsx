import { Check, Network } from 'lucide-react'
import type { BuildProgress } from '@/api/hooks/useIndexEvents'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { EmptyState } from '@/components/ui/EmptyState'
import { ErrorBanner } from '@/components/ui/ErrorBanner'
import { Progress } from '@/components/ui/Progress'
import { cn } from '@/lib/cn'
import { formatInt, formatMs, formatPercent } from '@/lib/format'
import { STATUS_LABELS } from './labels'

const STATUS_TONE = {
  idle: 'neutral',
  building: 'warn',
  ready: 'accent',
  error: 'danger',
} as const

interface BuildStatusProps {
  progress: BuildProgress
  onRetry: (() => void) | null
}

export function BuildStatus({ progress, onRetry }: BuildStatusProps) {
  return (
    <Card
      title="Состояние индекса"
      actions={
        <>
          {progress.status === 'building' && progress.processed !== null ? (
            <Badge tone="accent">SSE</Badge>
          ) : null}
          <Badge tone={STATUS_TONE[progress.status]}>{STATUS_LABELS[progress.status]}</Badge>
        </>
      }
    >
      {progress.status === 'idle' ? (
        <EmptyState
          icon={Network}
          title="Индекс ещё не построен"
          description="Выберите датасет и параметры слева, затем запустите сборку. Прогресс будет обновляться в реальном времени."
        />
      ) : null}

      {progress.status === 'building' ? (
        <div className="flex flex-col gap-3">
          <div className="flex items-baseline justify-between gap-3">
            <span className="font-mono text-3xl tabular-nums text-ink">
              {formatPercent(progress.progress, 0)}
            </span>
            <span className="font-mono text-[12px] tabular-nums text-ink-3">
              {progress.processed !== null && progress.total !== null
                ? `${formatInt(progress.processed)} / ${formatInt(progress.total)} векторов`
                : 'сборка идёт…'}
            </span>
          </div>
          <Progress value={progress.progress} label="Прогресс сборки индекса" />
          <div className="flex flex-wrap items-center justify-between gap-2 text-[12px] text-ink-3">
            <span>Прошло {formatMs(progress.elapsedMs)}</span>
            {progress.indexId ? (
              <span className="font-mono text-[11px]">{progress.indexId}</span>
            ) : null}
          </div>
        </div>
      ) : null}

      {progress.status === 'ready' ? (
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-2 text-accent">
            <Check aria-hidden className="size-4" />
            <span className="text-[13px] font-medium">Индекс готов</span>
          </div>
          <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-[12px] text-ink-3">
            <span>Сборка заняла {formatMs(progress.elapsedMs)}</span>
            {progress.indexId ? (
              <span className="font-mono text-[11px]">{progress.indexId}</span>
            ) : null}
          </div>
        </div>
      ) : null}

      {progress.status === 'error' ? (
        <div className="flex flex-col gap-3">
          <ErrorBanner
            title="Сборка завершилась ошибкой"
            message={progress.error ?? 'Сервер не сообщил подробности.'}
          />
          {onRetry ? (
            <div className="flex justify-end">
              <Button type="button" variant="secondary" size="sm" onClick={onRetry}>
                Повторить сборку
              </Button>
            </div>
          ) : null}
        </div>
      ) : null}

      <div
        aria-hidden
        className={cn('mt-4 h-px w-full', progress.status === 'ready' ? 'bg-accent/30' : 'bg-line')}
      />
      <p className="mt-3 text-[12px] leading-relaxed text-ink-3">
        {progress.status === 'building'
          ? 'Во время сборки поиск и вставка недоступны. Повторный запуск отклонится с 409.'
          : 'Повторная сборка заменяет текущий индекс: старый граф удаляется, поиск недоступен до готовности.'}
      </p>
    </Card>
  )
}
