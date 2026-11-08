export const queryKeys = {
  health: ['health'] as const,
  datasets: ['datasets'] as const,
  dataset: (id: string) => ['dataset', id] as const,
  indexStatus: ['index', 'status'] as const,
  indexStats: ['index', 'stats'] as const,
}
