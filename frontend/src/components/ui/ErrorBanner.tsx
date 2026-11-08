import { AlertTriangle } from 'lucide-react'
import type { ReactNode } from 'react'
import { cn } from '@/lib/cn'

interface ErrorBannerProps {
  title: string
  message?: ReactNode
  code?: string
  action?: ReactNode
  className?: string
}

export function ErrorBanner({ title, message, code, action, className }: ErrorBannerProps) {
  return (
    <div
      role="alert"
      className={cn(
        'flex items-start gap-3 rounded-lg border border-danger/40 bg-danger-soft px-4 py-3',
        className,
      )}
    >
      <AlertTriangle aria-hidden className="mt-0.5 size-4 shrink-0 text-danger" />
      <div className="min-w-0 flex-1">
        <p className="text-[13px] font-medium text-danger">{title}</p>
        {message ? (
          <p className="mt-0.5 text-[12px] leading-relaxed text-ink-2">{message}</p>
        ) : null}
        {code ? <p className="mt-1 font-mono text-[11px] text-ink-3">{code}</p> : null}
      </div>
      {action ? <div className="shrink-0">{action}</div> : null}
    </div>
  )
}
