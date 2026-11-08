import { Dices, Search } from 'lucide-react'
import { useState } from 'react'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Field, Input } from '@/components/ui/Field'
import { Select } from '@/components/ui/Select'
import { Switch } from '@/components/ui/Switch'
import type { Dataset, SearchInput, UISearchMode } from '@/api/schemas'
import { SEARCH_MODE_DESCRIPTIONS, SEARCH_MODE_LABELS } from './labels'

interface SearchFormProps {
  dataset?: Dataset | null
  searching: boolean
  onSubmit: (input: SearchInput, uiMode: UISearchMode) => void
  onPickRandomQuery?: () => void
}

type QueryInputType = 'vectorId' | 'manual'

export function SearchForm({ dataset, searching, onSubmit }: SearchFormProps) {
  const [inputType, setInputType] = useState<QueryInputType>('vectorId')
  const [vectorId, setVectorId] = useState<string>('0')
  const [manualText, setManualText] = useState<string>('')
  const [k, setK] = useState<number>(10)
  const [efSearch, setEfSearch] = useState<number>(50)
  const [uiMode, setUiMode] = useState<UISearchMode>('both')
  const [trace, setTrace] = useState<boolean>(true)
  const [error, setError] = useState<string | null>(null)

  const handleRandomFromDataset = () => {
    if (!dataset || dataset.n <= 0) return
    const randomRow = Math.floor(Math.random() * Math.min(dataset.n, 1000))
    setInputType('vectorId')
    setVectorId(String(randomRow))
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    if (inputType === 'vectorId') {
      const trimmed = vectorId.trim()
      if (!trimmed) {
        setError('Укажите ID вектора для поиска')
        return
      }
      onSubmit(
        {
          vectorId: trimmed,
          k,
          efSearch,
          mode: uiMode === 'both' ? 'hnsw' : uiMode,
          trace: trace && uiMode !== 'bruteforce',
          includeVectors: true,
        },
        uiMode,
      )
      return
    }

    // Manual floats
    const parts = manualText
      .split(/[\s,;]+/)
      .map((s) => s.trim())
      .filter(Boolean)
      .map(Number)

    if (parts.length === 0 || parts.some((n) => !Number.isFinite(n))) {
      setError('Введите корректные числовые компоненты вектора')
      return
    }

    if (dataset && dataset.dim > 0 && parts.length !== dataset.dim) {
      setError(`Размерность вектора (${parts.length}) не совпадает с датасетом (${dataset.dim})`)
      return
    }

    onSubmit(
      {
        vector: parts,
        k,
        efSearch,
        mode: uiMode === 'both' ? 'hnsw' : uiMode,
        trace: trace && uiMode !== 'bruteforce',
        includeVectors: true,
      },
      uiMode,
    )
  }

  return (
    <Card
      title="Параметры поиска"
      description="Запрос k ближайших соседей и сравнение HNSW с полным перебором."
    >
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        {/* Query Vector Choice */}
        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <span className="text-[13px] font-medium text-ink">Вектор запроса</span>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={handleRandomFromDataset}
                disabled={!dataset || dataset.n === 0}
                className="flex items-center gap-1 text-[11px] text-accent transition-colors hover:underline disabled:opacity-40"
              >
                <Dices className="size-3.5" />
                <span>Случайный из датасета</span>
              </button>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-1 rounded-lg border border-line bg-surface-2 p-0.5 text-[12px]">
            <button
              type="button"
              onClick={() => setInputType('vectorId')}
              className={`rounded-md py-1 font-medium transition-colors ${
                inputType === 'vectorId'
                  ? 'bg-surface text-ink shadow-xs'
                  : 'text-ink-3 hover:text-ink'
              }`}
            >
              По ID вектора
            </button>
            <button
              type="button"
              onClick={() => setInputType('manual')}
              className={`rounded-md py-1 font-medium transition-colors ${
                inputType === 'manual'
                  ? 'bg-surface text-ink shadow-xs'
                  : 'text-ink-3 hover:text-ink'
              }`}
            >
              Вручную (числа)
            </button>
          </div>

          {inputType === 'vectorId' ? (
            <Field
              label="ID вектора (строка или число)"
              htmlFor="query-vector-id"
              description={
                dataset ? `Датасет содержит ${dataset.n} векторов (0..${dataset.n - 1})` : undefined
              }
            >
              <Input
                id="query-vector-id"
                value={vectorId}
                onChange={(e) => setVectorId(e.target.value)}
                placeholder="0"
                className="font-mono text-xs"
              />
            </Field>
          ) : (
            <Field
              label="Компоненты вектора (через пробел или запятую)"
              htmlFor="query-vector-manual"
              description={dataset ? `Ожидается размерность ${dataset.dim}` : undefined}
            >
              <textarea
                id="query-vector-manual"
                rows={2}
                value={manualText}
                onChange={(e) => setManualText(e.target.value)}
                placeholder="0.12, -0.45, 0.88, ..."
                className="w-full rounded-lg border border-line bg-surface p-2 font-mono text-xs text-ink transition-colors placeholder:text-ink-3 hover:border-line-strong focus:border-accent"
              />
            </Field>
          )}
        </div>

        {/* Parameters: k, efSearch */}
        <div className="grid grid-cols-2 gap-3">
          <Field label="k (соседей)" htmlFor="search-k" description="от 1 до 1000">
            <Input
              id="search-k"
              type="number"
              min={1}
              max={1000}
              value={k}
              onChange={(e) => setK(Math.max(1, Math.min(1000, Number(e.target.value))))}
              className="font-mono"
            />
          </Field>

          <Field label="efSearch" htmlFor="search-ef" description="от 0 до 10000">
            <Input
              id="search-ef"
              type="number"
              min={0}
              max={10000}
              value={efSearch}
              onChange={(e) => setEfSearch(Math.max(0, Math.min(10000, Number(e.target.value))))}
              className="font-mono"
            />
          </Field>
        </div>

        {/* Mode Selector */}
        <Field
          label="Режим поиска"
          htmlFor="search-mode"
          description={SEARCH_MODE_DESCRIPTIONS[uiMode]}
        >
          <Select
            id="search-mode"
            value={uiMode}
            onChange={(e) => setUiMode(e.target.value as UISearchMode)}
          >
            {(Object.keys(SEARCH_MODE_LABELS) as UISearchMode[]).map((mode) => (
              <option key={mode} value={mode}>
                {SEARCH_MODE_LABELS[mode]}
              </option>
            ))}
          </Select>
        </Field>

        {/* Trace descent toggle */}
        {uiMode !== 'bruteforce' ? (
          <div className="flex items-center justify-between border-t border-line pt-3">
            <div>
              <p className="text-[13px] font-medium text-ink">Трассировка спуска</p>
              <p className="text-[11px] text-ink-3">Записать путь поиска по слоям для анимации</p>
            </div>
            <Switch checked={trace} onCheckedChange={setTrace} aria-label="Трассировка спуска" />
          </div>
        ) : null}

        {error ? (
          <p className="text-[12px] text-danger" role="alert">
            {error}
          </p>
        ) : null}

        <Button type="submit" disabled={searching} className="mt-1 w-full gap-2">
          <Search className="size-4" />
          <span>{searching ? 'Поиск...' : 'Выполнить поиск'}</span>
        </Button>
      </form>
    </Card>
  )
}
