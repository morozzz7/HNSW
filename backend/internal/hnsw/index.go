package hnsw

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"sync"
)

type node struct {
	id        uint32
	vector    []float32
	maxLevel  int
	neighbors [][]uint32
}

type Config struct {
	M                int
	Mmax0            int
	EfConstruction   int
	Metric           Metric
	Seed             int64
	UseHeuristic     bool
	ExtendCandidates bool
	KeepPruned       bool
}

func defaultConfig() Config {
	return Config{
		M:              16,
		Mmax0:          0,
		EfConstruction: 200,
		Metric:         MetricL2,
		Seed:           0,
		UseHeuristic:   true,
	}
}

func (c Config) resolve() Config {
	cfg := c
	if cfg.M == 0 {
		cfg.M = 16
	}
	if cfg.Mmax0 == 0 {
		cfg.Mmax0 = 2 * cfg.M
	}
	if cfg.EfConstruction == 0 {
		cfg.EfConstruction = 200
	}
	if cfg.Metric == "" {
		cfg.Metric = MetricL2
	}
	return cfg
}

func (c Config) Validate() error {
	if c.M < MinM || c.M > MaxM {
		return fmt.Errorf("M must be between %d and %d, got %d", MinM, MaxM, c.M)
	}
	if c.Mmax0 < c.M || c.Mmax0 > 4*c.M {
		return fmt.Errorf("Mmax0 must be between %d and %d, got %d", c.M, 4*c.M, c.Mmax0)
	}
	if c.EfConstruction < 1 || c.EfConstruction > MaxEfConstruction {
		return fmt.Errorf("efConstruction must be between 1 and %d, got %d", MaxEfConstruction, c.EfConstruction)
	}
	switch c.Metric {
	case MetricL2, MetricCosine, MetricInnerProduct:
	default:
		return fmt.Errorf("unknown metric %q", c.Metric)
	}
	return nil
}

func (c Config) Mmax(level int) int {
	if level == 0 {
		return c.Mmax0
	}
	return c.M
}

type Stats struct {
	Nodes           int
	MaxLevel        int
	EntryPoint      uint32
	NodesPerLevel   []int
	DegreeHistogram map[int]int
	Metric          Metric
	M               int
	Mmax0           int
	EfConstruction  int
}

type GraphNode struct {
	ID        uint32
	External  uint32
	Vector    []float32
	MaxLevel  int
	Neighbors [][]uint32
}

type GraphSnapshot struct {
	Config     Config
	Dimension  int
	EntryPoint uint32
	MaxLevel   int
	Nodes      []GraphNode
}

type Index struct {
	mu         sync.RWMutex
	config     Config
	dist       DistanceFunc
	rng        *rand.Rand
	nodes      []*node
	dim        int
	entryPoint uint32
	maxLevel   int
	mL         float64

	extToInt map[uint32]uint32
	intToExt []uint32
}

func New(cfg Config) (*Index, error) {
	cfg = cfg.resolve()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Index{
		config:     cfg,
		dist:       Distance(cfg.Metric),
		rng:        rand.New(rand.NewSource(cfg.Seed)),
		nodes:      make([]*node, 0, 64),
		dim:        0,
		entryPoint: 0,
		maxLevel:   -1,
		mL:         1.0 / math.Log(float64(cfg.M)),
		extToInt:   make(map[uint32]uint32, 64),
		intToExt:   make([]uint32, 0, 64),
	}, nil
}

func MustNew(cfg Config) *Index {
	idx, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return idx
}

func (idx *Index) Len() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.nodes)
}

func (idx *Index) Config() Config {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.config
}

func (idx *Index) Stats() Stats {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	s := Stats{
		Nodes:           len(idx.nodes),
		MaxLevel:        idx.maxLevel,
		EntryPoint:      idx.entryPoint,
		NodesPerLevel:   make([]int, idx.maxLevel+1),
		DegreeHistogram: make(map[int]int),
		Metric:          idx.config.Metric,
		M:               idx.config.M,
		Mmax0:           idx.config.Mmax0,
		EfConstruction:  idx.config.EfConstruction,
	}
	if s.MaxLevel < 0 {
		s.MaxLevel = 0
		s.NodesPerLevel = []int{0}
	}
	for _, nd := range idx.nodes {
		for l := 0; l <= nd.maxLevel; l++ {
			if l < len(s.NodesPerLevel) {
				s.NodesPerLevel[l]++
			}
			deg := len(nd.neighbors[l])
			s.DegreeHistogram[deg]++
		}
	}
	return s
}

