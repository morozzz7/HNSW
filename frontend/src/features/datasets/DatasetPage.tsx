import { RefreshCw } from 'lucide-react'
import { useState } from 'react'
import { PageHeader } from '@/app/PageHeader'
import { useDatasets } from '@/api/hooks/useDatasets'
import { Button } from '@/components/ui/Button'
import { DatasetDetail } from './DatasetDetail'
import { DatasetList } from './DatasetList'
import { GenerateCard } from './GenerateCard'
import { UploadCard } from './UploadCard'

export function DatasetPage() {
  const datasetsQuery = useDatasets()
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const datasets = datasetsQuery.data?.datasets ?? []
  const activeId = selectedId ?? datasets[0]?.id ?? null

  return (
    <>
      <PageHeader
        title="Датасеты"
        description="Загрузите CSV или JSON либо сгенерируйте синтетический набор. Лимиты: 1 000 000 векторов, размерность до 8192, не больше 50 млн компонентов."
        actions={
          <Button
            type="button"
            variant="secondary"
            size="sm"
            loading={datasetsQuery.isFetching}
            onClick={() => void datasetsQuery.refetch()}
          >
            <RefreshCw aria-hidden className="size-4" />
            Обновить
          </Button>
        }
      />

      <div className="grid gap-4 xl:grid-cols-2">
        <UploadCard onCreated={setSelectedId} />
        <GenerateCard onCreated={setSelectedId} />
      </div>

      <DatasetList
        datasets={datasets}
        activeId={activeId}
        isLoading={datasetsQuery.isLoading}
        error={datasetsQuery.error}
        onSelect={setSelectedId}
        onRetry={() => void datasetsQuery.refetch()}
      />

      {activeId ? <DatasetDetail id={activeId} /> : null}
    </>
  )
}
