import type { UISearchMode } from '@/api/schemas'

export const SEARCH_MODE_LABELS: Record<UISearchMode, string> = {
  hnsw: 'HNSW',
  bruteforce: 'Полный перебор',
  both: 'Оба (сравнение)',
}

export const SEARCH_MODE_DESCRIPTIONS: Record<UISearchMode, string> = {
  hnsw: 'Приближённый поиск по графу HNSW со спуском по слоям.',
  bruteforce: 'Точный линейный перебор всех векторов индекса.',
  both: 'Параллельный запуск HNSW и перебора для оценки Recall@k и выигрыша по времени.',
}

export type LayoutMode = 'pca' | 'force'

export const LAYOUT_LABELS: Record<LayoutMode, string> = {
  pca: 'PCA (2D)',
  force: 'Силовой (Force)',
}

export const LAYER_COLORS = [
  '#5be0b5', // L0 - mint
  '#38bdf8', // L1 - sky
  '#a78bfa', // L2 - violet
  '#fb923c', // L3 - orange
  '#f472b6', // L4 - pink
  '#facc15', // L5+ - yellow
]

export function getLayerColor(level: number): string {
  return LAYER_COLORS[level % LAYER_COLORS.length] ?? '#5be0b5'
}
