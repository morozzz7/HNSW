import type { BenchmarkStatusName } from '@/api/schemas'

export const BENCHMARK_STATUS_LABELS: Record<BenchmarkStatusName, string> = {
  queued: 'В очереди',
  running: 'Выполняется',
  done: 'Готово',
  error: 'Ошибка',
}

export const BENCHMARK_STATUS_TONE = {
  queued: 'neutral',
  running: 'warn',
  done: 'accent',
  error: 'danger',
} as const
