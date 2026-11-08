import type { Distribution } from '@/api/schemas'

export const DISTRIBUTION_LABELS: Record<Distribution, string> = {
  uniform: 'Равномерное',
  gaussian: 'Нормальное',
  clusters: 'Кластеры',
}

export function sourceLabel(source: string | undefined): string {
  switch (source) {
    case 'csv':
      return 'CSV'
    case 'json':
      return 'JSON'
    case 'generate':
      return 'Генерация'
    default:
      return 'Неизвестно'
  }
}
