import { Link } from 'react-router-dom'
import { PageHeader } from '@/app/PageHeader'
import { Card } from '@/components/ui/Card'

export function NotFoundPage() {
  return (
    <>
      <PageHeader title="Страница не найдена" description="Такого раздела в HNSW Lab нет." />
      <Card>
        <Link to="/datasets" className="text-sm font-medium text-accent hover:underline">
          Перейти к датасетам
        </Link>
      </Card>
    </>
  )
}
