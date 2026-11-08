import { ChartColumn } from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { EmptyState } from '@/components/ui/EmptyState'
import { PageHeader } from '@/app/PageHeader'

export function BenchmarkPage() {
  return (
    <>
      <PageHeader
        title="Бенчмарк"
        description="Сетка параметров, recall@k, QPS и задержки против полного перебора."
      />
      <Card title="Запуск серии">
        <EmptyState
          icon={ChartColumn}
          title="Раздел этапа 8"
          description="Графики и таблицы появятся после поиска и визуализации графа."
        />
      </Card>
    </>
  )
}
