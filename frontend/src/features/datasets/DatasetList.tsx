import { Database } from 'lucide-react'
import type { DatasetListItem } from '@/api/schemas'
import { errorMessage } from '@/api/errors'
import { Badge } from '@/components/ui/Badge'
import { Card } from '@/components/ui/Card'
import { EmptyState } from '@/components/ui/EmptyState'
import { ErrorBanner } from '@/components/ui/ErrorBanner'
import { Button } from '@/components/ui/Button'
import { Table, Td, Th } from '@/components/ui/Table'
import { cn } from '@/lib/cn'
import { formatDate, formatInt } from '@/lib/format'

interface DatasetListProps {
  datasets: DatasetListItem[]
  activeId: string | null
  isLoading: boolean
  error: unknown
  onSelect: (id: string) => void
  onRetry: () => void
}

export function DatasetList({
  datasets,
  activeId,
  isLoading,
  error,
  onSelect,
  onRetry,
}: DatasetListProps) {
  return (
    <Card
      title="Наборы данных"
      description="Выберите набор, чтобы посмотреть превью и статистику по координатам."
      actions={<Badge>{formatInt(datasets.length)}</Badge>}
    >
      {isLoading ? (
        <div className="flex flex-col gap-2 py-1">
          {Array.from({ length: 3 }, (_, index) => (
            <div key={index} className="h-9 animate-pulse rounded-lg bg-surface-2" aria-hidden />
          ))}
          <span className="sr-only">Загрузка датасетов</span>
        </div>
      ) : error ? (
        <ErrorBanner
          title="Не удалось получить список датасетов"
          message={errorMessage(error)}
          action={
            <Button type="button" variant="secondary" size="sm" onClick={onRetry}>
              Повторить
            </Button>
          }
        />
      ) : datasets.length === 0 ? (
        <EmptyState
          icon={Database}
          title="Пока нет ни одного датасета"
          description="Загрузите CSV или JSON или сгенерируйте набор — он сразу появится в этом списке."
        />
      ) : (
        <Table>
          <thead>
            <tr>
              <Th>Идентификатор</Th>
              <Th numeric>Векторы</Th>
              <Th numeric>Размерность</Th>
              <Th numeric className="hidden sm:table-cell">
                Компоненты
              </Th>
              <Th className="hidden md:table-cell">Создан</Th>
            </tr>
          </thead>
          <tbody>
            {datasets.map((dataset) => {
              const active = dataset.id === activeId
              return (
                <tr
                  key={dataset.id}
                  tabIndex={0}
                  aria-current={active ? 'true' : undefined}
                  onClick={() => onSelect(dataset.id)}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter' || event.key === ' ') {
                      event.preventDefault()
                      onSelect(dataset.id)
                    }
                  }}
                  className={cn(
                    'cursor-pointer transition-colors focus-visible:bg-surface-2',
                    active ? 'bg-accent-soft' : 'hover:bg-surface-2',
                  )}
                >
                  <Td className="font-mono text-ink">{dataset.id}</Td>
                  <Td numeric>{formatInt(dataset.n)}</Td>
                  <Td numeric>{formatInt(dataset.dim)}</Td>
                  <Td numeric className="hidden sm:table-cell">
                    {formatInt(dataset.n * dataset.dim)}
                  </Td>
                  <Td className="hidden md:table-cell">{formatDate(dataset.createdAt)}</Td>
                </tr>
              )
            })}
          </tbody>
        </Table>
      )}
    </Card>
  )
}
