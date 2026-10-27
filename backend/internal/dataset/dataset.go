// Package dataset provides in-memory datasets of equal-dimension float32
// vectors, loaders (CSV, JSON, .fvecs/.ivecs), synthetic generators, and
// validation. Vectors are []float32; internal ids are row indices.
package dataset

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"
)

// Limits guard against accidentally loading enormous files.
const (
	MaxVectors = 1_000_000
	MaxDim     = 8192
)

// previewRows is the number of vectors exposed in the dataset preview.
const previewRows = 10

// Dataset is a collection of equal-dimension vectors with metadata.
type Dataset struct {
	ID           string      `json:"id"`
	N            int         `json:"n"`
	Dim          int         `json:"dim"`
	Vectors      [][]float32 `json:"-"`       // full vectors; not exposed in list/preview
	Preview      [][]float32 `json:"preview"` // first previewRows vectors
	CreatedAt    time.Time   `json:"createdAt"`
	Source       string      `json:"source"` // csv | json | fvecs | ivecs | generate
	Distribution string      `json:"distribution,omitempty"`
	Stats        *Stats      `json:"stats,omitempty"`
	Path         string      `json:"-"` // source file path, if loaded from disk
}

// Stats holds per-dimension min/max/mean.
type Stats struct {
	Min  []float32 `json:"min"`
	Max  []float32 `json:"max"`
	Mean []float32 `json:"mean"`
}

// NewDataset validates vectors, computes stats and preview, and returns a
// Dataset with a generated id. It does not register the dataset in a store.
func NewDataset(source, distribution string, vectors [][]float32) (*Dataset, error) {
	if err := Validate(vectors); err != nil {
		return nil, err
	}

	n := len(vectors)
	dim := len(vectors[0])

	preview := make([][]float32, min(n, previewRows))
	copy(preview, vectors[:len(preview)])

	stats := computeStats(vectors)

	ds := &Dataset{
		N:            n,
		Dim:          dim,
		Vectors:      vectors,
		Preview:      preview,
		CreatedAt:    time.Now().UTC(),
		Source:       source,
		Distribution: distribution,
		Stats:        stats,
	}
	return ds, nil
}

// Validate checks that vectors are non-empty, equal-dimension, free of
// NaN/Inf, and within size limits.
func Validate(vectors [][]float32) error {
	if len(vectors) == 0 {
		return errors.New("dataset: no vectors")
	}
	if len(vectors) > MaxVectors {
		return fmt.Errorf("dataset: %d vectors exceeds limit %d", len(vectors), MaxVectors)
	}

	dim := len(vectors[0])
	if dim == 0 {
		return errors.New("dataset: zero-dimensional vector")
	}
	if dim > MaxDim {
		return fmt.Errorf("dataset: dimension %d exceeds limit %d", dim, MaxDim)
	}

	for i, v := range vectors {
		if len(v) != dim {
			return fmt.Errorf("dataset: vector %d has dimension %d, want %d", i, len(v), dim)
		}
		for j, x := range v {
			if math.IsNaN(float64(x)) {
				return fmt.Errorf("dataset: vector %d dimension %d is NaN", i, j)
			}
			if math.IsInf(float64(x), 0) {
				return fmt.Errorf("dataset: vector %d dimension %d is Inf", i, j)
			}
		}
	}
	return nil
}

// computeStats returns per-dimension min, max and mean.
func computeStats(vectors [][]float32) *Stats {
	n := len(vectors)
	dim := len(vectors[0])
	min := make([]float32, dim)
	max := make([]float32, dim)
	mean := make([]float32, dim)
	copy(min, vectors[0])
	copy(max, vectors[0])
	for _, v := range vectors {
		for j, x := range v {
			if x < min[j] {
				min[j] = x
			}
			if x > max[j] {
				max[j] = x
			}
			mean[j] += x
		}
	}
	for j := range mean {
		mean[j] /= float32(n)
	}
	return &Stats{Min: min, Max: max, Mean: mean}
}

// Store is an in-memory store of datasets, safe for concurrent use.
type Store struct {
	mu       sync.RWMutex
	datasets map[string]*Dataset
	nextID   int
}

// NewStore creates an empty dataset store.
func NewStore() *Store {
	return &Store{
		datasets: make(map[string]*Dataset),
		nextID:   1,
	}
}

// Add registers a dataset and returns its id.
func (s *Store) Add(ds *Dataset) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := fmt.Sprintf("ds_%06d", s.nextID)
	s.nextID++
	ds.ID = id
	s.datasets[id] = ds
	return id
}

// Get returns the dataset with the given id, or nil if not found.
func (s *Store) Get(id string) *Dataset {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.datasets[id]
}

// List returns all datasets (without full vectors).
func (s *Store) List() []*Dataset {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Dataset, 0, len(s.datasets))
	for _, ds := range s.datasets {
		cp := *ds
		cp.Vectors = nil // do not expose full vectors in list
		out = append(out, &cp)
	}
	return out
}

// Remove deletes the dataset with the given id. Returns true if removed.
func (s *Store) Remove(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.datasets[id]; !ok {
		return false
	}
	delete(s.datasets, id)
	return true
}

// Count returns the number of stored datasets.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.datasets)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
