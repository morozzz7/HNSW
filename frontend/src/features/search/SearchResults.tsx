import { Check } from 'lucide-react'
import { Badge } from '@/components/ui/Badge'
import { Card } from '@/components/ui/Card'
import { Stat } from '@/components/ui/Stat'
import { Table, Td, Th } from '@/components/ui/Table'
import { formatDecimal, formatMs, formatPercent, formatVectorPreview } from '@/lib/format'
import type { SearchOutcome } from '@/api/hooks/useSearch'
import type { SearchResultItem } from '@/api/schemas'

interface SearchResultsProps {
  outcome: SearchOutcome
  selectedNodeId: number | null
  hoveredNodeId?: number | null
  onSelectNode: (id: number) => void
  onHoverNode?: (id: number | null) => void
}

export function SearchResults({
  outcome,
  selectedNodeId,
  hoveredNodeId,
  onSelectNode,
  onHoverNode,
}: SearchResultsProps) {
  const { mode, hnsw, bruteforce, comparison } = outcome

  const results: SearchResultItem[] =
    mode === 'bruteforce' ? (bruteforce?.results ?? []) : (hnsw?.results ?? [])

  const bfIdMap = new Map<number, { rank: number; distance: number }>()
  if (bruteforce) {
    bruteforce.results.forEach((item, index) => {
      bfIdMap.set(item.id, { rank: index + 1, distance: item.distance })
    })
  }

  return (
    <Card
      title="Результаты k-NN"
      description={
        mode === 'both'
          ? 'Сравнение результатов HNSW и полного перебора'
          : `Найдено соседей: ${results.length}`
      }
      actions={
        mode === 'both' && comparison ? (
          <Badge tone={comparison.recall >= 0.95 ? 'accent' : 'warn'}>
            Recall@{results.length}: {formatPercent(comparison.recall)}
          </Badge>
        ) : null
      }
    >
      <div className="flex flex-col gap-4">
        {/* Comparison Metrics Banner */}
        {mode === 'both' && comparison ? (
          <div className="grid grid-cols-2 gap-2 rounded-lg border border-line bg-surface-2 p-3 sm:grid-cols-4">
            <Stat
              label="Точность (Recall)"
              value={formatPercent(comparison.recall, 1)}
              hint={`${comparison.intersectionCount} из ${results.length}`}
            />
            <Stat
              label="Ускорение HNSW"
              value={`${formatDecimal(comparison.speedup, 1)}x`}
              hint="к полному перебору"
            />
            <Stat label="Время HNSW" value={formatMs(comparison.hnswDurationMs)} />
            <Stat label="Время перебора" value={formatMs(comparison.bfDurationMs)} />
          </div>
        ) : null}

        {/* Results Table */}
        <Table>
          <thead>
            <tr>
              <Th>#</Th>
              <Th>ID</Th>
              <Th numeric>Дистанция</Th>
              {mode === 'both' ? (
                <>
                  <Th numeric>Дистанция BF</Th>
                  <Th>Совпадение</Th>
                </>
              ) : null}
              <Th>Вектор</Th>
            </tr>
          </thead>
          <tbody>
            {results.map((item, index) => {
              const rank = index + 1
              const isSelected = selectedNodeId === item.id
              const isHovered = hoveredNodeId === item.id
              const bfMatch = bfIdMap.get(item.id)

              return (
                <tr
                  key={item.id}
                  onClick={() => onSelectNode(item.id)}
                  onMouseEnter={() => onHoverNode?.(item.id)}
                  onMouseLeave={() => onHoverNode?.(null)}
                  className={`cursor-pointer transition-colors ${
                    isSelected
                      ? 'bg-accent-soft/30'
                      : isHovered
                        ? 'bg-surface-2'
                        : 'hover:bg-surface-2/60'
                  }`}
                >
                  <Td>{rank}</Td>
                  <Td>
                    <span className="font-semibold text-ink">#{item.id}</span>
                    {item.vectorId !== String(item.id) ? (
                      <span className="ml-1 text-[11px] text-ink-3">({item.vectorId})</span>
                    ) : null}
                  </Td>
                  <Td numeric>{formatDecimal(item.distance, 4)}</Td>
                  {mode === 'both' ? (
                    <>
                      <Td numeric>
                        {bfMatch ? (
                          formatDecimal(bfMatch.distance, 4)
                        ) : (
                          <span className="text-ink-3">—</span>
                        )}
                      </Td>
                      <Td>
                        {bfMatch ? (
                          <span className="inline-flex items-center gap-1 rounded bg-accent-soft px-1.5 py-0.5 text-[11px] font-medium text-accent">
                            <Check className="size-3" />
                            <span>BF #{bfMatch.rank}</span>
                          </span>
                        ) : (
                          <span className="text-[11px] text-ink-3">не вошёл в top-k</span>
                        )}
                      </Td>
                    </>
                  ) : null}
                  <Td className="max-w-[140px] truncate text-ink-3">
                    {item.vector ? formatVectorPreview(item.vector, 3) : '—'}
                  </Td>
                </tr>
              )
            })}
          </tbody>
        </Table>
      </div>
    </Card>
  )
}
