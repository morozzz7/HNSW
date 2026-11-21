import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import {
  seedMockState,
  setMockBenchmarkDuration,
  setMockBuildDuration,
  startMockBuild,
} from '@/mocks/state'
import { renderWithProviders } from '@/test/render'
import { BenchmarkPage } from './BenchmarkPage'

function prepareReadyIndex() {
  const { uniform } = seedMockState()
  setMockBuildDuration(0)
  setMockBenchmarkDuration(0)
  startMockBuild(uniform, {
    M: 16,
    efConstruction: 200,
    metric: 'l2',
    heuristic: true,
    seed: 42,
  })
}

describe('BenchmarkPage', () => {
  it('asks for an index before any build', async () => {
    seedMockState()
    renderWithProviders(<BenchmarkPage />)

    expect(await screen.findByText('Индекс ещё не построен')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Перейти к построению индекса/ })).toHaveAttribute(
      'href',
      '/index',
    )
  })

  it('validates the grid against backend limits', async () => {
    prepareReadyIndex()
    const user = userEvent.setup()
    renderWithProviders(<BenchmarkPage />)

    const m = await screen.findByLabelText('M')
    await user.clear(m)
    await user.type(m, '1')
    await user.click(screen.getByRole('button', { name: /Запустить бенчмарк/ }))

    expect(await screen.findByText('M от 2 до 128')).toBeInTheDocument()
  })

  it('warns when the grid exceeds 256 combinations', async () => {
    prepareReadyIndex()
    const user = userEvent.setup()
    renderWithProviders(<BenchmarkPage />)

    await user.type(
      await screen.findByLabelText('M'),
      '2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20',
    )
    await user.type(screen.getByLabelText('efConstruction'), '100,200,300,400')
    await user.type(screen.getByLabelText('efSearch'), '10,20,30,40')

    expect(await screen.findByText(/Сетка больше 256 комбинаций/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Запустить бенчмарк/ })).toBeDisabled()
  })

  it('runs the benchmark and renders charts, summary and table', async () => {
    prepareReadyIndex()
    const user = userEvent.setup()
    renderWithProviders(<BenchmarkPage />)

    const runButton = await screen.findByRole('button', { name: /Запустить бенчмарк/ })
    await user.click(runButton)

    expect(await screen.findByText('Ход прогона', {}, { timeout: 5_000 })).toBeInTheDocument()

    await waitFor(
      () => {
        expect(screen.getByText('Результаты сетки')).toBeInTheDocument()
        expect(screen.getByText('Сравнение с полным перебором')).toBeInTheDocument()
      },
      { timeout: 5_000 },
    )

    expect(screen.getByRole('img', { name: /Recall@k/ })).toBeInTheDocument()
    expect(screen.getByRole('img', { name: /QPS в зависимости от recall/ })).toBeInTheDocument()
    expect(screen.getByRole('img', { name: /Задержка p50/ })).toBeInTheDocument()
    expect(screen.getByRole('img', { name: /Время построения индекса/ })).toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: 'PNG' })).toHaveLength(4)
    expect(screen.getByRole('button', { name: 'CSV' })).toBeInTheDocument()
  })
})
