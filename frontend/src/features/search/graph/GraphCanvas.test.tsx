import { render } from '@testing-library/react'
import type Graph from 'graphology'
import type { ReactElement } from 'react'
import { describe, expect, it, vi } from 'vitest'
import type { GraphEdge, GraphVertex } from '@/api/schemas'
import { ThemeProvider } from '@/app/ThemeProvider'
import { GraphCanvas } from './GraphCanvas'

const { sigmaConstructor, sigmaCalls } = vi.hoisted(() => ({
  sigmaConstructor: vi.fn(),
  sigmaCalls: [] as unknown[][],
}))

vi.mock('sigma', () => ({
  default: class MockSigma {
    constructor(...args: unknown[]) {
      sigmaConstructor()
      sigmaCalls.push(args)
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

  it('shows the node index only for the selected node', () => {
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

    type NodeReducer = (key: string, data: Record<string, unknown>) => Record<string, unknown>
    const latest = () => {
      const [, , options] = sigmaCalls.at(-1) as unknown as [
        Graph,
        HTMLElement,
        { nodeReducer: NodeReducer },
      ]
      return options
    }

    expect(latest().nodeReducer('2', { label: '', size: 4 })).toMatchObject({ label: '' })

    rerender(wrap(<GraphCanvas {...props} selectedNodeId={2} />))
    expect(latest().nodeReducer('2', { label: '', size: 4 })).toMatchObject({ label: '#2' })
    expect(latest().nodeReducer('1', { label: '', size: 4 })).toMatchObject({ label: '' })
  })

  it('paints final neighbors over the route color', () => {
    render(
      <ThemeProvider>
        <GraphCanvas
          vertices={vertices}
          edges={edges}
          activeLevel={0}
          selectedNodeId={null}
          layoutMode="pca"
          isMultilayer={false}
          visitedNodeIds={new Set([1, 2])}
          currentDescentNodeId={1}
          tracePathIds={[1, 2]}
          resultRanks={new Map([[2, 1]])}
          onSelectNode={() => {}}
          onHoverNode={() => {}}
        />
      </ThemeProvider>,
    )

    type NodeReducer = (key: string, data: Record<string, unknown>) => Record<string, unknown>
    const [graph, , options] = sigmaCalls.at(-1) as unknown as [
      Graph,
      HTMLElement,
      { nodeReducer: NodeReducer },
    ]

    expect(options.nodeReducer('1', graph.getNodeAttributes('1'))).toMatchObject({
      color: '#22d3ee',
    })
    expect(options.nodeReducer('2', graph.getNodeAttributes('2'))).toMatchObject({
      color: '#ec4899',
    })
  })

  it('leaves edges between highlighted nodes in their base color', () => {
    render(
      <ThemeProvider>
        <GraphCanvas
          vertices={vertices}
          edges={edges}
          activeLevel={0}
          selectedNodeId={null}
          layoutMode="pca"
          isMultilayer={false}
          resultRanks={
            new Map([
              [1, 1],
              [2, 2],
            ])
          }
          onSelectNode={() => {}}
          onHoverNode={() => {}}
        />
      </ThemeProvider>,
    )

    type EdgeReducer = (key: string, data: Record<string, unknown>) => Record<string, unknown>
    const [graph, , options] = sigmaCalls.at(-1) as unknown as [
      Graph,
      HTMLElement,
      { edgeReducer: EdgeReducer },
    ]
    const edge = graph.edges()[0]!

    expect(options.edgeReducer(edge, graph.getEdgeAttributes(edge))).toMatchObject({
      color: 'rgba(125, 143, 135, 0.25)',
    })
  })
})
