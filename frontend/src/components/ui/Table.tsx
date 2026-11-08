import type { ReactNode, TdHTMLAttributes, ThHTMLAttributes } from 'react'
import { cn } from '@/lib/cn'

export function Table({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div className={cn('w-full overflow-x-auto', className)}>
      <table className="w-full border-collapse text-left text-sm">{children}</table>
    </div>
  )
}

interface CellProps extends ThHTMLAttributes<HTMLTableCellElement> {
  numeric?: boolean
}

export function Th({ numeric, className, children, ...props }: CellProps) {
  return (
    <th
      scope="col"
      className={cn(
        'whitespace-nowrap border-b border-line px-3 py-2 text-[12px] font-medium text-ink-3 first:pl-0 last:pr-0',
        numeric && 'text-right',
        className,
      )}
      {...props}
    >
      {children}
    </th>
  )
}

interface DataCellProps extends TdHTMLAttributes<HTMLTableCellElement> {
  numeric?: boolean
}

export function Td({ numeric, className, children, ...props }: DataCellProps) {
  return (
    <td
      className={cn(
        'border-b border-line px-3 py-2 text-[13px] text-ink-2 first:pl-0 last:pr-0',
        numeric && 'text-right font-mono tabular-nums',
        className,
      )}
      {...props}
    >
      {children}
    </td>
  )
}
