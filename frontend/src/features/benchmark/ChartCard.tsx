import { Download } from 'lucide-react'
import { useRef, useState, type ReactNode } from 'react'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { exportChartPng } from './export'

interface ChartCardProps {
  title: string
  description?: ReactNode
  filename: string
  toolbar?: ReactNode
  children: ReactNode
}

export function ChartCard({ title, description, filename, toolbar, children }: ChartCardProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [exporting, setExporting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleExport = async () => {
    if (!containerRef.current) {
      return
    }
    setExporting(true)
    setError(null)
    try {
      await exportChartPng(containerRef.current, filename)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Не удалось сохранить PNG')
    } finally {
      setExporting(false)
    }
  }

  return (
    <Card
      title={title}
      description={description}
      actions={
        <>
          {toolbar}
          <Button
            type="button"
            variant="secondary"
            size="sm"
            onClick={() => void handleExport()}
            loading={exporting}
            disabled={exporting}
          >
            <Download aria-hidden className="size-3.5" />
            PNG
          </Button>
        </>
      }
    >
      <div ref={containerRef}>{children}</div>
      {error ? (
        <p role="alert" className="mt-2 text-[12px] text-danger">
          {error}
        </p>
      ) : null}
    </Card>
  )
}
