import { useState } from 'react'
import { ChartAxes, ChartEmpty, ChartLegend, ChartTooltip } from './ChartParts'
import {
  CHART_HEIGHT,
  CHART_MARGIN,
  CHART_WIDTH,
  chartColor,
  clamp,
  linePath,
  linearScale,
  logScale,
  logTicks,
  niceTicks,
  paddedDomain,
} from './chartUtils'

export interface LineChartSeries {
  id: string
  label: string
  points: { x: number; y: number }[]
}

interface LineChartProps {
  series: LineChartSeries[]
  xTicks: number[]
  xLabel: string
  yLabel: string
  ariaLabel: string
  yDomain?: [number, number]
  xLog?: boolean
  yLog?: boolean
  xFormat?: (value: number) => string
  yFormat?: (value: number) => string
  emptyMessage?: string
  toggleable?: boolean
}

const defaultFormat = (value: number) => String(Math.round(value * 1000) / 1000)

export function LineChart({
  series,
  xTicks,
  xLabel,
  yLabel,
  ariaLabel,
  yDomain,
  xLog = false,
  yLog = false,
  xFormat = defaultFormat,
  yFormat = defaultFormat,
  emptyMessage = 'Нет данных для графика',
  toggleable = true,
}: LineChartProps) {
  const [hidden, setHidden] = useState<Set<string>>(new Set())
  const [hoveredX, setHoveredX] = useState<number | null>(null)

  const { top, right, bottom, left } = CHART_MARGIN
  const plotRight = CHART_WIDTH - right
  const plotBottom = CHART_HEIGHT - bottom
  const plotWidth = plotRight - left
  const plotHeight = plotBottom - top

  const visible = series.filter((item) => !hidden.has(item.id))

  if (xTicks.length === 0 || series.length === 0) {
    return <ChartEmpty message={emptyMessage} />
  }

  const tickMin = Math.min(...xTicks)
  const tickMax = Math.max(...xTicks)
  const xDomain: [number, number] =
    tickMin === tickMax
      ? xLog
        ? [tickMin / 1.5, tickMax * 1.5]
        : [tickMin - 1, tickMax + 1]
      : [tickMin, tickMax]
  const xScale = xLog
    ? logScale(xDomain, [left, plotRight])
    : linearScale(xDomain, [left, plotRight])

  const values = visible.flatMap((item) => item.points.map((point) => point.y))
  let resolvedYDomain: [number, number]
  if (yDomain) {
    resolvedYDomain = yDomain
  } else if (yLog) {
    const min = Math.max(1e-3, Math.min(...values))
    const max = Math.max(min * 1.1, Math.max(...values))
    resolvedYDomain = [min / 1.4, max * 1.4]
  } else {
    resolvedYDomain = paddedDomain(values)
    if (resolvedYDomain[0] < 0 && Math.min(...values) >= 0) {
      resolvedYDomain = [0, resolvedYDomain[1]]
    }
  }
  const yScale = yLog
    ? logScale(resolvedYDomain, [plotBottom, top])
    : linearScale(resolvedYDomain, [plotBottom, top])
  const yTicks = yLog
    ? logTicks(resolvedYDomain[0], resolvedYDomain[1])
    : niceTicks(resolvedYDomain[0], resolvedYDomain[1], 5)

  const hoveredPoints = visible.flatMap((item) => {
    const point = item.points.find((candidate) => candidate.x === hoveredX)
    return point ? [{ series: item, point }] : []
  })
  const hoveredY =
    hoveredPoints.length > 0
      ? hoveredPoints.reduce((sum, entry) => sum + entry.point.y, 0) / hoveredPoints.length
      : (resolvedYDomain[0] + resolvedYDomain[1]) / 2

  const toggle = (id: string) => {
    setHidden((previous) => {
      const next = new Set(previous)
      if (next.has(id)) {
        next.delete(id)
      } else if (next.size < series.length - 1) {
        next.add(id)
      }
      return next
    })
  }

  return (
    <div className="flex flex-col gap-3">
      {visible.length === 0 ? (
        <ChartEmpty message="Все серии скрыты" />
      ) : (
        <svg
          role="img"
          aria-label={ariaLabel}
          viewBox={`0 0 ${CHART_WIDTH} ${CHART_HEIGHT}`}
          xmlns="http://www.w3.org/2000/svg"
          className="h-auto w-full"
        >
          <ChartAxes
            xTicks={xTicks}
            yTicks={yTicks}
            xScale={xScale}
            yScale={yScale}
            xFormat={xFormat}
            yFormat={yFormat}
            xLabel={xLabel}
            yLabel={yLabel}
          />

          {visible.map((item) => {
            const color = chartColor(series.findIndex((candidate) => candidate.id === item.id))
            return (
              <g key={item.id}>
                <path
                  d={linePath(item.points, xScale, yScale)}
                  fill="none"
                  stroke={color}
                  strokeWidth={2}
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
                {item.points.map((point) => (
                  <circle
                    key={`${item.id}-${point.x}`}
                    cx={xScale(point.x)}
                    cy={yScale(point.y)}
                    r={2.5}
                    fill={color}
                  />
                ))}
              </g>
            )
          })}

          {hoveredX !== null ? (
            <line
              x1={xScale(hoveredX)}
              x2={xScale(hoveredX)}
              y1={top}
              y2={plotBottom}
              stroke="var(--line-strong)"
              strokeDasharray="3 3"
            />
          ) : null}

          {hoveredX !== null && hoveredPoints.length > 0 ? (
            <ChartTooltip
              x={xScale(hoveredX)}
              y={yScale(hoveredY)}
              title={`${xLabel}: ${xFormat(hoveredX)}`}
              lines={hoveredPoints.map((entry) => ({
                label: entry.series.label,
                value: yFormat(entry.point.y),
                color: chartColor(
                  series.findIndex((candidate) => candidate.id === entry.series.id),
                ),
              }))}
            />
          ) : null}

          <rect
            x={left}
            y={top}
            width={plotWidth}
            height={plotHeight}
            fill="transparent"
            onMouseMove={(event) => {
              const bounds = event.currentTarget.getBoundingClientRect()
              if (bounds.width === 0) {
                return
              }
              const px = ((event.clientX - bounds.left) / bounds.width) * CHART_WIDTH
              let nearest = xTicks[0] ?? 0
              let bestDistance = Number.POSITIVE_INFINITY
              for (const tick of xTicks) {
                const distance = Math.abs(xScale(tick) - px)
                if (distance < bestDistance) {
                  bestDistance = distance
                  nearest = tick
                }
              }
              setHoveredX(clamp(nearest, xDomain[0], xDomain[1]))
            }}
            onMouseLeave={() => setHoveredX(null)}
          />
        </svg>
      )}

      <ChartLegend
        items={series.map((item, index) => ({
          id: item.id,
          label: item.label,
          color: chartColor(index),
          hidden: hidden.has(item.id),
        }))}
        onToggle={toggleable && series.length > 1 ? toggle : undefined}
      />
    </div>
  )
}
