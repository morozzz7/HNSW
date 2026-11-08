import type { ReactNode } from 'react'
import { cn } from '@/lib/cn'

interface CardProps {
  title?: string
  description?: ReactNode
  actions?: ReactNode
  className?: string
  bodyClassName?: string
  children: ReactNode
}

export function Card({
  title,
  description,
  actions,
  className,
  bodyClassName,
  children,
}: CardProps) {
  const hasHeader = Boolean(title || description || actions)
  return (
    <section className={cn('rounded-xl border border-line bg-surface', className)}>
      {hasHeader ? (
        <header className="flex items-start justify-between gap-4 border-b border-line px-5 py-4">
          <div className="min-w-0">
            {title ? <h2 className="text-sm font-medium text-ink">{title}</h2> : null}
            {description ? (
              <p className="mt-1 text-[12px] leading-relaxed text-ink-3">{description}</p>
            ) : null}
          </div>
          {actions ? <div className="flex shrink-0 items-center gap-2">{actions}</div> : null}
        </header>
      ) : null}
      <div className={cn('px-5 py-4', bodyClassName)}>{children}</div>
    </section>
  )
}
