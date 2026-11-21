import { ChartColumn } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router-dom'
import { PageHeader } from '@/app/PageHeader'
import { errorMessage } from '@/api/errors'
import { useBenchmarkJob, useRunBenchmark } from '@/api/hooks/useBenchmark'
import { useIndexStats, useIndexStatus } from '@/api/hooks/useIndex'
import type { BenchmarkInput } from '@/api/schemas'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { EmptyState } from '@/components/ui/EmptyState'
import { ErrorBanner } from '@/components/ui/ErrorBanner'
import { BenchmarkCharts } from './BenchmarkCharts'
import { BenchmarkForm } from './BenchmarkForm'
import { BenchmarkProgress } from './BenchmarkProgress'
import { BenchmarkSummary } from './BenchmarkSummary'
import { BenchmarkTable } from './BenchmarkTable'

export function BenchmarkPage() {
  const statusQuery = useIndexStatus()
  const isReady = statusQuery.data?.status === 'ready'
  const statsQuery = useIndexStats(isReady)
  const stats = statsQuery.data

  const run = useRunBenchmark()
  const [jobId, setJobId] = useState<string | null>(null)
  const [startedAt, setStartedAt] = useState(0)
  const [lastInput, setLastInput] = useState<BenchmarkInput | null>(null)
  const [targetRecall, setTargetRecall] = useState(0.95)

  const jobQuery = useBenchmarkJob(jobId)
  const job = jobQuery.data
  const running = job?.status === 'queued' || job?.status === 'running'
  const completed =
    job?.status === 'done' && job.results ? { jobId: job.id, results: job.results } : null

  const submit = (input: BenchmarkInput) => {
    setLastInput(input)
    run.mutate(input, {
      onSuccess: (response) => {
        setJobId(response.id)
        setStartedAt(Date.now())
      },
    })
  }

  const combinations = lastInput
    ? lastInput.M.length * lastInput.efConstruction.length * lastInput.efSearch.length
    : 0

  if (!isReady) {
    return (
      <>
        <PageHeader
          title="Бенчмарк"
          description="Сетка параметров, recall@k, QPS и задержки против полного перебора."
        />
        <Card title="Индекс не готов">
          <EmptyState
            icon={ChartColumn}
            title="Индекс ещё не построен"
            description="Бенчмарк строит индекс заново для каждой комбинации параметров, поэтому ему нужен готовый индекс и датасет."
            action={
              <Link to="/index">
                <Button variant="primary">Перейти к построению индекса</Button>
              </Link>
            }
          />
        </Card>
      </>
    )
  }

  return (
    <>
      <PageHeader
        title="Бенчмарк"
        description="Сетка M × efConstruction × efSearch: recall@k, QPS, задержки и ускорение против полного перебора."
      />

      <div className="grid gap-4 xl:grid-cols-[minmax(0,380px)_minmax(0,1fr)] xl:items-start">
        <div className="flex flex-col gap-4">
          {stats ? (
            <BenchmarkForm
              nodes={stats.nodes}
              indexId={stats.indexId ?? ''}
              running={running}
              error={run.error}
              onSubmit={submit}
            />
          ) : (
            <Card title="Сетка параметров">
              <div className="flex flex-col gap-2 py-1">
                <div className="h-9 animate-pulse rounded-lg bg-surface-2" aria-hidden />
                <div className="h-9 animate-pulse rounded-lg bg-surface-2" aria-hidden />
                <span className="sr-only">Загрузка параметров индекса</span>
              </div>
            </Card>
          )}

          {job ? (
            <BenchmarkProgress job={job} startedAt={startedAt} combinations={combinations} />
          ) : null}
        </div>

        <div className="flex flex-col gap-4">
          {statsQuery.error ? (
            <ErrorBanner
              title="Не удалось получить параметры индекса"
              message={errorMessage(statsQuery.error)}
              action={
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={() => void statsQuery.refetch()}
                >
                  Повторить
                </Button>
              }
            />
          ) : null}

          {jobQuery.error ? (
            <ErrorBanner
              title="Не удалось получить состояние бенчмарка"
              message={errorMessage(jobQuery.error)}
            />
          ) : null}

          {job?.status === 'error' ? (
            <ErrorBanner
              title="Бенчмарк завершился ошибкой"
              message={job.error ?? 'Сервер не сообщил подробности.'}
            />
          ) : null}

          {completed ? (
            <>
              <BenchmarkSummary
                results={completed.results}
                targetRecall={targetRecall}
                onTargetRecallChange={setTargetRecall}
              />
              <BenchmarkCharts
                results={completed.results}
                targetRecall={targetRecall}
                jobId={completed.jobId}
              />
              <BenchmarkTable results={completed.results} jobId={completed.jobId} />
            </>
          ) : (
            <Card title="Результаты">
              <EmptyState
                icon={ChartColumn}
                title="Прогонов ещё не было"
                description="Задайте списки M, efConstruction и efSearch слева и запустите бенчмарк. Результаты появятся после завершения."
              />
            </Card>
          )}
        </div>
      </div>
    </>
  )
}
