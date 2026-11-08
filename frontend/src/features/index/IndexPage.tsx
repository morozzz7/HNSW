import { useState } from 'react'
import { PageHeader } from '@/app/PageHeader'
import { errorMessage } from '@/api/errors'
import { useDatasets } from '@/api/hooks/useDatasets'
import { useBuildIndex, useIndexStats } from '@/api/hooks/useIndex'
import { useBuildProgress } from '@/api/hooks/useIndexEvents'
import type { BuildIndexInput } from '@/api/schemas'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { ErrorBanner } from '@/components/ui/ErrorBanner'
import { BuildForm } from './BuildForm'
import { BuildStatus } from './BuildStatus'
import { GraphStats } from './GraphStats'

export function IndexPage() {
  const datasetsQuery = useDatasets()
  const build = useBuildIndex()
  const progress = useBuildProgress()
  const statsQuery = useIndexStats(progress.status === 'ready')
  const [lastConfig, setLastConfig] = useState<BuildIndexInput | null>(null)

  const submit = (input: BuildIndexInput) => {
    setLastConfig(input)
    build.mutate(input)
  }

  return (
    <>
      <PageHeader
        title="Построение индекса"
        description="Параметры HNSW, ход асинхронной сборки и статистика готового графа."
      />

      <div className="grid gap-4 xl:grid-cols-[minmax(0,400px)_minmax(0,1fr)] xl:items-start">
        <BuildForm
          datasets={datasetsQuery.data?.datasets ?? []}
          building={progress.status === 'building'}
          error={build.error}
          onSubmit={submit}
        />

        <div className="flex flex-col gap-4">
          <BuildStatus
            progress={progress}
            onRetry={
              lastConfig && progress.status === 'error' ? () => build.mutate(lastConfig) : null
            }
          />

          {progress.status === 'ready' ? (
            statsQuery.isLoading ? (
              <Card title="Статистика графа">
                <div className="flex flex-col gap-2 py-1">
                  <div className="h-16 animate-pulse rounded-lg bg-surface-2" aria-hidden />
                  <div className="h-24 animate-pulse rounded-lg bg-surface-2" aria-hidden />
                  <span className="sr-only">Загрузка статистики индекса</span>
                </div>
              </Card>
            ) : statsQuery.error ? (
              <ErrorBanner
                title="Не удалось получить статистику графа"
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
            ) : statsQuery.data ? (
              <GraphStats stats={statsQuery.data} />
            ) : null
          ) : null}
        </div>
      </div>
    </>
  )
}
