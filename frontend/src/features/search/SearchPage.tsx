import { Search } from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { EmptyState } from '@/components/ui/EmptyState'
import { PageHeader } from '@/app/PageHeader'

export function SearchPage() {
  return (
    <>
      <PageHeader
        title="Поиск"
        description="k-NN запросы, сравнение HNSW с полным перебором и трассировка спуска по слоям."
      />
      <Card title="Запрос">
        <EmptyState
          icon={Search}
          title="Раздел этапа 7"
          description="Форма запроса, таблица результатов и визуализация графа появятся после этапа 6."
        />
      </Card>
    </>
  )
}
