import type { ReactNode } from 'react'

interface PageHeaderProps {
  title: string
  description?: ReactNode
  actions?: ReactNode
}

export function PageHeader({ title, description, actions }: PageHeaderProps) {
  return (
    <header className="flex flex-col gap-3">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-[22px] font-medium leading-tight tracking-tight text-ink">{title}</h1>
          {description ? (
            <p className="mt-1 max-w-2xl text-[13px] leading-relaxed text-ink-3">{description}</p>
          ) : null}
        </div>
        {actions ? <div className="flex items-center gap-2">{actions}</div> : null}
      </div>
      <div aria-hidden className="tick-rule h-1 w-full opacity-70" />
    </header>
  )
}
