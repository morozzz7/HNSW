export const CHART_WIDTH = 720
export const CHART_HEIGHT = 300
export const CHART_MARGIN = { top: 16, right: 16, bottom: 40, left: 60 }

export const CHART_COLORS = [
  'var(--accent)',
  'var(--warn)',
  'var(--chart-3)',
  'var(--chart-4)',
  'var(--chart-5)',
  'var(--chart-6)',
]

export function chartColor(index: number): string {
  return CHART_COLORS[index % CHART_COLORS.length] ?? 'var(--accent)'
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}

export function linearScale(domain: [number, number], range: [number, number]) {
  const [d0, d1] = domain
  const [r0, r1] = range
  const span = d1 - d0
  return (value: number): number => {
    if (span === 0) {
      return (r0 + r1) / 2
    }
    return r0 + ((value - d0) / span) * (r1 - r0)
  }
}

export function logScale(domain: [number, number], range: [number, number]) {
  const lo = Math.max(domain[0], 1e-9)
  const hi = Math.max(domain[1], lo * 1.0001)
  const [r0, r1] = range
  return (value: number): number => {
    const clamped = clamp(value, lo, hi)
    const t = (Math.log10(clamped) - Math.log10(lo)) / (Math.log10(hi) - Math.log10(lo))
    return r0 + t * (r1 - r0)
  }
}

function roundTick(value: number): number {
  return Number(value.toPrecision(12))
}

export function niceTicks(min: number, max: number, count = 5): number[] {
  if (!Number.isFinite(min) || !Number.isFinite(max)) {
    return []
  }
  if (min === max) {
    return [roundTick(min)]
  }
  const span = max - min
  const rawStep = span / Math.max(1, count)
  const magnitude = 10 ** Math.floor(Math.log10(rawStep))
  const normalized = rawStep / magnitude
  const factor = normalized <= 1 ? 1 : normalized <= 2 ? 2 : normalized <= 5 ? 5 : 10
  const step = factor * magnitude
  const ticks: number[] = []
  for (let value = Math.ceil(min / step) * step; value <= max + step * 1e-6; value += step) {
    ticks.push(roundTick(value))
  }
  return ticks
}

export function logTicks(min: number, max: number): number[] {
  const lo = Math.max(min, 1e-9)
  const hi = Math.max(max, lo * 1.0001)
  const start = Math.floor(Math.log10(lo))
  const end = Math.ceil(Math.log10(hi))
  const ticks: number[] = []
  for (let exponent = start; exponent <= end; exponent += 1) {
    const value = 10 ** exponent
    if (value >= lo * 0.999 && value <= hi * 1.001) {
      ticks.push(value)
    }
  }
  return ticks.length >= 2 ? ticks : [lo, hi]
}

export function linePath(
  points: { x: number; y: number }[],
  xScale: (value: number) => number,
  yScale: (value: number) => number,
): string {
  return points
    .map((point, index) => {
      const x = xScale(point.x).toFixed(2)
      const y = yScale(point.y).toFixed(2)
      return `${index === 0 ? 'M' : 'L'}${x},${y}`
    })
    .join(' ')
}

export function paddedDomain(values: number[], padding = 0.06): [number, number] {
  if (values.length === 0) {
    return [0, 1]
  }
  const min = Math.min(...values)
  const max = Math.max(...values)
  if (min === max) {
    const delta = Math.abs(min) * padding || 0.5
    return [min - delta, max + delta]
  }
  const delta = (max - min) * padding
  return [min - delta, max + delta]
}
