import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '@/test/render'
import { api } from '@/api/endpoints'
import { server } from '@/mocks/server'
import { setMockBuildDuration } from '@/mocks/state'
import { SearchPage } from './SearchPage'

// Mock Sigma in tests to prevent jsdom WebGL errors
vi.mock('sigma', () => {
  return {
    default: class MockSigma {
      getCamera() {
        return {
          zoomIn: vi.fn(),
          zoomOut: vi.fn(),
          reset: vi.fn(),
        }
      }
      on() {}
      refresh() {}
      kill() {}
    },
  }
})

describe('SearchPage', () => {
  it('shows empty state when index is not yet built', async () => {
    renderWithProviders(<SearchPage />)
    expect(await screen.findByText('Индекс ещё не построен')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Перейти к построению индекса/ })).toBeInTheDocument()
  })

  it('renders search form and graph controls when index is ready', async () => {
    setMockBuildDuration(0)
    await api.buildIndex({
      datasetId: 'ds_000001',
      M: 8,
      efConstruction: 50,
      metric: 'l2',
      heuristic: true,
      seed: 1,
    })

    renderWithProviders(<SearchPage />)

    expect(await screen.findByText('Параметры поиска')).toBeInTheDocument()
    expect(screen.getByText('Вектор запроса')).toBeInTheDocument()
    expect(screen.getByLabelText(/k \(соседей\)/)).toBeInTheDocument()
    expect(screen.getByLabelText('efSearch')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Выполнить поиск/ })).toBeEnabled()
  })

  it('submits search and renders results table with comparison', async () => {
    setMockBuildDuration(0)
    await api.buildIndex({
      datasetId: 'ds_000001',
      M: 8,
      efConstruction: 50,
      metric: 'l2',
      heuristic: true,
      seed: 1,
    })

    const user = userEvent.setup()
    renderWithProviders(<SearchPage />)

    const submitBtn = await screen.findByRole('button', { name: /Выполнить поиск/ })
    await user.click(submitBtn)

    await waitFor(() => {
      expect(screen.getByText('Результаты k-NN')).toBeInTheDocument()
    })

    expect(screen.getByText(/Точность \(Recall\)/)).toBeInTheDocument()
    expect(screen.getByText(/Ускорение HNSW/)).toBeInTheDocument()
    expect(screen.getByText('Дистанция BF')).toBeInTheDocument()
  })

  it('loads trace vertices that fall outside the displayed graph limit', async () => {
    setMockBuildDuration(0)
    await api.buildIndex({
      datasetId: 'ds_000001',
      M: 8,
      efConstruction: 50,
      metric: 'l2',
      heuristic: true,
      seed: 1,
    })
    const graphSpy = vi.spyOn(api, 'graph')

    const user = userEvent.setup()
    renderWithProviders(<SearchPage />)
    await user.click(await screen.findByRole('button', { name: /Выполнить поиск/ }))

    await waitFor(() => {
      expect(graphSpy).toHaveBeenCalledWith(
        expect.objectContaining({ limit: 1, around: '666' }),
        expect.anything(),
      )
    })
  })

  it('shows a server error instead of silently dropping a failed search', async () => {
    setMockBuildDuration(0)
    await api.buildIndex({
      datasetId: 'ds_000001',
      M: 8,
      efConstruction: 50,
      metric: 'l2',
      heuristic: true,
      seed: 1,
    })
    server.use(
      http.post('*/api/v1/search', () =>
        HttpResponse.json(
          {
            error: {
              code: 'K_EXCEEDS_N',
              message: 'k=100 exceeds the number of indexed vectors 16',
            },
          },
          { status: 400 },
        ),
      ),
    )

    const user = userEvent.setup()
    renderWithProviders(<SearchPage />)

    await user.click(await screen.findByRole('button', { name: /Выполнить поиск/ }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'k=100 exceeds the number of indexed vectors 16',
    )
    expect(screen.queryByText('Результаты k-NN')).not.toBeInTheDocument()
  })
})
