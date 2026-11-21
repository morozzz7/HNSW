import { Play } from 'lucide-react'
import { useState } from 'react'
import { errorMessage } from '@/api/errors'
import {
  MAX_BENCHMARK_GRID,
  MAX_BENCHMARK_QUERIES,
  MAX_EF_CONSTRUCTION,
  MAX_EF_SEARCH,
  MAX_K,
  MAX_M,
  MIN_BENCHMARK_QUERIES,
  MIN_K,
  MIN_M,
  benchmarkInputSchema,
  type BenchmarkInput,
} from '@/api/schemas'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { ErrorBanner } from '@/components/ui/ErrorBanner'
import { Field, Input } from '@/components/ui/Field'
import { formatInt } from '@/lib/format'

type FieldName = 'M' | 'efConstruction' | 'efSearch' | 'k' | 'queries'
type FieldErrors = Partial<Record<FieldName, string>>

interface BenchmarkFormProps {
  nodes: number
  indexId: string
  running: boolean
  error: unknown
  onSubmit: (input: BenchmarkInput) => void
}

interface Preset {
  id: string
  label: string
  M: string
  efConstruction: string
  efSearch: string
  k: string
  queries: string
}

const PRESETS: Preset[] = [
  {
    id: 'fast',
    label: 'Быстрый',
    M: '16',
    efConstruction: '100',
    efSearch: '10, 50, 100',
    k: '10',
    queries: '100',
  },
  {
    id: 'standard',
    label: 'Стандартный',
    M: '8, 16, 32',
    efConstruction: '100, 200',
    efSearch: '10, 50, 100, 200',
    k: '10',
    queries: '100',
  },
  {
    id: 'wide',
    label: 'Широкий',
    M: '8, 16, 32, 64',
    efConstruction: '100, 200, 400',
    efSearch: '10, 25, 50, 100, 200',
    k: '10',
    queries: '200',
  },
]

function parseList(text: string): { values: number[]; invalid: boolean } {
  const parts = text
    .split(/[\s,;]+/)
    .map((part) => part.trim())
    .filter(Boolean)
  const numbers = parts.map(Number)
  return {
    values: [...new Set(numbers.filter((value) => Number.isFinite(value)))].sort((a, b) => a - b),
    invalid: numbers.some((value) => !Number.isFinite(value)),
  }
}

