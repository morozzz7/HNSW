import type { BenchmarkResult } from '@/api/schemas'

const CSV_COLUMNS = [
  'M',
  'efConstruction',
  'efSearch',
  'metric',
  'heuristic',
  'k',
  'recall',
  'qps',
  'latencyP50Ms',
  'latencyP95Ms',
  'latencyP99Ms',
  'buildTimeMs',
  'bruteforceQps',
  'speedup',
] as const

function escapeCsv(value: string | number | boolean): string {
  const text = String(value)
  if (text.includes(',') || text.includes('"') || text.includes('\n')) {
    return `"${text.replaceAll('"', '""')}"`
  }
  return text
}

export function benchmarkResultsToCsv(results: BenchmarkResult[]): string {
  const header = CSV_COLUMNS.join(',')
  const rows = results.map((row) =>
    [
      row.M,
      row.efConstruction,
      row.efSearch,
      row.metric,
      row.heuristic,
      row.k,
      row.recall,
      row.qps,
      row.latencyP50Ms,
      row.latencyP95Ms,
      row.latencyP99Ms,
      row.buildTimeMs,
      row.bruteforceQps,
      row.speedup,
    ]
      .map(escapeCsv)
      .join(','),
  )
  return [header, ...rows].join('\n')
}

export function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.append(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}

export function downloadCsv(filename: string, csv: string) {
  downloadBlob(new Blob([csv], { type: 'text/csv;charset=utf-8' }), filename)
}

const COPIED_STYLE_PROPERTIES = [
  'fill',
  'stroke',
  'stroke-width',
  'stroke-dasharray',
  'stroke-linecap',
  'stroke-linejoin',
  'fill-opacity',
  'opacity',
  'font-family',
  'font-size',
  'font-weight',
  'text-anchor',
  'dominant-baseline',
]

function loadImage(url: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image()
    image.onload = () => resolve(image)
    image.onerror = () => reject(new Error('Не удалось отрисовать SVG'))
    image.src = url
  })
}

export async function exportSvgToPng(
  svg: SVGSVGElement,
  filename: string,
  scale = 2,
): Promise<void> {
  const width = svg.viewBox.baseVal.width || svg.clientWidth || 720
  const height = svg.viewBox.baseVal.height || svg.clientHeight || 300

  const clone = svg.cloneNode(true) as SVGSVGElement
  const sourceElements = [svg, ...Array.from(svg.querySelectorAll('*'))]
  const cloneElements = [clone, ...Array.from(clone.querySelectorAll('*'))]
  sourceElements.forEach((source, index) => {
    const target = cloneElements[index] as HTMLElement | SVGElement | undefined
    if (!target) {
      return
    }
    const computed = window.getComputedStyle(source)
    for (const property of COPIED_STYLE_PROPERTIES) {
      const value = computed.getPropertyValue(property)
      if (value) {
        target.style.setProperty(property, value)
      }
    }
  })

  clone.setAttribute('width', String(width))
  clone.setAttribute('height', String(height))
  clone.setAttribute('xmlns', 'http://www.w3.org/2000/svg')

  const serialized = new XMLSerializer().serializeToString(clone)
  const url = URL.createObjectURL(new Blob([serialized], { type: 'image/svg+xml;charset=utf-8' }))

  try {
    const image = await loadImage(url)
    const canvas = document.createElement('canvas')
    canvas.width = Math.round(width * scale)
    canvas.height = Math.round(height * scale)
    const context = canvas.getContext('2d')
    if (!context) {
      throw new Error('Canvas недоступен')
    }
    const background =
      window.getComputedStyle(document.documentElement).getPropertyValue('--surface').trim() ||
      '#ffffff'
    context.fillStyle = background
    context.fillRect(0, 0, canvas.width, canvas.height)
    context.drawImage(image, 0, 0, canvas.width, canvas.height)
    const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/png'))
    if (!blob) {
      throw new Error('Не удалось сохранить PNG')
    }
    downloadBlob(blob, filename)
  } finally {
    URL.revokeObjectURL(url)
  }
}

export async function exportChartPng(container: HTMLElement, filename: string): Promise<void> {
  const svg = container.querySelector('svg')
  if (!svg) {
    throw new Error('На графике нет SVG')
  }
  await exportSvgToPng(svg, filename)
}
