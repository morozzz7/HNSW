import { Info } from 'lucide-react'
import { useId, useState } from 'react'
import type { ReactNode } from 'react'
import { cn } from '@/lib/cn'

interface HintProps {
  label: string
  children: ReactNode
  className?: string
}

export function Hint({ label, children, className }: HintProps) {
  const [open, setOpen] = useState(false)
  const id = useId()

  return (
    <span className={cn('relative inline-flex', className)}>
      <button
        type="button"
        aria-label={label}
        aria-expanded={open}
        aria-describedby={open ? id : undefined}
        className="inline-flex size-5 items-center justify-center rounded-md text-ink-3 transition-colors hover:bg-surface-2 hover:text-ink-2"
        onMouseEnter={() => setOpen(true)}
        onMouseLeave={() => setOpen(false)}
        onFocus={() => setOpen(true)}
        onBlur={() => setOpen(false)}
        onClick={() => setOpen((value) => !value)}
      >
        <Info aria-hidden className="size-3.5" />
      </button>
      {open ? (
        <span
          role="tooltip"
          id={id}
          className="absolute left-1/2 top-full z-30 mt-2 w-64 -translate-x-1/2 rounded-lg border border-line bg-surface-2 px-3 py-2 text-[12px] leading-relaxed text-ink-2 shadow-lg shadow-black/10"
        >
          {children}
        </span>
      ) : null}
    </span>
  )
}
