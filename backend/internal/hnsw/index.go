package hnsw

import (
	"fmt"
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
	if cfg.M <= 0 {
		cfg.M = 16
	}
	if cfg.Mmax0 <= 0 {
		cfg.Mmax0 = 2 * cfg.M
	}
	if cfg.EfConstruction <= 0 {
		cfg.EfConstruction = 200
	}
	if cfg.Metric == "" {
		cfg.Metric = MetricL2
	}
	return cfg
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

func New(cfg Config) *Index {
	cfg = cfg.resolve()
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
	}
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

func (idx *Index) ensureDim(vector []float32) error {
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
