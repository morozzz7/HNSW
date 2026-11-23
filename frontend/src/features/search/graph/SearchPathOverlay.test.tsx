import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { SearchTrace } from '@/api/schemas'
import { SearchPathOverlay } from './SearchPathOverlay'

const trace: SearchTrace = {
  layers: [
    { level: 2, visited: [1, 2], closest: 2 },
    { level: 0, visited: [3], closest: 3 },
  ],
}

describe('SearchPathOverlay', () => {
  it('marks only the last step of the whole descent as final', () => {
    const onStepChange = vi.fn()
    render(<SearchPathOverlay trace={trace} onStepChange={onStepChange} />)
    fireEvent.click(screen.getByTitle('Шаг вперёд (→)'))
    fireEvent.click(screen.getByTitle('Шаг вперёд (→)'))

    const steps = onStepChange.mock.calls.map(([step]) => step)
    expect(steps).toHaveLength(3)
    expect(steps[0]?.isLastStep).toBe(false)
    expect(steps.at(-1)?.isLastStep).toBe(true)
    expect(steps.at(-1)?.globalIndex).toBe(2)
  })
})
