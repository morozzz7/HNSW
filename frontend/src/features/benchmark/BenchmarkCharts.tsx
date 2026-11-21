import { useState } from 'react'
import type { BenchmarkResult } from '@/api/schemas'
import { BarChart } from '@/components/charts/BarChart'
import { LineChart } from '@/components/charts/LineChart'
import { ScatterChart } from '@/components/charts/ScatterChart'
import { Select } from '@/components/ui/Select'
import { formatDecimal, formatInt, formatMs, formatPercent } from '@/lib/format'
import { ChartCard } from './ChartCard'
import {
  buildTimeChart,
  efSearchValues,
  groupByConfig,
  latencySeries,
  paretoPoints,
  recallSeries,
} from './derive'

interface BenchmarkChartsProps {
  results: BenchmarkResult[]
  targetRecall: number
  jobId: string
}

export function BenchmarkCharts({ results, targetRecall, jobId }: BenchmarkChartsProps) {
  const [configId, setConfigId] = useState<string | null>(null)

  const efTicks = efSearchValues(results)
  const groups = groupByConfig(results)
  const activeGroup = groups.find((group) => group.id === configId) ?? groups[0]
  const latencyRows = activeGroup?.results ?? []
  const pareto = paretoPoints(results)
  const buildTime = buildTimeChart(results)
  const prefix = `benchmark_${jobId}`

  return (
    <div className="flex flex-col gap-4">
      <ChartCard
        title="Recall@k от efSearch"
        description="Точность относительно полного перебора; серия — пара M и efConstruction."
        filename={`${prefix}_recall.png`}
      >
        <LineChart
          series={recallSeries(results)}
          xTicks={efTicks}
          xLog
          xLabel="efSearch"
          yLabel="recall@k"
          ariaLabel="Recall@k в зависимости от efSearch"
          yDomain={[0, 1]}
          xFormat={(value) => String(value)}
          yFormat={(value) => formatPercent(value, 0)}
        />
      </ChartCard>

      <ChartCard
        title="QPS от recall"
        description="Кривая Парето: точки с максимальной скоростью при достигнутой точности."
        filename={`${prefix}_pareto.png`}
      >
        <ScatterChart
          points={pareto.points}
          frontier={pareto.frontier}
          xLabel="recall@k"
          yLabel="QPS"
          ariaLabel="QPS в зависимости от recall с кривой Парето"
          targetX={targetRecall}
          xFormat={(value) => formatPercent(value, 0)}
          yFormat={(value) => formatInt(value)}
        />
      </ChartCard>

      <ChartCard
        title="Задержка p50 / p95 / p99"
        description="Перцентили времени запроса для выбранной конфигурации."
        filename={`${prefix}_latency.png`}
        toolbar={
          <Select
            value={activeGroup?.id ?? ''}
            onChange={(event) => setConfigId(event.target.value)}
            className="h-7 w-[132px] text-[12px]"
            aria-label="Конфигурация для графика задержек"
          >
            {groups.map((group) => (
              <option key={group.id} value={group.id}>
                {group.label}
              </option>
            ))}
          </Select>
        }
      >
        <LineChart
          series={latencySeries(latencyRows)}
          xTicks={efTicks}
          xLog
          yLog
          xLabel="efSearch"
          yLabel="задержка, мс"
          ariaLabel="Задержка p50, p95 и p99 в зависимости от efSearch"
          toggleable={false}
          xFormat={(value) => String(value)}
          yFormat={(value) => `${formatDecimal(value, 2)} мс`}
        />
      </ChartCard>

      <ChartCard
        title="Время построения"
        description="Зависимость времени сборки от M и efConstruction."
        filename={`${prefix}_build_time.png`}
      >
        <BarChart
          groups={buildTime.groups}
          series={buildTime.series}
          xLabel="M"
          yLabel="время, мс"
          ariaLabel="Время построения индекса по M и efConstruction"
          yFormat={(value) => formatMs(value)}
          valueFormat={(value) => formatMs(value)}
        />
      </ChartCard>
    </div>
  )
}
