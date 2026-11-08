export function GraphLegend({ isMultilayer = false }: { isMultilayer?: boolean }) {
  return (
    <div
      role="region"
      aria-label="Обозначения графа"
      className="flex flex-wrap items-center gap-x-4 gap-y-1.5 rounded-lg border border-line bg-surface/90 px-3 py-1.5 text-[11px] backdrop-blur-xs"
    >
      <div className="flex items-center gap-1.5">
        <span className="size-2.5 rounded-full border border-warn bg-warn" aria-hidden />
        <span className="text-ink-2">Входная точка</span>
      </div>

      <div className="flex items-center gap-1.5">
        <span className="size-2.5 rounded-full border border-accent bg-accent" aria-hidden />
        <span className="text-ink-2">Узел спуска (трассировка)</span>
      </div>

      <div className="flex items-center gap-1.5">
        <span className="size-2.5 rounded-full border border-accent bg-accent-soft" aria-hidden />
        <span className="text-ink-2">Результат k-NN</span>
      </div>

      <div className="flex items-center gap-1.5">
        <span className="size-2.5 rounded-full border border-line bg-surface-3" aria-hidden />
        <span className="text-ink-2">Узел графа</span>
      </div>

      {isMultilayer ? (
        <div className="flex items-center gap-1.5">
          <span className="h-0.5 w-3 border-t border-dashed border-accent-ink" aria-hidden />
          <span className="text-ink-2">Межслойная связь</span>
        </div>
      ) : null}
    </div>
  )
}
