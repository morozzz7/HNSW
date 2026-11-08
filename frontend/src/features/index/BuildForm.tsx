import { Play } from 'lucide-react'
import { useState } from 'react'
import { errorMessage } from '@/api/errors'
import {
  MAX_EF_CONSTRUCTION,
  MAX_M,
  MIN_M,
  buildIndexInputSchema,
  type BuildIndexInput,
  type DatasetListItem,
  type Metric,
} from '@/api/schemas'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { ErrorBanner } from '@/components/ui/ErrorBanner'
import { Field, Input } from '@/components/ui/Field'
import { Hint } from '@/components/ui/Hint'
import { Select } from '@/components/ui/Select'
import { Switch } from '@/components/ui/Switch'
import { formatInt } from '@/lib/format'
import { METRIC_LABELS } from './labels'

type FieldName = 'datasetId' | 'M' | 'efConstruction' | 'metric' | 'seed'
type FieldErrors = Partial<Record<FieldName, string>>

interface BuildFormProps {
  datasets: DatasetListItem[]
  building: boolean
  error: unknown
  onSubmit: (input: BuildIndexInput) => void
}

export function BuildForm({ datasets, building, error, onSubmit }: BuildFormProps) {
  const [datasetId, setDatasetId] = useState<string | null>(null)
  const [m, setM] = useState('16')
  const [efConstruction, setEfConstruction] = useState('200')
  const [metric, setMetric] = useState<Metric>('l2')
  const [heuristic, setHeuristic] = useState(true)
  const [seed, setSeed] = useState('42')
  const [errors, setErrors] = useState<FieldErrors>({})

  const activeId = datasetId ?? datasets[0]?.id ?? ''
  const activeDataset = datasets.find((dataset) => dataset.id === activeId)

  const submit = () => {
    const parsed = buildIndexInputSchema.safeParse({
      datasetId: activeId,
      M: Number(m),
      efConstruction: Number(efConstruction),
      metric,
      heuristic,
      seed: Number(seed),
    })
    if (!parsed.success) {
      const next: FieldErrors = {}
      for (const issue of parsed.error.issues) {
        const key = String(issue.path[0] ?? '') as FieldName
        if (key && !next[key]) {
          next[key] = issue.message
        }
      }
      setErrors(next)
      return
    }
    setErrors({})
    onSubmit(parsed.data)
  }

  return (
    <Card
      title="Параметры сборки"
      description="Индекс строится асинхронно: прогресс приходит по SSE, статистика появится после готовности."
    >
      <form
        className="flex flex-col gap-4"
        noValidate
        onSubmit={(event) => {
          event.preventDefault()
          submit()
        }}
      >
        <Field
          label="Датасет"
          htmlFor="build-dataset"
          error={errors.datasetId}
          description={
            activeDataset
              ? `${formatInt(activeDataset.n)} векторов · размерность ${formatInt(activeDataset.dim)}`
              : undefined
          }
        >
          <Select
            id="build-dataset"
            value={activeId}
            disabled={datasets.length === 0}
            onChange={(event) => setDatasetId(event.target.value)}
          >
            {datasets.length === 0 ? <option value="">Нет доступных датасетов</option> : null}
            {datasets.map((dataset) => (
              <option key={dataset.id} value={dataset.id}>
                {dataset.id} · {formatInt(dataset.n)} × {formatInt(dataset.dim)}
              </option>
            ))}
          </Select>
        </Field>

        <div className="grid gap-3 sm:grid-cols-2">
          <Field
            label="M"
            htmlFor="build-m"
            error={errors.M}
            hintAction={
              <Hint label="Подсказка о параметре M">
                Максимум соседей на верхних уровнях, от {MIN_M} до {MAX_M}. Больше M — плотнее граф,
                дольше сборка и поиск.
              </Hint>
            }
          >
            <Input
              id="build-m"
              type="number"
              inputMode="numeric"
              min={MIN_M}
              max={MAX_M}
              value={m}
              onChange={(event) => setM(event.target.value)}
            />
          </Field>
          <Field
            label="efConstruction"
            htmlFor="build-ef"
            error={errors.efConstruction}
            hintAction={
              <Hint label="Подсказка об efConstruction">
                Размер списка кандидатов при вставке, от 1 до {formatInt(MAX_EF_CONSTRUCTION)}.
                Больше значение — качественнее граф и медленнее сборка.
              </Hint>
            }
          >
            <Input
              id="build-ef"
              type="number"
              inputMode="numeric"
              min={1}
              max={MAX_EF_CONSTRUCTION}
              value={efConstruction}
              onChange={(event) => setEfConstruction(event.target.value)}
            />
          </Field>
        </div>

        <Field
          label="Метрика"
          htmlFor="build-metric"
          error={errors.metric}
          hintAction={
            <Hint label="Подсказка о метрике">
              Расстояние задаёт геометрию графа. Смена метрики требует пересборки индекса.
            </Hint>
          }
        >
          <Select
            id="build-metric"
            value={metric}
            onChange={(event) => setMetric(event.target.value as Metric)}
          >
            {(Object.keys(METRIC_LABELS) as Metric[]).map((value) => (
              <option key={value} value={value}>
                {METRIC_LABELS[value]}
              </option>
            ))}
          </Select>
        </Field>

        <div className="flex items-start justify-between gap-4 rounded-lg border border-line bg-surface-2 px-3 py-2.5">
          <div className="min-w-0">
            <div className="flex items-center gap-1.5">
              <label htmlFor="build-heuristic" className="text-[13px] font-medium text-ink">
                Эвристика отбора
              </label>
              <Hint label="Подсказка об эвристике">
                Жадный отбор соседей (эвристика из статьи HNSW) вместо простых k ближайших. Обычно
                улучшает recall при той же памяти.
              </Hint>
            </div>
            <p className="mt-0.5 text-[12px] leading-relaxed text-ink-3">
              Отбирать соседей по диверсификации направлений.
            </p>
          </div>
          <Switch id="build-heuristic" checked={heuristic} onCheckedChange={setHeuristic} />
        </div>

        <Field
          label="Seed"
          htmlFor="build-seed"
          error={errors.seed}
          hintAction={
            <Hint label="Подсказка о seed">
              Управляет случайными уровнями узлов. Одинаковый seed и параметры дают воспроизводимый
              граф.
            </Hint>
          }
        >
          <Input
            id="build-seed"
            type="number"
            inputMode="numeric"
            value={seed}
            onChange={(event) => setSeed(event.target.value)}
          />
        </Field>

        {error ? (
          <ErrorBanner title="Не удалось запустить сборку" message={errorMessage(error)} />
        ) : null}

        {datasets.length === 0 ? (
          <p className="text-[12px] text-ink-3">
            Сначала создайте датасет на странице «Датасеты» — без него индекс не построить.
          </p>
        ) : null}

        <div className="flex justify-end">
          <Button type="submit" size="sm" loading={building} disabled={datasets.length === 0}>
            <Play aria-hidden className="size-4" />
            {building ? 'Идёт сборка' : 'Построить индекс'}
          </Button>
        </div>
      </form>
    </Card>
  )
}
