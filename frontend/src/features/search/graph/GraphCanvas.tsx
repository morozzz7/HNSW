import Graph from 'graphology'
import forceAtlas2 from 'graphology-layout-forceatlas2'
import { Focus, Maximize2, Minimize2, ZoomIn, ZoomOut } from 'lucide-react'
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import Sigma from 'sigma'
import type { GraphEdge, GraphVertex } from '@/api/schemas'
import { useTheme, type Theme } from '@/app/theme'
import { getLayerColor, type LayoutMode } from '../labels'

interface GraphCanvasProps {
  vertices: GraphVertex[]
  edges: GraphEdge[]
  activeLevel: number
  selectedNodeId: number | null
  hoveredNodeId?: number | null
  visitedNodeIds?: Set<number>
  currentDescentNodeId?: number | null
  tracePathIds?: number[]
  resultRanks?: Map<number, number>
  neighborhoodIds?: Set<number> | null
  layoutMode: LayoutMode
  isMultilayer: boolean
  allLayerVertices?: Map<number, GraphVertex[]>
  onSelectNode: (id: number | null) => void
  onHoverNode?: (id: number | null) => void
  className?: string
}

const ROUTE_COLOR = '#8b5cf6'
const CURRENT_STEP_COLOR = '#22d3ee'
const RESULT_COLOR = '#ec4899'

function mutedLayerColor(color: string, theme: Theme, weight = 0.4): string {
  const hex = color.replace('#', '')
  if (hex.length !== 6) return color
  const r = parseInt(hex.slice(0, 2), 16)
  const g = parseInt(hex.slice(2, 4), 16)
  const b = parseInt(hex.slice(4, 6), 16)
  const target = theme === 'dark' ? [11, 15, 14] : [16, 25, 21]
  const mix = (value: number, index: number) =>
    Math.round(value + (target[index]! - value) * weight)
  return `rgb(${mix(r, 0)}, ${mix(g, 1)}, ${mix(b, 2)})`
}

function normalizeCoordinates(verts: GraphVertex[]): Map<number, { x: number; y: number }> {
  const result = new Map<number, { x: number; y: number }>()
  if (verts.length === 0) return result
  if (verts.length === 1) {
    result.set(verts[0]!.id, { x: 0, y: 0 })
    return result
  }

  let minX = Infinity
  let maxX = -Infinity
  let minY = Infinity
  let maxY = -Infinity
  for (const v of verts) {
    if (v.x < minX) minX = v.x
    if (v.x > maxX) maxX = v.x
    if (v.y < minY) minY = v.y
    if (v.y > maxY) maxY = v.y
  }

  const spanX = maxX - minX || 1
  const spanY = maxY - minY || 1
  const maxSpan = Math.max(spanX, spanY)
  const midX = (minX + maxX) / 2
  const midY = (minY + maxY) / 2
  const scale = 200 / maxSpan

  for (const v of verts) {
    result.set(v.id, {
      x: (v.x - midX) * scale,
      y: (v.y - midY) * scale,
    })
  }
  return result
}

