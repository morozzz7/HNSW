package indexmanager

import (
	"slices"

	"hnsw-coursework/internal/hnsw"
)

type BenchmarkSnapshot struct {
	IndexID string
	Vectors [][]float32
	Metric  hnsw.Metric
	Seed    int64
}

func (m *Manager) BenchmarkSnapshot() (BenchmarkSnapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.status != StatusReady || m.idx == nil {
		return BenchmarkSnapshot{}, m.notReadyErrLocked()
	}
	v := make([][]float32, len(m.vectors))
	for i := range m.vectors {
		v[i] = slices.Clone(m.vectors[i])
	}
	cfg := m.idx.Config()
	return BenchmarkSnapshot{IndexID: m.indexID, Vectors: v, Metric: cfg.Metric, Seed: cfg.Seed}, nil
}
