import type { ReactNode } from 'react'
import { cn } from '@/lib/cn'

type Tone = 'neutral' | 'accent' | 'warn' | 'danger'

const TONES: Record<Tone, string> = {
  neutral: 'border-line bg-surface-2 text-ink-2',
  accent: 'border-transparent bg-accent-soft text-accent',
  warn: 'border-transparent bg-warn-soft text-warn',
  danger: 'border-transparent bg-danger-soft text-danger',
}

interface BadgeProps {
  tone?: Tone
  className?: string
  children: ReactNode
}

export function Badge({ tone = 'neutral', className, children }: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-md border px-1.5 py-0.5 font-mono text-[11px] leading-4',
        TONES[tone],
        className,
      )}
    >
      {children}
    </span>
  )
}
