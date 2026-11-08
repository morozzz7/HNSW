import type { ReactNode } from 'react'
import { cn } from '@/lib/cn'

interface StatProps {
  label: string
  value: ReactNode
  unit?: string
  hint?: ReactNode
  className?: string
}

export function Stat({ label, value, unit, hint, className }: StatProps) {
  return (
    <div className={cn('flex flex-col gap-1', className)}>
      <span className="text-[12px] text-ink-3">{label}</span>
      <span className="flex items-baseline gap-1 font-mono text-xl tabular-nums text-ink">
        {value}
        {unit ? <span className="text-[12px] text-ink-3">{unit}</span> : null}
      </span>
      {hint ? <span className="text-[12px] text-ink-3">{hint}</span> : null}
    </div>
  )
}
