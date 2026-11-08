import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'
import { cn } from '@/lib/cn'

interface EmptyStateProps {
  icon?: LucideIcon
  title: string
  description?: ReactNode
  action?: ReactNode
  className?: string
}

export function EmptyState({ icon: Icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div className={cn('flex flex-col items-start gap-2 py-6', className)}>
      {Icon ? <Icon aria-hidden className="size-5 text-ink-3" /> : null}
      <p className="text-sm font-medium text-ink">{title}</p>
      {description ? (
        <p className="max-w-md text-[13px] leading-relaxed text-ink-3">{description}</p>
      ) : null}
      {action ? <div className="mt-2">{action}</div> : null}
    </div>
  )
}
