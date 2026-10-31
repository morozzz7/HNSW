package indexmanager

import (
	"fmt"
	"slices"

	"hnsw-coursework/internal/projection"
)

type GraphVertex struct {
	ID           uint32
	VectorID     string
	Level        int
	X            float64
	Y            float64
	IsEntryPoint bool
}

type GraphEdge struct {
	From  uint32
	To    uint32
	Level int
}

type Graph struct {
	Level         int
	Vertices      []GraphVertex
	Edges         []GraphEdge
	TotalVertices int
}

func (m *Manager) Graph(level int, limit int, around string) (Graph, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.status != StatusReady || m.idx == nil {
		return Graph{}, m.notReadyErrLocked()
	}
	s := m.idx.GraphSnapshot()
	if level < 0 || level > s.MaxLevel {
		return Graph{}, fmt.Errorf("%w: graph level %d is outside 0..%d", ErrInvalid, level, s.MaxLevel)
	}
	if limit <= 0 {
		return Graph{}, fmt.Errorf("%w: graph limit must be positive", ErrInvalid)
	}
	coords, err := projection.PCA2D(m.vectors)
	if err != nil {
		return Graph{}, fmt.Errorf("graph projection: %w", err)
	}
	selected := make(map[uint32]struct{}, limit)
	if around != "" {
		id, ok := m.extToInt[around]
		if !ok {
			var parsed uint32
			_, err := fmt.Sscanf(around, "%d", &parsed)
			if err == nil && int(parsed) < len(s.Nodes) {
				id, ok = parsed, true
			}
		}
		if !ok {
			return Graph{}, fmt.Errorf("%w: unknown around id %q", ErrInvalid, around)
		}
		selected[id] = struct{}{}
		for _, nid := range s.Nodes[id].Neighbors[level] {
			if len(selected) < limit {
				selected[nid] = struct{}{}
			}
		}
	} else {
		for i := range s.Nodes {
			if len(selected) == limit {
				break
			}
			selected[uint32(i)] = struct{}{}
		}
	}
	ids := make([]uint32, 0, len(selected))
	for id := range selected {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	vertices := make([]GraphVertex, 0, len(ids))
	for _, id := range ids {
		n := s.Nodes[id]
		vertices = append(vertices, GraphVertex{ID: id, VectorID: m.externalIDLocked(id), Level: n.MaxLevel, X: coords[id][0], Y: coords[id][1], IsEntryPoint: id == s.EntryPoint})
	}
	edges := make([]GraphEdge, 0)
	for _, id := range ids {
		for _, to := range s.Nodes[id].Neighbors[level] {
			if _, ok := selected[to]; ok && id < to {
				edges = append(edges, GraphEdge{From: id, To: to, Level: level})
			}
		}
	}
	return Graph{Level: level, Vertices: vertices, Edges: edges, TotalVertices: len(s.Nodes)}, nil
}

type PersistMeta struct {
	IndexID, DatasetID string
	ExternalIDs        []string
}
