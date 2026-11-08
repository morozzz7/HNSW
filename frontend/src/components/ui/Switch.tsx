import { cn } from '@/lib/cn'

interface SwitchProps {
  id?: string
  checked: boolean
  onCheckedChange: (checked: boolean) => void
  disabled?: boolean
  'aria-describedby'?: string
}

export function Switch({ id, checked, onCheckedChange, disabled, ...props }: SwitchProps) {
  return (
    <button
      type="button"
      id={id}
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => onCheckedChange(!checked)}
      className={cn(
        'relative inline-flex h-5 w-9 shrink-0 items-center rounded-full border transition-colors disabled:opacity-50',
        checked ? 'border-accent bg-accent' : 'border-line bg-surface-3',
      )}
      {...props}
    >
      <span
        aria-hidden
        className={cn(
          'absolute left-0.5 size-3.5 rounded-full transition-transform',
          checked ? 'translate-x-4 bg-accent-ink' : 'translate-x-0 bg-ink-2',
        )}
      />
    </button>
  )
}
