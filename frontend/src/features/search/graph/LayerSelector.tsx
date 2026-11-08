import { Layers } from 'lucide-react'
import { cn } from '@/lib/cn'
import { formatInt } from '@/lib/format'
import { getLayerColor } from '../labels'

interface LayerSelectorProps {
  maxLevel: number
  activeLevel: number
  nodesPerLevel: number[]
  visitedLevels?: Set<number>
  isMultilayer: boolean
  onSelectLevel: (level: number) => void
  onToggleMultilayer: () => void
  totalEdges?: number
}

export function LayerSelector({
  maxLevel,
  activeLevel,
  nodesPerLevel,
  visitedLevels = new Set(),
  isMultilayer,
  onSelectLevel,
  onToggleMultilayer,
  totalEdges,
}: LayerSelectorProps) {
  const levels = Array.from({ length: maxLevel + 1 }, (_, i) => i).reverse()

  return (
    <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line px-4 py-2.5 bg-surface">
      <div className="flex items-center gap-1.5 overflow-x-auto pb-0.5">
        <span className="text-[11px] font-medium text-ink-3 mr-1">Слой:</span>
        {levels.map((level) => {
          const isActive = !isMultilayer && activeLevel === level
          const count = nodesPerLevel[level] ?? 0
          const hasVisited = visitedLevels.has(level)
          const color = getLayerColor(level)

          return (
            <button
              key={level}
              type="button"
              onClick={() => onSelectLevel(level)}
              className={cn(
                'group relative flex items-center gap-1.5 rounded-md px-2.5 py-1 text-[12px] font-mono transition-all',
                isActive
                  ? 'bg-surface-2 text-ink shadow-xs'
                  : 'text-ink-2 hover:bg-surface-2 hover:text-ink',
              )}
            >
              <span
                className="size-2 rounded-full transition-transform group-hover:scale-125"
                style={{ backgroundColor: color }}
                aria-hidden
              />
              <span className="font-medium">L{level}</span>
              <span className="text-[11px] text-ink-3 tabular-nums">({formatInt(count)})</span>
              {hasVisited ? (
                <span
                  title="Слой задействован в спуске"
                  className="size-1.5 rounded-full bg-accent"
                  aria-hidden
                />
              ) : null}
              {isActive ? (
                <span
                  className="absolute bottom-0 left-2 right-2 h-0.5 rounded-full"
                  style={{ backgroundColor: color }}
                  aria-hidden
                />
              ) : null}
            </button>
          )
        })}
      </div>

      <div className="flex items-center gap-3">
        {totalEdges !== undefined ? (
          <span className="hidden font-mono text-[11px] text-ink-3 sm:inline">
            Рёбер на слое: <span className="tabular-nums text-ink-2">{formatInt(totalEdges)}</span>
          </span>
        ) : null}

        <button
          type="button"
          onClick={onToggleMultilayer}
          className={cn(
            'flex h-7 items-center gap-1.5 rounded-md border px-2.5 text-[11px] font-medium transition-colors',
            isMultilayer
              ? 'border-accent bg-accent-soft text-accent'
              : 'border-line bg-surface text-ink-2 hover:border-line-strong hover:text-ink',
          )}
          title="Отобразить все слои в изометрической 2.5D проекции"
        >
          <Layers className="size-3.5" aria-hidden />
          <span>Многослойный вид</span>
        </button>
      </div>
    </div>
  )
}
