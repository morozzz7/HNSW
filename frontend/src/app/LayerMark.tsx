export function LayerMark({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden className={className}>
      <g fill="currentColor">
        <circle cx="12" cy="4.5" r="2" />
        <circle cx="6" cy="12" r="1.7" />
        <circle cx="18" cy="12" r="1.7" />
        <circle cx="4" cy="19.5" r="1.4" />
        <circle cx="12" cy="19.5" r="1.4" />
        <circle cx="20" cy="19.5" r="1.4" />
      </g>
      <g stroke="currentColor" strokeOpacity="0.45" strokeWidth="1.1" fill="none">
        <path d="M12 4.5 6 12m6-7.5 6 7.5M6 12l-2 7.5M6 12l6 7.5m6-7.5-6 7.5m6-7.5 2 7.5" />
      </g>
    </svg>
  )
}
