import { ChartAxes, ChartEmpty, ChartLegend } from './ChartParts'
import {
  CHART_HEIGHT,
  CHART_MARGIN,
  CHART_WIDTH,
  chartColor,
  linearScale,
  niceTicks,
} from './chartUtils'

export interface BarChartSeries {
  key: string
  label: string
}

export interface BarChartGroup {
  label: string
  values: Record<string, number>
}

interface BarChartProps {
  groups: BarChartGroup[]
  series: BarChartSeries[]
  xLabel: string
  yLabel: string
  ariaLabel: string
  yFormat?: (value: number) => string
  valueFormat?: (value: number) => string
  emptyMessage?: string
}

const defaultFormat = (value: number) => String(Math.round(value * 1000) / 1000)

export function BarChart({
  groups,
  series,
  xLabel,
  yLabel,
  ariaLabel,
  yFormat = defaultFormat,
  valueFormat = defaultFormat,
  emptyMessage = 'Нет данных для графика',
}: BarChartProps) {
  if (groups.length === 0 || series.length === 0) {
    return <ChartEmpty message={emptyMessage} />
  }

  const { top, right, bottom, left } = CHART_MARGIN
  const plotRight = CHART_WIDTH - right
  const plotBottom = CHART_HEIGHT - bottom

  const maxValue = Math.max(
    1,
    ...groups.flatMap((group) => series.map((item) => group.values[item.key] ?? 0)),
  )
  const yDomain: [number, number] = [0, maxValue * 1.08]
  const yScale = linearScale(yDomain, [plotBottom, top])
  const yTicks = niceTicks(0, yDomain[1], 5)

  const groupWidth = (plotRight - left) / groups.length
  const barWidth = Math.min(28, (groupWidth * 0.68) / series.length)
  const xTicks = groups.map((_, index) => index)
  const xScale = linearScale([-0.5, groups.length - 0.5], [left, plotRight])

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
          xTicks={xTicks}
          yTicks={yTicks}
          xScale={(index) => xScale(index)}
          yScale={yScale}
          xFormat={(index) => groups[Math.round(index)]?.label ?? ''}
          yFormat={yFormat}
          xLabel={xLabel}
          yLabel={yLabel}
        />

        {groups.map((group, groupIndex) => {
          const center = left + groupWidth * (groupIndex + 0.5)
          const totalWidth = barWidth * series.length
          return (
            <g key={group.label}>
              {series.map((item, seriesIndex) => {
                const value = group.values[item.key] ?? 0
                const height = Math.max(1, plotBottom - yScale(value))
                const x = center - totalWidth / 2 + seriesIndex * barWidth
                return (
                  <rect
                    key={item.key}
                    x={x}
                    y={yScale(value)}
                    width={Math.max(2, barWidth - 2)}
                    height={height}
                    rx={2}
                    fill={chartColor(seriesIndex)}
                    fillOpacity={0.85}
                  >
                    <title>{`${group.label} · ${item.label}: ${valueFormat(value)}`}</title>
                  </rect>
                )
              })}
            </g>
          )
        })}
      </svg>

      <ChartLegend
        items={series.map((item, index) => ({
          id: item.key,
          label: item.label,
          color: chartColor(index),
        }))}
      />
    </div>
  )
}
