import { Target, X } from 'lucide-react'
import { Select } from '@/components/ui/Select'

interface NodeNeighborhoodControlProps {
  limit: number
  onLimitChange: (limit: number) => void
  around: string | undefined
  onClearAround: () => void
  hopRadius: number
  onHopRadiusChange: (radius: number) => void
  hasSelection: boolean
}

const LIMIT_OPTIONS = [100, 250, 500, 1000, 2000]

export function NodeNeighborhoodControl({
  limit,
  onLimitChange,
  around,
  onClearAround,
  hopRadius,
  onHopRadiusChange,
  hasSelection,
}: NodeNeighborhoodControlProps) {
  return (
    <div className="flex flex-wrap items-center gap-2 text-[12px]">
      <div className="flex items-center gap-1.5">
        <label htmlFor="graph-limit-select" className="text-ink-3">
          Лимит узлов:
        </label>
        <Select
          id="graph-limit-select"
          value={limit}
          onChange={(e) => onLimitChange(Number(e.target.value))}
          className="h-7 w-20 py-0 text-[11px] font-mono"
        >
          {LIMIT_OPTIONS.map((val) => (
            <option key={val} value={val}>
              {val}
            </option>
          ))}
        </Select>
      </div>

      {hasSelection ? (
        <div className="flex items-center gap-1.5">
          <label htmlFor="hop-radius-range" className="text-ink-3">
            Радиус (hops):
          </label>
          <span className="font-mono text-[11px] tabular-nums text-ink">{hopRadius}</span>
          <input
            id="hop-radius-range"
            type="range"
            min={1}
            max={3}
            step={1}
            value={hopRadius}
            onChange={(e) => onHopRadiusChange(Number(e.target.value))}
            className="h-1.5 w-16 accent-accent"
          />
        </div>
      ) : null}

      {around ? (
        <div className="flex items-center gap-1 rounded-md border border-accent-soft bg-accent-soft/30 px-2 py-0.5 font-mono text-[11px] text-accent">
          <Target className="size-3" aria-hidden />
          <span>Центр: {around}</span>
          <button
            type="button"
            onClick={onClearAround}
            className="ml-1 text-ink-3 hover:text-ink"
            title="Сбросить центрирование"
          >
            <X className="size-3" />
          </button>
        </div>
      ) : null}
    </div>
  )
}
