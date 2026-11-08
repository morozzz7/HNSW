import { cn } from '@/lib/cn'

interface ProgressProps {
  value: number
  label: string
  className?: string
}

export function Progress({ value, label, className }: ProgressProps) {
  const clamped = Math.min(1, Math.max(0, value))
  return (
    <div
      role="progressbar"
      aria-label={label}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={Math.round(clamped * 100)}
      className={cn('h-1.5 w-full overflow-hidden rounded-full bg-surface-3', className)}
    >
      <div
        className="h-full origin-left rounded-full bg-accent transition-transform duration-300 ease-out"
        style={{ transform: `scaleX(${clamped})` }}
      />
    </div>
  )
}
