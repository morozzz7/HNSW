import { Check, Copy, Crosshair, Star, X } from 'lucide-react'
import { useState } from 'react'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { formatDecimal, formatInt } from '@/lib/format'
import type { GraphVertex } from '@/api/schemas'

interface NodeInfoPanelProps {
  vertex: GraphVertex
  neighbors: number[]
  activeLevel: number
  vector?: readonly number[]
  onCenterAround: (id: string) => void
  onSelectNeighbor: (id: number) => void
  onClose: () => void
}

export function NodeInfoPanel({
  vertex,
  neighbors,
  activeLevel,
  vector,
  onCenterAround,
  onSelectNeighbor,
  onClose,
}: NodeInfoPanelProps) {
  const [copied, setCopied] = useState(false)

  const copyVector = () => {
    if (!vector) return
    void navigator.clipboard.writeText(JSON.stringify(Array.from(vector)))
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  return (
    <Card
      title={`Узел #${vertex.id}`}
      description={`Внешний ID: ${vertex.vectorId}`}
      actions={
        <button
          type="button"
          onClick={onClose}
          className="rounded p-1 text-ink-3 transition-colors hover:bg-surface-2 hover:text-ink"
          title="Закрыть панель"
        >
          <X className="size-4" />
        </button>
      }
      className="animate-in fade-in-0 duration-150"
    >
      <div className="flex flex-col gap-3 text-[12px]">
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone={vertex.isEntryPoint ? 'warn' : 'neutral'}>
            {vertex.isEntryPoint ? (
              <span className="flex items-center gap-1">
                <Star className="size-3 fill-warn text-warn" />
                Входная точка
              </span>
            ) : (
              `Макс. уровень: L${vertex.level}`
            )}
          </Badge>
          <Badge tone="accent">
            Соседей на L{activeLevel}: {formatInt(neighbors.length)}
          </Badge>
        </div>

        <div className="grid grid-cols-2 gap-2 rounded-md border border-line bg-surface-2 p-2 font-mono text-[11px]">
          <div>
            <span className="text-ink-3">X (2D): </span>
            <span className="tabular-nums text-ink">{formatDecimal(vertex.x, 3)}</span>
          </div>
          <div>
            <span className="text-ink-3">Y (2D): </span>
            <span className="tabular-nums text-ink">{formatDecimal(vertex.y, 3)}</span>
          </div>
        </div>

        {neighbors.length > 0 ? (
          <div>
            <p className="mb-1 text-[11px] font-medium text-ink-3">
              Соседи на текущем слое ({neighbors.length}):
            </p>
            <div className="flex max-h-24 flex-wrap gap-1 overflow-y-auto pr-1">
              {neighbors.slice(0, 32).map((nid) => (
                <button
                  key={nid}
                  type="button"
                  onClick={() => onSelectNeighbor(nid)}
                  className="rounded border border-line bg-surface px-1.5 py-0.5 font-mono text-[10px] text-ink-2 transition-colors hover:border-accent hover:text-accent"
                >
                  #{nid}
                </button>
              ))}
              {neighbors.length > 32 ? (
                <span className="self-center font-mono text-[10px] text-ink-3">
                  +{neighbors.length - 32} ещё
                </span>
              ) : null}
            </div>
          </div>
        ) : null}

        {vector && vector.length > 0 ? (
          <div>
            <div className="mb-1 flex items-center justify-between">
              <span className="text-[11px] font-medium text-ink-3">
                Вектор ({vector.length} dim):
              </span>
              <button
                type="button"
                onClick={copyVector}
                className="flex items-center gap-1 text-[10px] text-ink-3 transition-colors hover:text-ink"
              >
                {copied ? <Check className="size-3 text-accent" /> : <Copy className="size-3" />}
                <span>{copied ? 'Скопировано' : 'Копировать'}</span>
              </button>
            </div>
            <div className="max-h-20 overflow-y-auto rounded bg-surface-2 p-2 font-mono text-[10px] tabular-nums text-ink-2">
              [
              {vector
                .slice(0, 16)
                .map((v) => formatDecimal(v, 3))
                .join(', ')}
              {vector.length > 16 ? ', …' : ''}]
            </div>
          </div>
        ) : null}

        <Button
          type="button"
          variant="secondary"
          size="sm"
          onClick={() => onCenterAround(String(vertex.id))}
          className="w-full gap-1.5"
        >
          <Crosshair className="size-3.5" />
          <span>Центрировать окрестность (around)</span>
        </Button>
      </div>
    </Card>
  )
}
