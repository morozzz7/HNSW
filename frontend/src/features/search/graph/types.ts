import type { GraphVertex } from '@/api/schemas'

export type NodeRole = 'default' | 'entryPoint' | 'visited' | 'closest' | 'result' | 'muted'

export interface GraphVisualState {
  selectedNodeId: number | null
  hoveredNodeId: number | null
  activeVisitedIds: Set<number>
  closestNodeId: number | null
  resultRanks: Map<number, number>
  neighborhoodIds: Set<number> | null
  isMultilayer: boolean
  layoutMode: 'pca' | 'force'
}

export interface NodeDetails {
  vertex: GraphVertex
  neighbors: number[]
  degree: number
}