export function BenchmarkForm({ nodes, indexId, running, error, onSubmit }: BenchmarkFormProps) {
  const [mText, setMText] = useState(PRESETS[1]?.M ?? '16')
  const [efConstructionText, setEfConstructionText] = useState(PRESETS[1]?.efConstruction ?? '200')
  const [efSearchText, setEfSearchText] = useState(PRESETS[1]?.efSearch ?? '50')
  const [kText, setKText] = useState(PRESETS[1]?.k ?? '10')
  const [queriesText, setQueriesText] = useState(PRESETS[1]?.queries ?? '100')
  const [errors, setErrors] = useState<FieldErrors>({})

  const mList = parseList(mText)
  const efConstructionList = parseList(efConstructionText)
  const efSearchList = parseList(efSearchText)
  const combinations =
    mList.values.length * efConstructionList.values.length * efSearchList.values.length

  const applyPreset = (preset: Preset) => {
    setMText(preset.M)
    setEfConstructionText(preset.efConstruction)
    setEfSearchText(preset.efSearch)
    setKText(preset.k)
    setQueriesText(preset.queries)
    setErrors({})
  }

  const submit = () => {
    const next: FieldErrors = {}
    if (mList.invalid) {
      next.M = 'Введите целые числа через запятую'
    }
    if (efConstructionList.invalid) {
      next.efConstruction = 'Введите целые числа через запятую'
    }
    if (efSearchList.invalid) {
      next.efSearch = 'Введите целые числа через запятую'
    }
    if (Object.keys(next).length > 0) {
      setErrors(next)
      return
    }

    const parsed = benchmarkInputSchema.safeParse({
      M: mList.values,
      efConstruction: efConstructionList.values,
      efSearch: efSearchList.values,
      k: Number(kText),
      queries: Number(queriesText),
      indexId,
    })

    if (!parsed.success) {
      for (const issue of parsed.error.issues) {
        const key = String(issue.path[0] ?? '') as FieldName
        if (key && !next[key]) {
          next[key] = issue.message
        }
      }
      setErrors(next)
      return
    }

    if (parsed.data.k > nodes) {
      setErrors({ k: `k не больше числа узлов индекса (${formatInt(nodes)})` })
      return
    }

    setErrors({})
    onSubmit(parsed.data)
  }

  const overLimit = combinations > MAX_BENCHMARK_GRID

  return (
    <Card
      title="Сетка параметров"
      description="Метрика и эвристика берутся из текущего индекса. Каждая комбинация — отдельное построение графа."
      actions={<Badge tone="neutral">{indexId}</Badge>}
    >
      <form
        className="flex flex-col gap-4"
        noValidate
        onSubmit={(event) => {
          event.preventDefault()
          submit()
        }}
      >
        <div className="flex flex-wrap gap-1.5">
          {PRESETS.map((preset) => (
            <Button
              key={preset.id}
              type="button"
              variant="secondary"
              size="sm"
              disabled={running}
              onClick={() => applyPreset(preset)}
            >
              {preset.label}
            </Button>
          ))}
        </div>

        <Field
          label="M"
          htmlFor="benchmark-m"
          error={errors.M}
          description={`Через запятую, от ${MIN_M} до ${MAX_M}`}
        >
          <Input
            id="benchmark-m"
            value={mText}
            disabled={running}
            onChange={(event) => setMText(event.target.value)}
            className="font-mono text-xs"
          />
        </Field>

        <Field
          label="efConstruction"
          htmlFor="benchmark-ef-construction"
          error={errors.efConstruction}
          description={`Через запятую, от 1 до ${formatInt(MAX_EF_CONSTRUCTION)}`}
        >
          <Input
            id="benchmark-ef-construction"
            value={efConstructionText}
            disabled={running}
            onChange={(event) => setEfConstructionText(event.target.value)}
            className="font-mono text-xs"
          />
        </Field>

        <Field
          label="efSearch"
          htmlFor="benchmark-ef-search"
          error={errors.efSearch}
          description={`Через запятую, от 1 до ${formatInt(MAX_EF_SEARCH)}`}
        >
          <Input
            id="benchmark-ef-search"
            value={efSearchText}
            disabled={running}
            onChange={(event) => setEfSearchText(event.target.value)}
            className="font-mono text-xs"
          />
        </Field>

        <div className="grid grid-cols-2 gap-3">
          <Field
            label="k"
            htmlFor="benchmark-k"
            error={errors.k}
            description={`от ${MIN_K} до ${MAX_K}`}
          >
            <Input
              id="benchmark-k"
              type="number"
              inputMode="numeric"
              min={MIN_K}
              max={MAX_K}
              value={kText}
              disabled={running}
              onChange={(event) => setKText(event.target.value)}
              className="font-mono"
            />
          </Field>

          <Field
            label="queries"
            htmlFor="benchmark-queries"
            error={errors.queries}
            description={`${MIN_BENCHMARK_QUERIES}–${formatInt(MAX_BENCHMARK_QUERIES)}`}
          >
            <Input
              id="benchmark-queries"
              type="number"
              inputMode="numeric"
              min={MIN_BENCHMARK_QUERIES}
              max={MAX_BENCHMARK_QUERIES}
              value={queriesText}
              disabled={running}
              onChange={(event) => setQueriesText(event.target.value)}
              className="font-mono"
            />
          </Field>
        </div>

        <div className="flex items-center justify-between gap-3 rounded-lg border border-line bg-surface-2 px-3 py-2">
          <span className="text-[12px] text-ink-3">Комбинаций</span>
          <span
            className={`font-mono text-[12px] tabular-nums ${overLimit ? 'text-danger' : 'text-ink'}`}
          >
            {formatInt(combinations)} / {MAX_BENCHMARK_GRID}
          </span>
        </div>

        {overLimit ? (
          <p role="alert" className="text-[12px] text-danger">
            Сетка больше {MAX_BENCHMARK_GRID} комбинаций — сократите списки.
          </p>
        ) : null}

        {error ? (
          <ErrorBanner title="Не удалось запустить бенчмарк" message={errorMessage(error)} />
        ) : null}

        <div className="flex justify-end">
          <Button
            type="submit"
            size="sm"
            loading={running}
            disabled={running || overLimit || combinations === 0}
          >
            <Play aria-hidden className="size-4" />
            {running ? 'Идёт прогон' : 'Запустить бенчмарк'}
          </Button>
        </div>
      </form>
    </Card>
  )
}
