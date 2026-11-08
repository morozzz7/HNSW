import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { AppShell } from './AppShell'
import { renderWithProviders } from '@/test/render'

describe('AppShell', () => {
  it('renders navigation links to every section', () => {
    renderWithProviders(<AppShell />, { route: '/datasets' })

    expect(screen.getByRole('link', { name: /HNSW Lab/ })).toHaveAttribute('href', '/datasets')
    expect(screen.getByRole('link', { name: /Датасеты/ })).toHaveAttribute('href', '/datasets')
    expect(screen.getByRole('link', { name: /Построение/ })).toHaveAttribute('href', '/index')
    expect(screen.getByRole('link', { name: /Поиск/ })).toHaveAttribute('href', '/search')
    expect(screen.getByRole('link', { name: /Бенчмарк/ })).toHaveAttribute('href', '/benchmark')
  })

  it('switches between dark and light themes', async () => {
    document.documentElement.classList.add('dark')
    const user = userEvent.setup()
    renderWithProviders(<AppShell />, { route: '/datasets' })

    const toggle = screen.getByRole('button', { name: /Светлая тема/ })
    await user.click(toggle)

    expect(document.documentElement).not.toHaveClass('dark')
    expect(localStorage.getItem('hnsw-theme')).toBe('light')
    expect(screen.getByRole('button', { name: /Тёмная тема/ })).toBeInTheDocument()
  })
})
