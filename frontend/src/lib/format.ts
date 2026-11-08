const intFormatter = new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 0 })
const dateFormatter = new Intl.DateTimeFormat('ru-RU', {
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
})

export function formatInt(value: number): string {
  return intFormatter.format(value)
}

export function formatDecimal(value: number, digits = 2): string {
  return new Intl.NumberFormat('ru-RU', {
    minimumFractionDigits: 0,
    maximumFractionDigits: digits,
  }).format(value)
}

export function formatPercent(value: number, digits = 0): string {
  return `${formatDecimal(value * 100, digits)}%`
}

export function formatMs(value: number): string {
  if (value < 1000) {
    return `${formatDecimal(value, 0)} мс`
  }
  return `${formatDecimal(value / 1000, 2)} с`
}

export function formatBytes(value: number): string {
  if (value < 1024) {
    return `${value} Б`
  }
  const units = ['КиБ', 'МиБ', 'ГиБ']
  let size = value / 1024
  let unit = 0
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024
    unit += 1
  }
  return `${formatDecimal(size, 1)} ${units[unit]}`
}

export function formatDate(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) {
    return iso
  }
  return dateFormatter.format(date)
}

export function formatVectorPreview(vector: readonly number[], limit = 4): string {
  const head = vector.slice(0, limit).map((value) => formatDecimal(value, 3))
  const tail = vector.length > limit ? ', …' : ''
  return `[${head.join(', ')}${tail}]`
}
