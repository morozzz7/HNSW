import { Sparkles } from 'lucide-react'
import { useState } from 'react'
import { errorMessage } from '@/api/errors'
import { useGenerateDataset } from '@/api/hooks/useDatasets'
import { DISTRIBUTION_LABELS } from './labels'
import { MAX_VALUES, generateInputSchema, type Distribution } from '@/api/schemas'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { ErrorBanner } from '@/components/ui/ErrorBanner'
import { Field, Input } from '@/components/ui/Field'
import { Hint } from '@/components/ui/Hint'
import { Progress } from '@/components/ui/Progress'
import { Select } from '@/components/ui/Select'
import { formatInt, formatPercent } from '@/lib/format'

interface GenerateCardProps {
  onCreated: (id: string) => void
}

type FieldErrors = Partial<Record<'n' | 'dim' | 'seed', string>>

export function GenerateCard({ onCreated }: GenerateCardProps) {
  const generate = useGenerateDataset()
  const [n, setN] = useState('10000')
  const [dim, setDim] = useState('16')
  const [distribution, setDistribution] = useState<Distribution>('uniform')
  const [seed, setSeed] = useState('42')
  const [errors, setErrors] = useState<FieldErrors>({})

  const nValue = Number(n)
  const dimValue = Number(dim)
  const components =
    Number.isFinite(nValue) && Number.isFinite(dimValue) && nValue > 0 && dimValue > 0
      ? nValue * dimValue
      : 0
  const overBudget = components > MAX_VALUES
  const budget = Math.min(1, components / MAX_VALUES)

  const submit = () => {
    const parsed = generateInputSchema.safeParse({
      n: nValue,
      dim: dimValue,
      distribution,
      seed: Number(seed),
    })
    if (!parsed.success) {
      const next: FieldErrors = {}
      for (const issue of parsed.error.issues) {
        const key = String(issue.path[0] ?? '')
        if ((key === 'n' || key === 'dim' || key === 'seed') && !next[key]) {
          next[key] = issue.message
        }
      }
      setErrors(next)
      return
    }
    setErrors({})
    generate.mutate(parsed.data, {
      onSuccess: (dataset) => onCreated(dataset.id),
    })
  }

  return (
    <Card
      title="Сгенерировать набор"
      description="Синтетические векторы с фиксированным seed — удобно для повторяемых замеров."
    >
      <form
        className="flex flex-col gap-4"
        noValidate
        onSubmit={(event) => {
          event.preventDefault()
          submit()
        }}
      >
        <div className="grid gap-3 sm:grid-cols-2">
          <Field
            label="Векторов, n"
            htmlFor="generate-n"
            error={errors.n}
            hintAction={
              <Hint label="Подсказка о числе векторов">
                От 1 до 1 000 000. Вместе с размерностью ограничено бюджетом в 50 млн компонентов.
              </Hint>
            }
          >
            <Input
              id="generate-n"
              type="number"
              inputMode="numeric"
              min={1}
              max={1_000_000}
              value={n}
              onChange={(event) => setN(event.target.value)}
            />
          </Field>
          <Field
            label="Размерность, dim"
            htmlFor="generate-dim"
            error={errors.dim}
            hintAction={
              <Hint label="Подсказка о размерности">
                Число координат в каждом векторе, от 1 до 8192.
              </Hint>
            }
          >
            <Input
              id="generate-dim"
              type="number"
              inputMode="numeric"
              min={1}
              max={8192}
              value={dim}
              onChange={(event) => setDim(event.target.value)}
            />
          </Field>
        </div>

        <div className="grid gap-3 sm:grid-cols-2">
          <Field label="Распределение" htmlFor="generate-distribution">
            <Select
              id="generate-distribution"
              value={distribution}
              onChange={(event) => setDistribution(event.target.value as Distribution)}
            >
              {(Object.keys(DISTRIBUTION_LABELS) as Distribution[]).map((value) => (
                <option key={value} value={value}>
                  {DISTRIBUTION_LABELS[value]}
                </option>
              ))}
            </Select>
          </Field>
          <Field
            label="Seed"
            htmlFor="generate-seed"
            error={errors.seed}
            hintAction={
              <Hint label="Подсказка о seed">
                Одинаковые параметры и seed дают одинаковый набор векторов.
              </Hint>
            }
          >
            <Input
              id="generate-seed"
              type="number"
              inputMode="numeric"
              value={seed}
              onChange={(event) => setSeed(event.target.value)}
            />
          </Field>
        </div>

        <div className="flex flex-col gap-1.5">
          <div className="flex items-baseline justify-between gap-3">
            <span className="text-[12px] text-ink-3">Бюджет компонентов</span>
            <span
              className={
                overBudget
                  ? 'font-mono text-[12px] tabular-nums text-danger'
                  : 'font-mono text-[12px] tabular-nums text-ink-2'
              }
            >
              {formatInt(components)} / {formatInt(MAX_VALUES)}
            </span>
          </div>
          <Progress value={budget} label="Бюджет компонентов датасета" />
          <span className="text-[12px] text-ink-3">
            n × dim = {formatInt(components)} компонентов, использовано{' '}
            {formatPercent(components / MAX_VALUES, 1)}.
          </span>
        </div>

        {generate.error ? (
          <ErrorBanner
            title="Не удалось сгенерировать датасет"
            message={errorMessage(generate.error)}
          />
        ) : null}

        <div className="flex justify-end">
          <Button type="submit" size="sm" loading={generate.isPending} disabled={overBudget}>
            <Sparkles aria-hidden className="size-4" />
            Сгенерировать
          </Button>
        </div>
      </form>
    </Card>
  )
}
