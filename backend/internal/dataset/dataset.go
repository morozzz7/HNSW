package dataset

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"
)

const (
	MaxVectors = 1_000_000
	MaxDim     = 8192
)

const previewRows = 10

type Dataset struct {
	ID           string      `json:"id"`
	N            int         `json:"n"`
	Dim          int         `json:"dim"`
	Vectors      [][]float32 `json:"-"`
	Preview      [][]float32 `json:"preview"`
	CreatedAt    time.Time   `json:"createdAt"`
	Source       string      `json:"source"`
	Distribution string      `json:"distribution,omitempty"`
	Stats        *Stats      `json:"stats,omitempty"`
	Path         string      `json:"-"`
}

type Stats struct {
	Min  []float32 `json:"min"`
	Max  []float32 `json:"max"`
	Mean []float32 `json:"mean"`
}

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

type Store struct {
	mu       sync.RWMutex
	datasets map[string]*Dataset
	nextID   int
}

func NewStore() *Store {
	return &Store{
		datasets: make(map[string]*Dataset),
		nextID:   1,
	}
}

func (s *Store) Add(ds *Dataset) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := fmt.Sprintf("ds_%06d", s.nextID)
	s.nextID++
	ds.ID = id
	s.datasets[id] = ds
	return id
}

func (s *Store) Get(id string) *Dataset {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.datasets[id]
}

func (s *Store) List() []*Dataset {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Dataset, 0, len(s.datasets))
	for _, ds := range s.datasets {
		cp := *ds
		cp.Vectors = nil
		out = append(out, &cp)
	}
	return out
}

func (s *Store) Remove(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.datasets[id]; !ok {
		return false
	}
	delete(s.datasets, id)
	return true
}

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
