import { render } from '@testing-library/react'
import type { ReactElement } from 'react'
import { describe, expect, it, vi } from 'vitest'
import type { GraphEdge, GraphVertex } from '@/api/schemas'
import { ThemeProvider } from '@/app/ThemeProvider'
import { GraphCanvas } from './GraphCanvas'

const { sigmaConstructor } = vi.hoisted(() => ({ sigmaConstructor: vi.fn() }))

vi.mock('sigma', () => ({
  default: class MockSigma {
    constructor() {
      sigmaConstructor()
    }
    getCamera() {
      return { zoomIn: vi.fn(), zoomOut: vi.fn(), reset: vi.fn() }
    }
    on() {}
    refresh() {}
    kill() {}
  },
}))

const vertices: GraphVertex[] = [
  { id: 1, vectorId: '1', level: 0, x: 0, y: 0, isEntryPoint: true },
  { id: 2, vectorId: '2', level: 0, x: 1, y: 1, isEntryPoint: false },
]
const edges: GraphEdge[] = [{ from: 1, to: 2, level: 0 }]

describe('GraphCanvas', () => {
  it('keeps the sigma instance when only handlers change', () => {
    const props = {
      vertices,
      edges,
      activeLevel: 0,
      selectedNodeId: null,
      layoutMode: 'pca' as const,
      isMultilayer: false,
      onSelectNode: () => {},
      onHoverNode: () => {},
    }

    const wrap = (ui: ReactElement) => <ThemeProvider>{ui}</ThemeProvider>
    const { rerender } = render(wrap(<GraphCanvas {...props} />))
    expect(sigmaConstructor).toHaveBeenCalledTimes(1)

    rerender(
      wrap(
        <GraphCanvas
          {...props}
          onSelectNode={() => {}}
          onHoverNode={() => {}}
          selectedNodeId={2}
        />,
      ),
    )
    expect(sigmaConstructor).toHaveBeenCalledTimes(1)
  })
})
