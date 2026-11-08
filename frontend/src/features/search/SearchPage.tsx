import { useQueries } from '@tanstack/react-query'
import { Network, Search as SearchIcon } from 'lucide-react'
import { useCallback, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { PageHeader } from '@/app/PageHeader'
import { api } from '@/api/endpoints'
import { useDataset } from '@/api/hooks/useDatasets'
import { useGraphQuery } from '@/api/hooks/useGraph'
import { useIndexStats, useIndexStatus } from '@/api/hooks/useIndex'
import { useSearchMutation, type SearchOutcome } from '@/api/hooks/useSearch'
import { queryKeys } from '@/api/keys'
import type { GraphEdge, GraphVertex, SearchInput, UISearchMode } from '@/api/schemas'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { EmptyState } from '@/components/ui/EmptyState'
import { GraphCanvas } from './graph/GraphCanvas'
import { GraphLegend } from './graph/GraphLegend'
import { LayerSelector } from './graph/LayerSelector'
import { LayoutToggle } from './graph/LayoutToggle'
import { NodeInfoPanel } from './graph/NodeInfoPanel'
import { NodeNeighborhoodControl } from './graph/NodeNeighborhoodControl'
import { SearchPathOverlay, type TraceStep } from './graph/SearchPathOverlay'
import type { LayoutMode } from './labels'
import { SearchForm } from './SearchForm'
import { SearchResults } from './SearchResults'

export function SearchPage() {
  const statusQuery = useIndexStatus()
  const isIndexReady = statusQuery.data?.status === 'ready'
  const statsQuery = useIndexStats(isIndexReady)
  const stats = statsQuery.data

  const datasetQuery = useDataset(stats?.datasetId ?? null)
  const dataset = datasetQuery.data

  // Search state
  const searchMutation = useSearchMutation()
  const [searchOutcome, setSearchOutcome] = useState<SearchOutcome | null>(null)

  // Graph state
  const [activeLevel, setActiveLevel] = useState<number>(0)
  const [graphLimit, setGraphLimit] = useState<number>(500)
  const [aroundId, setAroundId] = useState<string | undefined>(undefined)
  const [selectedNodeId, setSelectedNodeId] = useState<number | null>(null)
  const [hoveredNodeId, setHoveredNodeId] = useState<number | null>(null)
  const [layoutMode, setLayoutMode] = useState<LayoutMode>('pca')
  const [isMultilayer, setIsMultilayer] = useState<boolean>(false)
  const [hopRadius, setHopRadius] = useState<number>(1)

  // Trace animation state
  const [activeTraceStep, setActiveTraceStep] = useState<TraceStep | null>(null)
  const [showTraceOverlay, setShowTraceOverlay] = useState<boolean>(true)

  // Query graph for single active layer
  const graphQuery = useGraphQuery({
    level: activeLevel,
    limit: graphLimit,
    around: aroundId,
    enabled: isIndexReady && !isMultilayer,
  })

  // Queries for multilayer 2.5D view across all levels
  const maxLevel = stats?.maxLevel ?? 0
  const multilayerQueries = useQueries({
    queries: Array.from({ length: maxLevel + 1 }, (_, lvl) => ({
      queryKey: queryKeys.graph(lvl, 300, undefined),
      queryFn: ({ signal }: { signal?: AbortSignal }) =>
        api.graph({ level: lvl, limit: 300 }, signal),
      enabled: isIndexReady && isMultilayer,
      staleTime: 60_000,
    })),
  })

  const allLayerVertices = useMemo(() => {
    if (!isMultilayer) return undefined
    const map = new Map<number, GraphVertex[]>()
    multilayerQueries.forEach((q, lvl) => {
      if (q.data?.vertices) {
        map.set(lvl, q.data.vertices)
      }
    })
    return map
  }, [isMultilayer, multilayerQueries])

  const allLayerEdges = useMemo(() => {
    if (!isMultilayer) return []
    const list: GraphEdge[] = []
    multilayerQueries.forEach((q) => {
      if (q.data?.edges) {
        list.push(...q.data.edges)
      }
    })
    return list
  }, [isMultilayer, multilayerQueries])

  const isMultilayerLoading = isMultilayer && multilayerQueries.some((q) => q.isLoading)
  const isMultilayerError = isMultilayer && multilayerQueries.some((q) => q.isError)

  const isGraphLoading = isMultilayer ? isMultilayerLoading : graphQuery.isLoading
  const isGraphError = isMultilayer ? isMultilayerError : graphQuery.isError

  const vertices = useMemo(
    () => (isMultilayer ? [] : (graphQuery.data?.vertices ?? [])),
    [isMultilayer, graphQuery.data?.vertices],
  )
  const edges = useMemo(
    () => (isMultilayer ? allLayerEdges : (graphQuery.data?.edges ?? [])),
    [isMultilayer, allLayerEdges, graphQuery.data?.edges],
  )

  // Map of neighbors on current layer for selected node
  const inLayerNeighbors = useMemo(() => {
    if (selectedNodeId === null) return []
    const neighbors: number[] = []
    for (const e of edges) {
      if (e.from === selectedNodeId) neighbors.push(e.to)
      else if (e.to === selectedNodeId) neighbors.push(e.from)
    }
    return neighbors
  }, [edges, selectedNodeId])

  // Hop neighborhood set (Task 7.7)
  const neighborhoodIds = useMemo<Set<number> | null>(() => {
    if (selectedNodeId === null || hopRadius <= 0) return null
    const visited = new Set<number>([selectedNodeId])
    let frontier = new Set<number>([selectedNodeId])

    for (let hop = 0; hop < hopRadius; hop += 1) {
      const nextFrontier = new Set<number>()
      for (const u of frontier) {
        for (const e of edges) {
          if (e.from === u && !visited.has(e.to)) {
            visited.add(e.to)
            nextFrontier.add(e.to)
          } else if (e.to === u && !visited.has(e.from)) {
            visited.add(e.from)
            nextFrontier.add(e.from)
          }
        }
      }
      frontier = nextFrontier
      if (frontier.size === 0) break
    }

    return visited
  }, [edges, hopRadius, selectedNodeId])

  // Selected vertex details
  const selectedVertex = useMemo<GraphVertex | null>(() => {
    if (selectedNodeId === null) return null
    if (isMultilayer && allLayerVertices) {
      for (const layerVerts of allLayerVertices.values()) {
        const found = layerVerts.find((v) => v.id === selectedNodeId)
        if (found) return found
      }
      return null
    }
    return vertices.find((v) => v.id === selectedNodeId) ?? null
  }, [selectedNodeId, vertices, isMultilayer, allLayerVertices])

  // Results rank map
  const resultRanks = useMemo(() => {
    const map = new Map<number, number>()
    const list =
      searchOutcome?.mode === 'bruteforce'
        ? (searchOutcome.bruteforce?.results ?? [])
        : (searchOutcome?.hnsw?.results ?? [])
    list.forEach((item, idx) => map.set(item.id, idx + 1))
    return map
  }, [searchOutcome])

  // Search trace visited node IDs on current layer
  const visitedNodeIds = useMemo(() => {
    if (!activeTraceStep) return undefined
    if (activeTraceStep.level !== activeLevel) return undefined
    return new Set(activeTraceStep.visitedOnLevel)
  }, [activeLevel, activeTraceStep])

  // Set of levels traversed by trace
  const visitedLevels = useMemo(() => {
    const trace = searchOutcome?.hnsw?.trace
    if (!trace) return new Set<number>()
    return new Set(trace.layers.map((l) => l.level))
  }, [searchOutcome?.hnsw?.trace])

  const handleSearchSubmit = (input: SearchInput, uiMode: UISearchMode) => {
    searchMutation.mutate(
      { input, uiMode },
      {
        onSuccess: (outcome) => {
          setSearchOutcome(outcome)
          if (outcome.hnsw?.trace) {
            setShowTraceOverlay(true)
            const topLayer = outcome.hnsw.trace.layers[0]?.level
            if (topLayer !== undefined) {
              setActiveLevel(topLayer)
              setAroundId(undefined)
            }
          }
        },
      },
    )
  }

  const handleStepChange = useCallback((step: TraceStep) => {
    setActiveTraceStep(step)
    setActiveLevel(step.level)
  }, [])

  if (!isIndexReady) {
    return (
      <>
        <PageHeader
          title="Поиск и визуализация графа"
          description="k-NN запросы, сравнение HNSW с полным перебором и послойная визуализация графа."
        />
        <Card title="Индекс не готов">
          <EmptyState
            icon={Network}
            title="Индекс ещё не построен"
            description="Чтобы выполнять k-NN запросы и исследовать граф HNSW, сначала постройте индекс по датасету."
            action={
              <Link to="/index">
                <Button variant="primary">Перейти к построению индекса</Button>
              </Link>
            }
          />
        </Card>
      </>
    )
  }

  return (
    <>
      <PageHeader
        title="Поиск и визуализация графа"
        description="k-NN запросы, сравнение HNSW с полным перебором и интерактивная визуализация слоёв графа."
      />

      <div className="grid gap-4 xl:grid-cols-[400px_minmax(0,1fr)] xl:items-start">
        {/* Left Column: Search Form, Results, Node Inspector */}
        <div className="flex flex-col gap-4">
          <SearchForm
            dataset={dataset}
            searching={searchMutation.isPending}
            onSubmit={handleSearchSubmit}
          />

          {searchOutcome ? (
            <SearchResults
              outcome={searchOutcome}
              selectedNodeId={selectedNodeId}
              hoveredNodeId={hoveredNodeId}
              onSelectNode={(id) => setSelectedNodeId(id)}
              onHoverNode={(id) => setHoveredNodeId(id)}
            />
          ) : null}

          {selectedVertex ? (
            <NodeInfoPanel
              vertex={selectedVertex}
              neighbors={inLayerNeighbors}
              activeLevel={activeLevel}
              vector={
                dataset && selectedVertex.id < dataset.preview.length
                  ? dataset.preview[selectedVertex.id]
                  : undefined
              }
              onCenterAround={(id) => setAroundId(id)}
              onSelectNeighbor={(nid) => setSelectedNodeId(nid)}
              onClose={() => setSelectedNodeId(null)}
            />
          ) : null}
        </div>

        {/* Right Column: Interactive Graph Canvas */}
        <div className="flex flex-col overflow-hidden rounded-xl border border-line bg-surface shadow-xs">
          {/* Top Layer Selector */}
          <LayerSelector
            maxLevel={stats?.maxLevel ?? 0}
            activeLevel={activeLevel}
            nodesPerLevel={stats?.nodesPerLevel ?? []}
            visitedLevels={visitedLevels}
            isMultilayer={isMultilayer}
            onSelectLevel={(lvl) => {
              setIsMultilayer(false)
              setActiveLevel(lvl)
              setAroundId(undefined)
            }}
            onToggleMultilayer={() => {
              setIsMultilayer((prev) => !prev)
              setAroundId(undefined)
            }}
            totalEdges={edges.length}
          />

          {/* Subheader Toolbar: Layout, Neighborhood, Legend */}
          <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line bg-surface-2/60 px-4 py-2">
            <div className="flex flex-wrap items-center gap-3">
              <LayoutToggle mode={layoutMode} onChange={setLayoutMode} disabled={isMultilayer} />
              <NodeNeighborhoodControl
                limit={graphLimit}
                onLimitChange={setGraphLimit}
                around={aroundId}
                onClearAround={() => setAroundId(undefined)}
                hopRadius={hopRadius}
                onHopRadiusChange={setHopRadius}
                hasSelection={selectedNodeId !== null}
              />
            </div>
            <GraphLegend isMultilayer={isMultilayer} />
          </div>

          {/* WebGL Canvas */}
          <div className="relative h-[620px] w-full">
            {isGraphLoading ? (
              <div className="flex h-full w-full items-center justify-center bg-canvas">
                <div className="flex flex-col items-center gap-2">
                  <div className="size-6 animate-spin rounded-full border-2 border-line border-t-accent" />
                  <span className="font-mono text-xs text-ink-3">
                    {isMultilayer ? 'Загрузка всех слоёв...' : `Загрузка слоя L${activeLevel}...`}
                  </span>
                </div>
              </div>
            ) : isGraphError ? (
              <div className="flex h-full w-full items-center justify-center bg-canvas p-6">
                <EmptyState
                  icon={SearchIcon}
                  title="Не удалось загрузить граф слоя"
                  description="Проверьте соединение с сервером или выберите другой уровень."
                  action={
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => {
                        if (isMultilayer) {
                          multilayerQueries.forEach((q) => void q.refetch())
                        } else {
                          void graphQuery.refetch()
                        }
                      }}
                    >
                      Повторить
                    </Button>
                  }
                />
              </div>
            ) : (
              <GraphCanvas
                vertices={vertices}
                edges={edges}
                activeLevel={activeLevel}
                selectedNodeId={selectedNodeId}
                hoveredNodeId={hoveredNodeId}
                visitedNodeIds={visitedNodeIds}
                currentDescentNodeId={activeTraceStep?.nodeId}
                resultRanks={resultRanks}
                neighborhoodIds={neighborhoodIds}
                layoutMode={layoutMode}
                isMultilayer={isMultilayer}
                allLayerVertices={allLayerVertices}
                onSelectNode={(id) => setSelectedNodeId(id)}
                onHoverNode={(id) => setHoveredNodeId(id)}
              />
            )}

            {/* Trace Step Player HUD */}
            {searchOutcome?.hnsw?.trace && showTraceOverlay ? (
              <div className="absolute bottom-4 left-4 right-4 z-10 mx-auto max-w-2xl">
                <SearchPathOverlay
                  trace={searchOutcome.hnsw.trace}
                  onStepChange={handleStepChange}
                  onClose={() => setShowTraceOverlay(false)}
                />
              </div>
            ) : null}
          </div>
        </div>
      </div>
    </>
  )
}