func (idx *Index) randomLevel() int {
	u := idx.rng.Float64()
	if u <= 0 {
		u = 1e-18
	} else if u >= 1 {
		u = 1 - 1e-18
	}
	return int(math.Floor(-math.Log(u) * idx.mL))
}

func (idx *Index) Dim() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.dim
}

func (idx *Index) GraphSnapshot() GraphSnapshot {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	s := GraphSnapshot{Config: idx.config, Dimension: idx.dim, EntryPoint: idx.entryPoint, MaxLevel: idx.maxLevel}
	s.Nodes = make([]GraphNode, len(idx.nodes))
	for i, nd := range idx.nodes {
		n := GraphNode{ID: nd.id, MaxLevel: nd.maxLevel, Vector: append([]float32(nil), nd.vector...), Neighbors: make([][]uint32, len(nd.neighbors))}
		if i < len(idx.intToExt) {
			n.External = idx.intToExt[i]
		}
		for l := range nd.neighbors {
			n.Neighbors[l] = append([]uint32(nil), nd.neighbors[l]...)
		}
		s.Nodes[i] = n
	}
	return s
}

type persistedIndex struct {
	Version  int
	Snapshot GraphSnapshot
}

func (idx *Index) Save(w io.Writer) error {
	if w == nil {
		return errors.New("hnsw: nil save writer")
	}
	return gob.NewEncoder(w).Encode(persistedIndex{Version: 1, Snapshot: idx.GraphSnapshot()})
}

func Load(r io.Reader) (*Index, error) {
	if r == nil {
		return nil, errors.New("hnsw: nil load reader")
	}
	var p persistedIndex
	if err := gob.NewDecoder(r).Decode(&p); err != nil {
		return nil, fmt.Errorf("hnsw: decode index: %w", err)
	}
	if p.Version != 1 {
		return nil, fmt.Errorf("hnsw: unsupported index version %d", p.Version)
	}
	s := p.Snapshot
	if err := s.Config.Validate(); err != nil {
		return nil, fmt.Errorf("hnsw: invalid saved config: %w", err)
	}
	if s.Dimension <= 0 || len(s.Nodes) == 0 {
		return nil, errors.New("hnsw: saved index is empty or has invalid dimension")
	}
	return LoadSnapshot(s)
}

func LoadSnapshot(s GraphSnapshot) (*Index, error) {
	if err := s.Config.Validate(); err != nil {
		return nil, fmt.Errorf("hnsw: invalid saved config: %w", err)
	}
	if s.Dimension <= 0 || len(s.Nodes) == 0 {
		return nil, errors.New("hnsw: saved index is empty or has invalid dimension")
	}
	idx, err := New(s.Config)
	if err != nil {
		return nil, err
	}
	idx.dim, idx.entryPoint, idx.maxLevel = s.Dimension, s.EntryPoint, s.MaxLevel
	idx.nodes = make([]*node, len(s.Nodes))
	idx.intToExt = make([]uint32, len(s.Nodes))
	idx.extToInt = make(map[uint32]uint32, len(s.Nodes))
	for i, sn := range s.Nodes {
		if int(sn.ID) != i || len(sn.Vector) != s.Dimension || sn.MaxLevel < 0 || len(sn.Neighbors) != sn.MaxLevel+1 {
			return nil, fmt.Errorf("hnsw: invalid saved node %d", i)
		}
		nd := &node{id: sn.ID, vector: append([]float32(nil), sn.Vector...), maxLevel: sn.MaxLevel, neighbors: make([][]uint32, len(sn.Neighbors))}
		for l, ids := range sn.Neighbors {
			for _, id := range ids {
				if int(id) >= len(s.Nodes) || id == sn.ID {
					return nil, fmt.Errorf("hnsw: invalid edge %d -> %d", sn.ID, id)
				}
			}
			nd.neighbors[l] = append([]uint32(nil), ids...)
		}
		idx.nodes[i], idx.intToExt[i] = nd, sn.External
		if _, ok := idx.extToInt[sn.External]; ok {
			return nil, errors.New("hnsw: duplicate external id")
		}
		idx.extToInt[sn.External] = sn.ID
	}
	if int(idx.entryPoint) >= len(idx.nodes) || idx.maxLevel < 0 {
		return nil, errors.New("hnsw: invalid entry point")
	}
	return idx, nil
}

func (idx *Index) ensureDim(vector []float32) error {
	if len(vector) == 0 {
		return errors.New("hnsw: zero-dimensional vector")
	}
	if idx.dim == 0 {
		idx.dim = len(vector)
		return nil
	}
	if len(vector) != idx.dim {
		return fmt.Errorf("vector dimension %d does not match index dimension %d",
			len(vector), idx.dim)
	}
	return nil
}
