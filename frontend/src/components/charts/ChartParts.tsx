import { CHART_HEIGHT, CHART_MARGIN, CHART_WIDTH, clamp } from './chartUtils'

interface ChartAxesProps {
  xTicks: number[]
  yTicks: number[]
  xScale: (value: number) => number
  yScale: (value: number) => number
  xFormat: (value: number) => string
  yFormat: (value: number) => string
  xLabel: string
  yLabel: string
}

export function ChartAxes({
  xTicks,
  yTicks,
  xScale,
  yScale,
  xFormat,
  yFormat,
  xLabel,
  yLabel,
}: ChartAxesProps) {
  const { top, right, bottom, left } = CHART_MARGIN
  const plotRight = CHART_WIDTH - right
  const plotBottom = CHART_HEIGHT - bottom

  return (
    <g>
      {yTicks.map((tick) => (
        <g key={`y-${tick}`}>
          <line
            x1={left}
            x2={plotRight}
            y1={yScale(tick)}
            y2={yScale(tick)}
            stroke="var(--line)"
            strokeWidth={1}
          />
          <text
            x={left - 8}
            y={yScale(tick)}
            textAnchor="end"
            dominantBaseline="middle"
            fill="var(--ink-3)"
            fontSize={11}
            fontFamily="var(--font-mono)"
          >
            {yFormat(tick)}
          </text>
        </g>
      ))}

      {xTicks.map((tick) => (
        <g key={`x-${tick}`}>
          <line
            x1={xScale(tick)}
            x2={xScale(tick)}
            y1={top}
            y2={plotBottom}
            stroke="var(--line)"
            strokeDasharray="2 4"
          />
          <text
            x={xScale(tick)}
            y={plotBottom + 14}
            textAnchor="middle"
            fill="var(--ink-3)"
            fontSize={11}
            fontFamily="var(--font-mono)"
          >
            {xFormat(tick)}
          </text>
        </g>
      ))}

      <line
        x1={left}
        x2={plotRight}
        y1={plotBottom}
        y2={plotBottom}
        stroke="var(--line-strong)"
        strokeWidth={1}
      />
      <line
        x1={left}
        x2={left}
        y1={top}
        y2={plotBottom}
        stroke="var(--line-strong)"
        strokeWidth={1}
      />

      <text
        x={(left + plotRight) / 2}
        y={CHART_HEIGHT - 6}
        textAnchor="middle"
        fill="var(--ink-3)"
        fontSize={11}
      >
        {xLabel}
      </text>
      <text
        transform={`translate(13 ${(top + plotBottom) / 2}) rotate(-90)`}
        textAnchor="middle"
        fill="var(--ink-3)"
        fontSize={11}
      >
        {yLabel}
      </text>
    </g>
  )
}

interface TooltipLine {
  label: string
  value: string
  color?: string
}

interface ChartTooltipProps {
  x: number
  y: number
  title: string
  lines: TooltipLine[]
}

export function ChartTooltip({ x, y, title, lines }: ChartTooltipProps) {
  const width = 190
  const height = 20 + lines.length * 16
  const { top, right, bottom } = CHART_MARGIN
  const flip = x + width + 12 > CHART_WIDTH - right
  const left = flip ? x - width - 12 : x + 12
  const topPos = clamp(y - height / 2, top, CHART_HEIGHT - bottom - height)

  return (
    <g pointerEvents="none">
      <rect
        x={left}
        y={topPos}
        width={width}
        height={height}
        rx={6}
        fill="var(--surface)"
        stroke="var(--line-strong)"
      />
      <text x={left + 10} y={topPos + 14} fill="var(--ink)" fontSize={12} fontWeight={500}>
        {title}
      </text>
      {lines.map((line, index) => (
        <g key={line.label} transform={`translate(0 ${topPos + 20 + index * 16})`}>
          {line.color ? (
            <circle cx={left + 13} cy={7} r={3} fill={line.color} />
          ) : (
            <circle cx={left + 13} cy={7} r={3} fill="var(--ink-3)" />
          )}
          <text x={left + 22} y={11} fill="var(--ink-3)" fontSize={11}>
            {line.label}
          </text>
          <text
            x={left + width - 10}
            y={11}
            textAnchor="end"
            fill="var(--ink)"
            fontSize={11}
            fontFamily="var(--font-mono)"
          >
            {line.value}
          </text>
        </g>
      ))}
    </g>
  )
}

interface ChartLegendProps {
  items: { id: string; label: string; color: string; hidden?: boolean }[]
  onToggle?: (id: string) => void
}

export function ChartLegend({ items, onToggle }: ChartLegendProps) {
  return (
    <ul className="flex flex-wrap items-center gap-x-4 gap-y-1.5">
      {items.map((item) => (
        <li key={item.id}>
          {onToggle ? (
            <button
              type="button"
              onClick={() => onToggle(item.id)}
              aria-pressed={!item.hidden}
              className="flex items-center gap-1.5 text-[11px] text-ink-3 transition-opacity hover:text-ink aria-pressed:opacity-100 opacity-45"
            >
              <span
                aria-hidden
                className="size-2 rounded-full"
                style={{ backgroundColor: item.color }}
              />
              <span className="font-mono">{item.label}</span>
            </button>
          ) : (
            <span className="flex items-center gap-1.5 text-[11px] text-ink-3">
              <span
                aria-hidden
                className="size-2 rounded-full"
                style={{ backgroundColor: item.color }}
              />
              <span className="font-mono">{item.label}</span>
            </span>
          )}
        </li>
      ))}
    </ul>
  )
}

export function ChartEmpty({ message }: { message: string }) {
  return (
    <div className="flex h-[240px] items-center justify-center rounded-lg border border-dashed border-line text-[12px] text-ink-3">
      {message}
    </div>
  )
}
