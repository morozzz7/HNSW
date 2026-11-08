import type { Metric } from '@/api/schemas'

export const METRIC_LABELS: Record<Metric, string> = {
  l2: 'L2, евклидово',
  cosine: 'Косинус',
  inner_product: 'Скалярное произведение',
}

export const STATUS_LABELS: Record<'idle' | 'building' | 'ready' | 'error', string> = {
  idle: 'Не построен',
  building: 'Сборка',
  ready: 'Готов',
  error: 'Ошибка',
}