export function GraphCanvas({
  vertices,
  edges,
  activeLevel,
  selectedNodeId,
  hoveredNodeId,
  visitedNodeIds,
  currentDescentNodeId,
  tracePathIds,
  resultRanks,
  neighborhoodIds,
  layoutMode,
  isMultilayer,
  allLayerVertices,
  onSelectNode,
  onHoverNode,
  className,
}: GraphCanvasProps) {
  const { theme } = useTheme()
  const labelColor = theme === 'dark' ? '#e8f0ec' : '#3a4a43'
  const accentColor = theme === 'dark' ? '#5be0b5' : '#0b7a5f'
  const rootRef = useRef<HTMLDivElement | null>(null)
  const containerRef = useRef<HTMLDivElement | null>(null)
  const sigmaRef = useRef<Sigma | null>(null)
  const graphRef = useRef<Graph | null>(null)
  const [isFullscreen, setIsFullscreen] = useState<boolean>(false)

  const tracePathEdgeKeys = useMemo(() => {
    const pairs = new Set<string>()
    if (!tracePathIds) return pairs
    for (let i = 1; i < tracePathIds.length; i += 1) {
      const a = tracePathIds[i - 1]!
      const b = tracePathIds[i]!
      pairs.add(`${Math.min(a, b)}:${Math.max(a, b)}`)
    }
    return pairs
  }, [tracePathIds])

  const visualStateRef = useRef({
    selectedNodeId,
    hoveredNodeId,
    visitedNodeIds,
    currentDescentNodeId,
    tracePathIds,
    tracePathEdgeKeys,
    resultRanks,
    neighborhoodIds,
  })

  const callbacksRef = useRef({ onSelectNode, onHoverNode })

  useEffect(() => {
    callbacksRef.current = { onSelectNode, onHoverNode }
  }, [onSelectNode, onHoverNode])

  // Track fullscreen changes and resize camera
  useEffect(() => {
    const handleFullscreenChange = () => {
      setIsFullscreen(document.fullscreenElement === rootRef.current)
      if (sigmaRef.current) {
        sigmaRef.current.refresh()
      }
    }
    document.addEventListener('fullscreenchange', handleFullscreenChange)
    return () => {
      document.removeEventListener('fullscreenchange', handleFullscreenChange)
    }
  }, [])

  // Build and update graph structure
  useEffect(() => {
    if (!containerRef.current) return

    const graph = new Graph({ multi: false, type: 'undirected' })
    graphRef.current = graph

    if (isMultilayer && allLayerVertices && allLayerVertices.size > 0) {
      // Multilayer 2.5D isometric view
      const sortedLevels = Array.from(allLayerVertices.keys()).sort((a, b) => a - b)
      const nodesByOriginalId = new Map<number, { level: number; key: string }[]>()

      const allVertsList: GraphVertex[] = []
      for (const layerVerts of allLayerVertices.values()) {
        allVertsList.push(...layerVerts)
      }
      const coords = normalizeCoordinates(allVertsList)

      for (const level of sortedLevels) {
        const layerVerts = allLayerVertices.get(level) ?? []
        for (const v of layerVerts) {
          const key = `L${level}_${v.id}`
          const pos = coords.get(v.id) ?? { x: 0, y: 0 }
          const xIso = pos.x + level * 28
          const yIso = pos.y - level * 42
          const color = v.isEntryPoint ? '#e0b15b' : getLayerColor(level)

          graph.addNode(key, {
            id: v.id,
            vectorId: v.vectorId,
            level,
            x: xIso,
            y: yIso,
            size: v.isEntryPoint ? 6 : Math.max(3, Math.min(8, 2.5 + level * 1.0)),
            label: v.isEntryPoint ? '★' : '',
            labelColor,
            color,
            isEntryPoint: v.isEntryPoint,
          })

          const existing = nodesByOriginalId.get(v.id) ?? []
          existing.push({ level, key })
          nodesByOriginalId.set(v.id, existing)
        }
      }

      // Add in-layer edges
      for (const e of edges) {
        const uKey = `L${e.level}_${e.from}`
        const vKey = `L${e.level}_${e.to}`
        if (graph.hasNode(uKey) && graph.hasNode(vKey) && !graph.hasEdge(uKey, vKey)) {
          graph.addEdge(uKey, vKey, {
            size: 1,
            color: 'rgba(125, 143, 135, 0.25)',
            level: e.level,
          })
        }
      }

      // Add inter-layer vertical edges connecting same vertex ID across levels
      for (const [, entries] of nodesByOriginalId.entries()) {
        if (entries.length > 1) {
          entries.sort((a, b) => a.level - b.level)
          for (let i = 0; i < entries.length - 1; i += 1) {
            const current = entries[i]!
            const next = entries[i + 1]!
            if (!graph.hasEdge(current.key, next.key)) {
              graph.addEdge(current.key, next.key, {
                size: 1.5,
                color: 'rgba(91, 224, 181, 0.45)',
                isInterLayer: true,
              })
            }
          }
        }
      }
    } else {
      // Single active layer view
      const coords =
        layoutMode === 'pca'
          ? normalizeCoordinates(vertices)
          : new Map(vertices.map((v) => [v.id, { x: v.x, y: v.y }]))

      for (const v of vertices) {
        const key = String(v.id)
        if (!graph.hasNode(key)) {
          const color = v.isEntryPoint ? '#e0b15b' : getLayerColor(v.level)
          const pos = coords.get(v.id) ?? { x: v.x, y: v.y }
          graph.addNode(key, {
            id: v.id,
            vectorId: v.vectorId,
            level: v.level,
            x: pos.x,
            y: pos.y,
            size: v.isEntryPoint ? 7 : Math.max(3, Math.min(8, 2.5 + v.level * 1.2)),
            label: v.isEntryPoint ? '★' : '',
            labelColor,
            color,
            isEntryPoint: v.isEntryPoint,
          })
        }
      }

      for (const e of edges) {
        const u = String(e.from)
        const v = String(e.to)
        if (graph.hasNode(u) && graph.hasNode(v) && !graph.hasEdge(u, v)) {
          graph.addEdge(u, v, {
            size: 1,
            color: 'rgba(125, 143, 135, 0.25)',
            level: e.level,
          })
        }
      }

      if (layoutMode === 'force') {
        const state = visualStateRef.current
        const keepIds = new Set<number>([
          ...(state.selectedNodeId !== null ? [state.selectedNodeId] : []),
          ...(state.currentDescentNodeId !== null && state.currentDescentNodeId !== undefined
            ? [state.currentDescentNodeId]
            : []),
          ...(state.visitedNodeIds ? Array.from(state.visitedNodeIds) : []),
          ...(state.resultRanks ? Array.from(state.resultRanks.keys()) : []),
        ])
        for (const key of graph.nodes()) {
          if (graph.degree(key) === 0 && !keepIds.has(Number(graph.getNodeAttribute(key, 'id')))) {
            graph.dropNode(key)
          }
        }
      }

      if (layoutMode === 'force' && graph.order > 1) {
        forceAtlas2.assign(graph, {
          iterations: graph.order > 1500 ? 30 : 60,
          settings: {
            gravity: 8,
            scalingRatio: 1.5,
            strongGravityMode: true,
            barnesHutOptimize: true,
          },
        })
      }
    }

    // Clean up old sigma instance
    if (sigmaRef.current) {
      sigmaRef.current.kill()
      sigmaRef.current = null
    }

    const sigma = new Sigma(graph, containerRef.current, {
      settings: {
        itemSizesReference: 'screen',
        autoRescale: true,
        stagePadding: 30,
        renderLabels: true,
      },
      nodeReducer: (nodeKey, data) => {
        const {
          selectedNodeId: selId,
          hoveredNodeId: hovId,
          visitedNodeIds: visIds,
          currentDescentNodeId: descentId,
          tracePathIds: pathIds,
          resultRanks: resRanks,
          neighborhoodIds: nbrIds,
        } = visualStateRef.current

        const nodeId = Number(graph.getNodeAttribute(nodeKey, 'id'))
        const isCurrentDescent = descentId === nodeId
        const isVisited = visIds?.has(nodeId)
        const pathPosition = pathIds ? pathIds.indexOf(nodeId) : -1
        const isOnTrace = pathPosition >= 0
        const traceActive = Boolean(pathIds && pathIds.length > 0)
        const isResult = resRanks?.has(nodeId)
        const isSelected = selId === nodeId
        const isHovered = hovId === nodeId
        const inNeighborhood = nbrIds ? nbrIds.has(nodeId) : true
        const styled = (value: typeof data) => ({
          ...value,
          labelColor: isHovered ? '#111827' : labelColor,
        })
        const labelAttrs = isSelected
          ? {
              label: `#${nodeId}`,
              labelVisibility: 'visible' as const,
              labelSize: 16,
              forceLabel: true,
            }
          : {}

        if (!inNeighborhood && !isCurrentDescent && !isVisited && !isResult && !isSelected) {
          return styled({
            ...data,
            color: mutedLayerColor(String(data.color), theme, 0.65),
            label: '',
            zIndex: 0,
          })
        }

        if (isSelected) {
          return styled({
            ...data,
            ...labelAttrs,
            color: accentColor,
            size: Math.max(7.5, data.size * 1.8),
            highlighted: true,
            zIndex: 120,
          })
        }

        if (isResult) {
          return styled({
            ...data,
            label: '',
            color: RESULT_COLOR,
            size: Math.max(5.5, data.size * 1.6),
            highlighted: true,
            zIndex: 110,
          })
        }

        if (isOnTrace && isCurrentDescent) {
          return styled({
            ...data,
            color: CURRENT_STEP_COLOR,
            size: Math.max(8, data.size * 2),
            highlighted: true,
            zIndex: 100,
          })
        }

        if (isOnTrace) {
          return styled({
            ...data,
            label: '',
            color: ROUTE_COLOR,
            size: Math.max(6, data.size * 1.7),
            highlighted: true,
            zIndex: 90,
          })
        }

        if (traceActive) {
          return styled({
            ...data,
            color: mutedLayerColor(String(data.color), theme),
            label: '',
            zIndex: 0,
          })
        }

        if (isVisited) {
          return styled({
            ...data,
            label: '',
            color: accentColor,
            size: Math.max(6, data.size * 1.4),
            highlighted: true,
            zIndex: 50,
          })
        }

        if (isHovered) {
          return styled({
            ...data,
            size: data.size * 1.3,
            zIndex: 60,
          })
        }

        return styled(data)
      },
      edgeReducer: (edgeKey, data) => {
        const isInterLayer = Boolean(graph.getEdgeAttribute(edgeKey, 'isInterLayer'))
        if (isInterLayer) {
          return {
            ...data,
            color: 'rgba(125, 143, 135, 0.45)',
            size: 1.5,
          }
        }

        const { tracePathEdgeKeys: pathEdgeKeys, neighborhoodIds: nbrIds } = visualStateRef.current

        const [u, v] = graph.extremities(edgeKey)
        const uId = Number(graph.getNodeAttribute(u, 'id'))
        const vId = Number(graph.getNodeAttribute(v, 'id'))

        if (pathEdgeKeys.has(`${Math.min(uId, vId)}:${Math.max(uId, vId)}`)) {
          return {
            ...data,
            color: ROUTE_COLOR,
            size: 3,
            zIndex: 60,
          }
        }

        if (nbrIds && (!nbrIds.has(uId) || !nbrIds.has(vId))) {
          return {
            ...data,
            color: 'rgba(125, 143, 135, 0.05)',
            size: 0.5,
          }
        }

        return data
      },
    })

    sigma.on('clickNode', ({ node }) => {
      const id = Number(graph.getNodeAttribute(node, 'id'))
      callbacksRef.current.onSelectNode(id)
    })

    sigma.on('clickStage', () => {
      callbacksRef.current.onSelectNode(null)
    })

    sigma.on('enterNode', ({ node }) => {
      const id = Number(graph.getNodeAttribute(node, 'id'))
      callbacksRef.current.onHoverNode?.(id)
    })

    sigma.on('leaveNode', () => {
      callbacksRef.current.onHoverNode?.(null)
    })

    sigmaRef.current = sigma

    return () => {
      sigma.kill()
      sigmaRef.current = null
      graphRef.current = null
    }
  }, [
    vertices,
    edges,
    activeLevel,
    layoutMode,
    isMultilayer,
    allLayerVertices,
    labelColor,
    accentColor,
    theme,
  ])

  // Apply visual state before the graph effect rebuilds Sigma on a layer switch
  useLayoutEffect(() => {
    visualStateRef.current = {
      selectedNodeId,
      hoveredNodeId,
      visitedNodeIds,
      currentDescentNodeId,
      tracePathIds,
      tracePathEdgeKeys,
      resultRanks,
      neighborhoodIds,
    }
    if (sigmaRef.current) {
      sigmaRef.current.refresh()
    }
  }, [
    selectedNodeId,
    hoveredNodeId,
    visitedNodeIds,
    currentDescentNodeId,
    tracePathIds,
    tracePathEdgeKeys,
    resultRanks,
    neighborhoodIds,
  ])

  const zoomIn = () => {
    const camera = sigmaRef.current?.getCamera()
    if (camera) {
      void camera.zoomIn({ factor: 1.4, duration: 200 })
    }
  }

  const zoomOut = () => {
    const camera = sigmaRef.current?.getCamera()
    if (camera) {
      void camera.zoomOut({ factor: 1.4, duration: 200 })
    }
  }

  const resetView = () => {
    const camera = sigmaRef.current?.getCamera()
    if (camera) {
      void camera.reset({ duration: 250 })
    }
  }

  const toggleFullscreen = async () => {
    if (!rootRef.current) return
    try {
      if (document.fullscreenElement) {
        await document.exitFullscreen()
      } else {
        await rootRef.current.requestFullscreen()
      }
    } catch {
      // Ignore fullscreen rejection if denied by browser
    }
  }

  return (
    <div
      ref={rootRef}
      className={`relative h-full w-full overflow-hidden rounded-xl bg-canvas ${className ?? ''}`}
    >
      <div ref={containerRef} className="h-full w-full" />

      {/* Floating Canvas Controls */}
      <div
        role="toolbar"
        aria-label="Управление камерой графа"
        className="absolute bottom-3 right-3 z-20 flex flex-col gap-1 rounded-lg border border-line bg-surface/90 p-1 shadow-md backdrop-blur-xs"
      >
        <button
          type="button"
          onClick={zoomIn}
          className="rounded p-1.5 text-ink-2 transition-colors hover:bg-surface-2 hover:text-ink active:scale-95"
          title="Приблизить (+)"
        >
          <ZoomIn className="size-4" />
        </button>
        <button
          type="button"
          onClick={zoomOut}
          className="rounded p-1.5 text-ink-2 transition-colors hover:bg-surface-2 hover:text-ink active:scale-95"
          title="Отдалить (-)"
        >
          <ZoomOut className="size-4" />
        </button>
        <button
          type="button"
          onClick={resetView}
          className="rounded p-1.5 text-ink-2 transition-colors hover:bg-surface-2 hover:text-ink active:scale-95"
          title="Сбросить камеру и центрировать"
        >
          <Focus className="size-4" />
        </button>
        <button
          type="button"
          onClick={() => void toggleFullscreen()}
          className="rounded p-1.5 text-ink-2 transition-colors hover:bg-surface-2 hover:text-ink active:scale-95"
          title={isFullscreen ? 'Выйти из полноэкранного режима (Esc)' : 'Во весь экран'}
        >
          {isFullscreen ? <Minimize2 className="size-4" /> : <Maximize2 className="size-4" />}
        </button>
      </div>
    </div>
  )
}
