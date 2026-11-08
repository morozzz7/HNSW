import { FileUp, Upload } from 'lucide-react'
import { useId, useRef, useState } from 'react'
import { useUploadDataset } from '@/api/hooks/useDatasets'
import { errorMessage } from '@/api/errors'
import { MAX_UPLOAD_BYTES } from '@/api/schemas'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { ErrorBanner } from '@/components/ui/ErrorBanner'
import { Progress } from '@/components/ui/Progress'
import { cn } from '@/lib/cn'
import { formatBytes, formatPercent } from '@/lib/format'

interface UploadCardProps {
  onCreated: (id: string) => void
}

const ACCEPTED = '.csv,.json'

function checkFile(file: File): string | null {
  if (!/\.(csv|json)$/i.test(file.name)) {
    return 'Допустимы только файлы .csv и .json'
  }
  if (file.size > MAX_UPLOAD_BYTES) {
    return `Файл больше ${formatBytes(MAX_UPLOAD_BYTES)}`
  }
  if (file.size === 0) {
    return 'Файл пустой'
  }
  return null
}

export function UploadCard({ onCreated }: UploadCardProps) {
  const upload = useUploadDataset()
  const inputRef = useRef<HTMLInputElement>(null)
  const inputId = useId()
  const [dragging, setDragging] = useState(false)
  const [progress, setProgress] = useState<number | null>(null)
  const [file, setFile] = useState<File | null>(null)
  const [localError, setLocalError] = useState<string | null>(null)

  const start = (next: File) => {
    const problem = checkFile(next)
    if (problem) {
      setLocalError(problem)
      setFile(null)
      return
    }
    setLocalError(null)
    setFile(next)
    setProgress(0)
    upload.mutate(
      { file: next, onProgress: setProgress },
      {
        onSuccess: (dataset) => {
          setProgress(null)
          setFile(null)
          onCreated(dataset.id)
        },
        onError: () => setProgress(null),
      },
    )
  }

  const busy = upload.isPending
  const error = localError ?? (upload.error ? errorMessage(upload.error) : null)

  return (
    <Card
      title="Загрузить файл"
      description="Матрица векторов: строка — вектор, столбцы — координаты."
    >
      <div className="flex flex-col gap-3">
        <input
          ref={inputRef}
          id={inputId}
          type="file"
          accept={ACCEPTED}
          className="sr-only"
          disabled={busy}
          onChange={(event) => {
            const next = event.target.files?.[0]
            event.target.value = ''
            if (next) {
              start(next)
            }
          }}
        />
        <label
          htmlFor={inputId}
          onDragOver={(event) => {
            event.preventDefault()
            if (!busy) {
              setDragging(true)
            }
          }}
          onDragLeave={() => setDragging(false)}
          onDrop={(event) => {
            event.preventDefault()
            setDragging(false)
            const next = event.dataTransfer.files?.[0]
            if (next && !busy) {
              start(next)
            }
          }}
          className={cn(
            'flex min-h-40 cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border border-dashed px-4 py-6 text-center transition-colors',
            dragging
              ? 'border-accent bg-accent-soft'
              : 'border-line-strong bg-surface-2 hover:border-accent hover:bg-accent-soft',
            busy && 'pointer-events-none opacity-60',
          )}
        >
          <span className="text-ink-2">
            <Upload aria-hidden className="size-5" />
          </span>
          <span className="text-[13px] font-medium text-ink">Перетащите файл сюда или нажмите</span>
          <span className="font-mono text-[11px] text-ink-3">
            .csv · .json · до {formatBytes(MAX_UPLOAD_BYTES)}
          </span>
        </label>

        {file ? (
          <div className="flex flex-col gap-2 rounded-lg border border-line bg-surface-2 px-3 py-2">
            <div className="flex items-center gap-2">
              <FileUp aria-hidden className="size-4 shrink-0 text-accent" />
              <span className="min-w-0 flex-1 truncate font-mono text-[12px] text-ink">
                {file.name}
              </span>
              <span className="shrink-0 font-mono text-[11px] text-ink-3">
                {formatBytes(file.size)}
              </span>
            </div>
            {progress !== null ? (
              <div className="flex items-center gap-2">
                <Progress value={progress} label="Загрузка файла" className="flex-1" />
                <span className="w-10 text-right font-mono text-[11px] tabular-nums text-ink-2">
                  {formatPercent(progress)}
                </span>
              </div>
            ) : null}
          </div>
        ) : null}

        {error ? <ErrorBanner title="Не удалось загрузить датасет" message={error} /> : null}

        <div className="flex justify-end">
          <Button
            type="button"
            variant="secondary"
            size="sm"
            loading={busy}
            onClick={() => inputRef.current?.click()}
          >
            Выбрать файл
          </Button>
        </div>
      </div>
    </Card>
  )
}
