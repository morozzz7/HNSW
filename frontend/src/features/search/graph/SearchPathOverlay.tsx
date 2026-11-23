import { ChevronLeft, ChevronRight, Pause, Play, RotateCcw } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { cn } from '@/lib/cn'
import type { SearchTrace } from '@/api/schemas'

export interface TraceStep {
  globalIndex: number
  level: number
  nodeId: number
  closest: number
  visitedOnLevel: number[]
  isLayerLastStep: boolean
  isLastStep: boolean
}

interface SearchPathOverlayProps {
  trace: SearchTrace
  onStepChange: (step: TraceStep) => void
  onClose?: () => void
}

const SPEED_OPTIONS = [
  { label: '0.5x', delayMs: 800 },
  { label: '1x', delayMs: 400 },
  { label: '2x', delayMs: 200 },
]

export function SearchPathOverlay({ trace, onStepChange, onClose }: SearchPathOverlayProps) {
  const steps = useMemo<TraceStep[]>(() => {
    const list: TraceStep[] = []
    let globalIndex = 0

    for (const layer of trace.layers) {
      const visitedOnLevelSoFar: number[] = []
      for (let i = 0; i < layer.visited.length; i += 1) {
        const nodeId = layer.visited[i]!
        visitedOnLevelSoFar.push(nodeId)
        list.push({
          globalIndex,
          level: layer.level,
          nodeId,
          closest: layer.closest,
          visitedOnLevel: [...visitedOnLevelSoFar],
          isLayerLastStep: i === layer.visited.length - 1,
          isLastStep: false,
        })
        globalIndex += 1
      }
    }

    if (list.length > 0) {
      list[list.length - 1]!.isLastStep = true
    }
    return list
  }, [trace])

  const [currentIndex, setCurrentIndex] = useState(0)
  const [isPlaying, setIsPlaying] = useState(false)
  const [speedIdx, setSpeedIdx] = useState(1) // 1x

  const currentStep = steps[currentIndex]

  // Broadcast step change
  useEffect(() => {
    if (currentStep) {
      onStepChange(currentStep)
    }
  }, [currentStep, onStepChange])

  // Playback timer
  useEffect(() => {
    if (!isPlaying) return
    const delay = SPEED_OPTIONS[speedIdx]?.delayMs ?? 400
    const timer = setInterval(() => {
      setCurrentIndex((prev) => {
        if (prev >= steps.length - 1) {
          setIsPlaying(false)
          return prev
        }
        return prev + 1
      })
    }, delay)

    return () => clearInterval(timer)
  }, [isPlaying, speedIdx, steps.length])

  // Keyboard navigation
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
        return
      }
      if (e.code === 'Space') {
        e.preventDefault()
        setIsPlaying((p) => !p)
      } else if (e.code === 'ArrowRight') {
        e.preventDefault()
        setIsPlaying(false)
        setCurrentIndex((p) => Math.min(steps.length - 1, p + 1))
      } else if (e.code === 'ArrowLeft') {
        e.preventDefault()
        setIsPlaying(false)
        setCurrentIndex((p) => Math.max(0, p - 1))
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [steps.length])

  if (steps.length === 0) {
    return null
  }

  return (
    <div
      role="region"
      aria-label="Управление анимацией спуска"
      className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-line bg-surface/95 px-4 py-2.5 shadow-lg backdrop-blur-md"
    >
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={() => {
            setIsPlaying(false)
            setCurrentIndex(0)
          }}
          className="rounded-md p-1.5 text-ink-3 transition-colors hover:bg-surface-2 hover:text-ink active:scale-95"
          title="В начало (Сброс)"
        >
          <RotateCcw className="size-4" />
        </button>

        <button
          type="button"
          disabled={currentIndex <= 0}
          onClick={() => {
            setIsPlaying(false)
            setCurrentIndex((p) => Math.max(0, p - 1))
          }}
          className="rounded-md p-1.5 text-ink-2 transition-colors hover:bg-surface-2 hover:text-ink disabled:opacity-40 active:scale-95"
          title="Шаг назад (←)"
        >
          <ChevronLeft className="size-4" />
        </button>

        <button
          type="button"
          onClick={() => {
            if (currentIndex >= steps.length - 1) {
              setCurrentIndex(0)
            }
            setIsPlaying((p) => !p)
          }}
          className="flex size-8 items-center justify-center rounded-lg bg-accent text-accent-ink transition-all hover:opacity-90 active:scale-95"
          title={isPlaying ? 'Пауза (Пробел)' : 'Воспроизведение (Пробел)'}
        >
          {isPlaying ? (
            <Pause className="size-4 fill-current" />
          ) : (
            <Play className="ml-0.5 size-4 fill-current" />
          )}
        </button>

        <button
          type="button"
          disabled={currentIndex >= steps.length - 1}
          onClick={() => {
            setIsPlaying(false)
            setCurrentIndex((p) => Math.min(steps.length - 1, p + 1))
          }}
          className="rounded-md p-1.5 text-ink-2 transition-colors hover:bg-surface-2 hover:text-ink disabled:opacity-40 active:scale-95"
          title="Шаг вперёд (→)"
        >
          <ChevronRight className="size-4" />
        </button>
      </div>

      <div className="flex items-center gap-3 font-mono text-[11px]">
        <div className="flex items-center gap-1.5">
          <span className="rounded bg-surface-2 px-1.5 py-0.5 font-medium text-accent">
            L{currentStep?.level}
          </span>
          <span className="text-ink-3">Узел:</span>
          <span className="font-semibold text-ink">#{currentStep?.nodeId}</span>
        </div>

        <div className="text-ink-3">
          Шаг <span className="tabular-nums text-ink">{currentIndex + 1}</span> / {steps.length}
        </div>
      </div>

      <div className="flex items-center gap-1.5">
        <span className="text-[11px] text-ink-3">Скорость:</span>
        <div className="inline-flex rounded-md border border-line p-0.5">
          {SPEED_OPTIONS.map((opt, idx) => (
            <button
              key={opt.label}
              type="button"
              onClick={() => setSpeedIdx(idx)}
              className={cn(
                'rounded px-1.5 py-0.5 font-mono text-[10px] transition-colors',
                speedIdx === idx
                  ? 'bg-surface-2 font-medium text-ink'
                  : 'text-ink-3 hover:text-ink',
              )}
            >
              {opt.label}
            </button>
          ))}
        </div>

        {onClose ? (
          <button
            type="button"
            onClick={onClose}
            className="ml-2 rounded px-2 py-1 text-[11px] text-ink-3 hover:text-ink"
          >
            Скрыть
          </button>
        ) : null}
      </div>
    </div>
  )
}
