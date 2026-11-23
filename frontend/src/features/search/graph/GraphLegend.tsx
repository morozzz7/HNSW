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
        <span className="size-2.5 rounded-full border border-[#22d3ee] bg-[#22d3ee]" aria-hidden />
        <span className="text-ink-2">Текущий узел маршрута</span>
      </div>

      <div className="flex items-center gap-1.5">
        <span className="size-2.5 rounded-full border border-[#8b5cf6] bg-[#8b5cf6]" aria-hidden />
        <span className="text-ink-2">Пройденный маршрут</span>
      </div>

      <div className="flex items-center gap-1.5">
        <span className="size-2.5 rounded-full border border-[#ec4899] bg-[#ec4899]" aria-hidden />
        <span className="text-ink-2">Результат k-NN</span>
      </div>

      <div className="flex items-center gap-1.5">
        <span className="size-2.5 rounded-full border border-line bg-surface-3" aria-hidden />
        <span className="text-ink-2">Узел графа</span>
      </div>

      <div className="flex items-center gap-1.5">
        <span className="text-ink-3">Индекс узла — по клику</span>
      </div>

      {isMultilayer ? (
        <div className="flex items-center gap-1.5">
          <span className="h-0.5 w-3 border-t border-dashed border-line-strong" aria-hidden />
          <span className="text-ink-2">Межслойная связь</span>
        </div>
      ) : null}
    </div>
  )
}
