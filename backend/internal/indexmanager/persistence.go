package indexmanager

import (
	"encoding/gob"
	"fmt"
	"os"
	"slices"

	"hnsw-coursework/internal/bruteforce"
	"hnsw-coursework/internal/hnsw"
)

type persistedManager struct {
	Version int
	Meta    PersistMeta
	Index   hnsw.GraphSnapshot
	Vectors [][]float32
}

func (m *Manager) Save(path string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.status != StatusReady || m.idx == nil {
		return m.notReadyErrLocked()
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("index: create save file: %w", err)
	}
	defer f.Close()
	p := persistedManager{Version: 1, Meta: PersistMeta{IndexID: m.indexID, DatasetID: m.datasetID, ExternalIDs: slices.Clone(m.extIDs)}, Index: m.idx.GraphSnapshot()}
	p.Vectors = make([][]float32, len(m.vectors))
	for i := range m.vectors {
		p.Vectors[i] = slices.Clone(m.vectors[i])
	}
	if err := gob.NewEncoder(f).Encode(p); err != nil {
		return fmt.Errorf("index: save: %w", err)
	}
	return nil
}

func (m *Manager) Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("index: open: %w", err)
	}
	defer f.Close()
	var p persistedManager
	if err := gob.NewDecoder(f).Decode(&p); err != nil {
		return fmt.Errorf("index: load: %w", err)
	}
	if p.Version != 1 || len(p.Vectors) == 0 || len(p.Meta.ExternalIDs) != len(p.Vectors) {
		return fmt.Errorf("index: invalid saved manager")
	}
	idx, err := hnsw.LoadSnapshot(p.Index)
	if err != nil {
		return err
	}
	bf, err := bruteforce.NewWithWorkers(p.Vectors, hnsw.Distance(idx.Config().Metric), 0)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.status == StatusBuilding {
		return ErrBuildInProgress
	}
	m.idx, m.bf, m.vectors = idx, bf, p.Vectors
	m.extIDs, m.extToInt = slices.Clone(p.Meta.ExternalIDs), make(map[string]uint32, len(p.Meta.ExternalIDs))
	for i, id := range m.extIDs {
		m.extToInt[id] = uint32(i)
	}
	m.indexID, m.datasetID, m.metric = p.Meta.IndexID, p.Meta.DatasetID, idx.Config().Metric
	m.status, m.progress, m.processed, m.total, m.lastErr = StatusReady, 1, len(p.Vectors), len(p.Vectors), ""
	return nil
}
