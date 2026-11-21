import { useState } from 'react'
import { ChartAxes, ChartEmpty, ChartLegend, ChartTooltip } from './ChartParts'
import {
  CHART_HEIGHT,
  CHART_MARGIN,
  CHART_WIDTH,
  chartColor,
  linearScale,
  logScale,
  logTicks,
  niceTicks,
} from './chartUtils'

export interface ScatterChartPoint {
  id: string
  x: number
  y: number
  label: string
  frontier?: boolean
}

interface ScatterChartProps {
  points: ScatterChartPoint[]
  frontier: ScatterChartPoint[]
  xLabel: string
  yLabel: string
  ariaLabel: string
  targetX?: number
  xDomain?: [number, number]
  yLog?: boolean
  xFormat?: (value: number) => string
  yFormat?: (value: number) => string
}

const defaultFormat = (value: number) => String(Math.round(value * 1000) / 1000)

export function ScatterChart({
  points,
  frontier,
  xLabel,
  yLabel,
  ariaLabel,
  targetX,
  xDomain = [0, 1],
  yLog = true,
  xFormat = defaultFormat,
  yFormat = defaultFormat,
}: ScatterChartProps) {
  const [hoveredId, setHoveredId] = useState<string | null>(null)

  if (points.length === 0) {
    return <ChartEmpty message="Нет данных для графика" />
  }

  const { top, right, bottom, left } = CHART_MARGIN
  const plotRight = CHART_WIDTH - right
  const plotBottom = CHART_HEIGHT - bottom

  const xScale = linearScale(xDomain, [left, plotRight])
  const yValues = points.map((point) => point.y)
  const yMin = Math.max(1e-3, Math.min(...yValues))
  const yMax = Math.max(yMin * 1.1, Math.max(...yValues))
  const yDomain: [number, number] = [yMin / 1.4, yMax * 1.4]
  const yScale = yLog
    ? logScale(yDomain, [plotBottom, top])
    : linearScale(yDomain, [plotBottom, top])
  const yTicks = yLog ? logTicks(yDomain[0], yDomain[1]) : niceTicks(yDomain[0], yDomain[1], 5)

  const orderedFrontier = [...frontier].sort((a, b) => a.x - b.x)
  const hovered = points.find((point) => point.id === hoveredId) ?? null

  return (
    <div className="flex flex-col gap-3">
      <svg
        role="img"
        aria-label={ariaLabel}
        viewBox={`0 0 ${CHART_WIDTH} ${CHART_HEIGHT}`}
        xmlns="http://www.w3.org/2000/svg"
        className="h-auto w-full"
      >
        <ChartAxes
          xTicks={[0, 0.25, 0.5, 0.75, 1]}
          yTicks={yTicks}
          xScale={xScale}
          yScale={yScale}
          xFormat={xFormat}
          yFormat={yFormat}
          xLabel={xLabel}
          yLabel={yLabel}
        />

        {targetX !== undefined ? (
          <g>
            <line
              x1={xScale(targetX)}
              x2={xScale(targetX)}
              y1={top}
              y2={plotBottom}
              stroke="var(--warn)"
              strokeDasharray="4 4"
              strokeWidth={1}
            />
            <text
              x={xScale(targetX) + 4}
              y={top + 10}
              fill="var(--warn)"
              fontSize={10}
              fontFamily="var(--font-mono)"
            >
              recall ≥ {xFormat(targetX)}
            </text>
          </g>
        ) : null}

        {orderedFrontier.length > 1 ? (
          <polyline
            points={orderedFrontier
              .map((point) => `${xScale(point.x)},${yScale(point.y)}`)
              .join(' ')}
            fill="none"
            stroke="var(--warn)"
            strokeWidth={1.5}
            strokeDasharray="5 4"
          />
        ) : null}

        {points.map((point) => (
          <circle
            key={point.id}
            cx={xScale(point.x)}
            cy={yScale(point.y)}
            r={point.frontier ? 4.5 : 3}
            fill={point.frontier ? 'var(--warn)' : 'var(--accent)'}
            fillOpacity={hovered && hovered.id !== point.id ? 0.5 : 0.9}
            onMouseEnter={() => setHoveredId(point.id)}
            onMouseLeave={() => setHoveredId(null)}
          />
        ))}

        {hovered ? (
          <ChartTooltip
            x={xScale(hovered.x)}
            y={yScale(hovered.y)}
            title={hovered.label}
            lines={[
              { label: 'recall', value: xFormat(hovered.x), color: 'var(--accent)' },
              { label: 'QPS', value: yFormat(hovered.y), color: 'var(--accent)' },
              ...(hovered.frontier
                ? [{ label: 'Парето-фронт', value: 'да', color: 'var(--warn)' }]
                : []),
            ]}
          />
        ) : null}
      </svg>

      <ChartLegend
        items={[
          { id: 'all', label: 'Все прогоны', color: chartColor(0) },
          { id: 'frontier', label: 'Парето-фронт', color: chartColor(1) },
        ]}
      />
    </div>
  )
}
