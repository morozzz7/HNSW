import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { LayerSelector } from './LayerSelector'
import { SearchPathOverlay } from './SearchPathOverlay'
import type { SearchTrace } from '@/api/schemas'

describe('LayerSelector', () => {
  it('renders all layers with counts and handles level selection', async () => {
    const onSelect = vi.fn()
    const onToggleMultilayer = vi.fn()
    const user = userEvent.setup()

    render(
      <LayerSelector
        maxLevel={3}
        activeLevel={1}
        nodesPerLevel={[1000, 100, 10, 1]}
        visitedLevels={new Set([2, 1])}
        isMultilayer={false}
        onSelectLevel={onSelect}
        onToggleMultilayer={onToggleMultilayer}
        totalEdges={42}
      />,
    )

    expect(screen.getByText('L0')).toBeInTheDocument()
    expect(screen.getByText('L1')).toBeInTheDocument()
    expect(screen.getByText('L2')).toBeInTheDocument()
    expect(screen.getByText('L3')).toBeInTheDocument()
    expect(screen.getByText('Рёбер на слое:')).toBeInTheDocument()

    const l2Btn = screen.getByRole('button', { name: /L2/ })
    await user.click(l2Btn)
    expect(onSelect).toHaveBeenCalledWith(2)

    const multiBtn = screen.getByRole('button', { name: /Многослойный вид/ })
    await user.click(multiBtn)
    expect(onToggleMultilayer).toHaveBeenCalled()
  })
})

describe('SearchPathOverlay', () => {
  const mockTrace: SearchTrace = {
    layers: [
      { level: 2, visited: [100, 105], closest: 105 },
      { level: 1, visited: [105, 42], closest: 42 },
      { level: 0, visited: [42, 10, 7], closest: 7 },
    ],
  }

  it('renders player controls and steps forward on button click', async () => {
    const onStepChange = vi.fn()
    const user = userEvent.setup()

    render(<SearchPathOverlay trace={mockTrace} onStepChange={onStepChange} />)

    expect(screen.getByTitle(/Воспроизведение/)).toBeInTheDocument()
    expect(screen.getByText(/Шаг/)).toBeInTheDocument()

    const nextBtn = screen.getByTitle(/Шаг вперёд/)
    await user.click(nextBtn)

    expect(onStepChange).toHaveBeenCalled()
  })
})
