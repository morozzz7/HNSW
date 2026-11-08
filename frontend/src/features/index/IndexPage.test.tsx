import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { renderWithProviders } from '@/test/render'
import { setMockBuildDuration } from '@/mocks/state'
import { IndexPage } from './IndexPage'

describe('IndexPage', () => {
  it('shows the empty state before any build', async () => {
    renderWithProviders(<IndexPage />)

    expect(await screen.findByText('Индекс ещё не построен')).toBeInTheDocument()
    expect(await screen.findByRole('option', { name: /ds_000001/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Построить индекс/ })).toBeEnabled()
  })

  it('validates M against the backend limits', async () => {
    const user = userEvent.setup()
    renderWithProviders(<IndexPage />)

    const m = await screen.findByLabelText('M')
    await user.clear(m)
    await user.type(m, '1')
    await user.click(screen.getByRole('button', { name: /Построить индекс/ }))

    expect(await screen.findByText('M от 2 до 128')).toBeInTheDocument()
  })

  it('shows progress while the index is building', async () => {
    setMockBuildDuration(10_000)
    const user = userEvent.setup()
    renderWithProviders(<IndexPage />)

    await screen.findByLabelText('Датасет')
    await user.click(screen.getByRole('button', { name: /Построить индекс/ }))

    await waitFor(() => {
      expect(screen.getByText('Идёт сборка')).toBeInTheDocument()
      expect(
        screen.getByRole('progressbar', { name: 'Прогресс сборки индекса' }),
      ).toBeInTheDocument()
    })
  })

  it('renders graph statistics once the index is ready', async () => {
    setMockBuildDuration(0)
    const user = userEvent.setup()
    renderWithProviders(<IndexPage />)

    await screen.findByLabelText('Датасет')
    await user.click(screen.getByRole('button', { name: /Построить индекс/ }))

    await waitFor(() => {
      expect(screen.getByText('Индекс готов')).toBeInTheDocument()
    })
    expect(await screen.findByText('Узлы по уровням')).toBeInTheDocument()
    expect(screen.getByText('Гистограмма степеней')).toBeInTheDocument()
    expect(screen.getByText('L0')).toBeInTheDocument()
    expect(screen.getByRole('img', { name: /Гистограмма степеней/ })).toBeInTheDocument()
  })
})
