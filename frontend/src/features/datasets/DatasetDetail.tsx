import { useDataset } from '@/api/hooks/useDatasets'
import { errorMessage } from '@/api/errors'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { ErrorBanner } from '@/components/ui/ErrorBanner'
import { Stat } from '@/components/ui/Stat'
import { Table, Td, Th } from '@/components/ui/Table'
import { formatDate, formatDecimal, formatInt } from '@/lib/format'
import { DISTRIBUTION_LABELS, sourceLabel } from './labels'
import type { Distribution } from '@/api/schemas'

const PREVIEW_COLUMNS = 8
const PREVIEW_ROWS = 10

interface DatasetDetailProps {
  id: string
}

export function DatasetDetail({ id }: DatasetDetailProps) {
  const query = useDataset(id)

  if (query.isLoading) {
    return (
      <Card title="Датасет">
        <div className="flex flex-col gap-2 py-1">
          <div className="h-5 w-48 animate-pulse rounded bg-surface-2" aria-hidden />
          <div className="h-24 animate-pulse rounded-lg bg-surface-2" aria-hidden />
          <span className="sr-only">Загрузка датасета</span>
        </div>
      </Card>
    )
  }

  if (query.error || !query.data) {
    return (
      <Card title="Датасет">
        <ErrorBanner
          title="Не удалось получить датасет"
          message={errorMessage(query.error)}
          action={
            <Button
              type="button"
              variant="secondary"
              size="sm"
              onClick={() => void query.refetch()}
            >
              Повторить
            </Button>
          }
        />
      </Card>
    )
  }

  const dataset = query.data
  const columns = Math.min(dataset.dim, PREVIEW_COLUMNS)
  const stats = dataset.stats
  const distribution = dataset.distribution as Distribution | undefined

  return (
    <Card
      title={`Датасет ${dataset.id}`}
      description="Превью первых векторов и статистика по каждой координате."
      actions={
        <>
          <Badge tone="accent">{sourceLabel(dataset.source)}</Badge>
          {distribution && DISTRIBUTION_LABELS[distribution] ? (
            <Badge>{DISTRIBUTION_LABELS[distribution]}</Badge>
          ) : null}
        </>
      }
    >
      <div className="flex flex-col gap-5">
        <div className="grid gap-4 sm:grid-cols-4">
          <Stat label="Векторов" value={formatInt(dataset.n)} />
          <Stat label="Размерность" value={formatInt(dataset.dim)} />
          <Stat label="Компонентов" value={formatInt(dataset.n * dataset.dim)} />
          <Stat label="Создан" value={dataset.createdAt ? formatDate(dataset.createdAt) : '—'} />
        </div>

        <section className="flex flex-col gap-2">
          <div className="flex items-baseline justify-between gap-3">
            <h3 className="text-[13px] font-medium text-ink">Превью</h3>
            <span className="font-mono text-[11px] text-ink-3">
              {Math.min(dataset.n, PREVIEW_ROWS)} из {formatInt(dataset.n)} строк
              {dataset.dim > PREVIEW_COLUMNS
                ? `, ${PREVIEW_COLUMNS} из ${dataset.dim} столбцов`
                : ''}
            </span>
          </div>
          <Table>
            <thead>
              <tr>
                <Th className="w-10">#</Th>
                {Array.from({ length: columns }, (_, index) => (
                  <Th
                    key={index}
                    numeric
                    className={index >= 4 ? 'hidden sm:table-cell' : undefined}
                  >
                    d{index + 1}
                  </Th>
                ))}
                {dataset.dim > PREVIEW_COLUMNS ? <Th>…</Th> : null}
              </tr>
            </thead>
            <tbody>
              {dataset.preview.slice(0, PREVIEW_ROWS).map((row, rowIndex) => (
                <tr key={rowIndex}>
                  <Td className="text-ink-3">{rowIndex + 1}</Td>
                  {Array.from({ length: columns }, (_, columnIndex) => (
                    <Td
                      key={columnIndex}
                      numeric
                      className={columnIndex >= 4 ? 'hidden sm:table-cell' : undefined}
                    >
                      {formatDecimal(row[columnIndex] ?? 0, 3)}
                    </Td>
                  ))}
                  {dataset.dim > PREVIEW_COLUMNS ? <Td className="text-ink-3">…</Td> : null}
                </tr>
              ))}
            </tbody>
          </Table>
        </section>

        <section className="flex flex-col gap-2">
          <div className="flex items-baseline justify-between gap-3">
            <h3 className="text-[13px] font-medium text-ink">Статистика координат</h3>
            <span className="font-mono text-[11px] text-ink-3">
              {stats ? `первые ${columns} из ${dataset.dim}` : 'нет данных'}
            </span>
          </div>
          {stats ? (
            <Table>
              <thead>
                <tr>
                  <Th>Координата</Th>
                  <Th numeric>min</Th>
                  <Th numeric>mean</Th>
                  <Th numeric>max</Th>
                </tr>
              </thead>
              <tbody>
                {Array.from({ length: columns }, (_, index) => (
                  <tr key={index}>
                    <Td className="font-mono text-ink">d{index + 1}</Td>
                    <Td numeric>{formatDecimal(stats.min[index] ?? 0, 3)}</Td>
                    <Td numeric>{formatDecimal(stats.mean[index] ?? 0, 3)}</Td>
                    <Td numeric>{formatDecimal(stats.max[index] ?? 0, 3)}</Td>
                  </tr>
                ))}
              </tbody>
            </Table>
          ) : (
            <p className="text-[13px] text-ink-3">Сервер не вернул статистику для этого набора.</p>
          )}
        </section>
      </div>
    </Card>
  )
}
