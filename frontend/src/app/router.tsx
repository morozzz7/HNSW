import { createBrowserRouter, Navigate } from 'react-router-dom'
import { NotFoundPage } from './NotFoundPage'
import { AppShell } from './AppShell'
import { BenchmarkPage } from '@/features/benchmark/BenchmarkPage'
import { DatasetPage } from '@/features/datasets/DatasetPage'
import { IndexPage } from '@/features/index/IndexPage'
import { SearchPage } from '@/features/search/SearchPage'

export const router = createBrowserRouter([
  {
    path: '/',
    element: <AppShell />,
    children: [
      { index: true, element: <Navigate to="/datasets" replace /> },
      { path: 'datasets', element: <DatasetPage /> },
      { path: 'index', element: <IndexPage /> },
      { path: 'search', element: <SearchPage /> },
      { path: 'benchmark', element: <BenchmarkPage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
])
