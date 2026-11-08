import type { InputHTMLAttributes, ReactNode } from 'react'
import { cn } from '@/lib/cn'

interface FieldProps {
  label: string
  htmlFor: string
  hint?: ReactNode
  hintAction?: ReactNode
  error?: string
  description?: ReactNode
  className?: string
  children: ReactNode
}

export function Field({
  label,
  htmlFor,
  hint,
  hintAction,
  error,
  description,
  className,
  children,
}: FieldProps) {
  return (
    <div className={cn('flex flex-col gap-1.5', className)}>
      <div className="flex items-center gap-1.5">
        <label htmlFor={htmlFor} className="text-[13px] font-medium text-ink">
          {label}
        </label>
        {hintAction}
      </div>
      {children}
      {description ? <p className="text-[12px] leading-relaxed text-ink-3">{description}</p> : null}
      {error ? (
        <p className="text-[12px] leading-relaxed text-danger" role="alert">
          {error}
        </p>
      ) : (
        hint
      )}
    </div>
  )
}

export function Input({ className, ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      className={cn(
        'h-9 w-full rounded-lg border border-line bg-surface px-3 text-sm text-ink transition-colors placeholder:text-ink-3 hover:border-line-strong focus:border-accent disabled:opacity-60',
        className,
      )}
      {...props}
    />
  )
}
