import { Compass, Sparkles } from 'lucide-react'
import { cn } from '@/lib/cn'
import type { LayoutMode } from '../labels'

interface LayoutToggleProps {
  mode: LayoutMode
  onChange: (mode: LayoutMode) => void
  disabled?: boolean
}

export function LayoutToggle({ mode, onChange, disabled }: LayoutToggleProps) {
  return (
    <div
      role="group"
      aria-label="Переключение укладки графа"
      className="inline-flex rounded-lg border border-line bg-surface p-0.5"
    >
      <button
        type="button"
        disabled={disabled}
        onClick={() => onChange('pca')}
        className={cn(
          'flex h-7 items-center gap-1.5 rounded-md px-2.5 text-[11px] font-medium transition-colors disabled:opacity-50',
          mode === 'pca' ? 'bg-surface-2 text-ink shadow-xs' : 'text-ink-3 hover:text-ink',
        )}
      >
        <Compass className="size-3.5" aria-hidden />
        <span>PCA (2D)</span>
      </button>

      <button
        type="button"
        disabled={disabled}
        onClick={() => onChange('force')}
        className={cn(
          'flex h-7 items-center gap-1.5 rounded-md px-2.5 text-[11px] font-medium transition-colors disabled:opacity-50',
          mode === 'force' ? 'bg-surface-2 text-ink shadow-xs' : 'text-ink-3 hover:text-ink',
        )}
      >
        <Sparkles className="size-3.5" aria-hidden />
        <span>Силовая (ForceAtlas2)</span>
      </button>
    </div>
  )
}
