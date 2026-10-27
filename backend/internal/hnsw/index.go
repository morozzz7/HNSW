package hnsw

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
)

// node is a single element in the HNSW graph. Its internal id is its index in
// the Index.nodes slice. neighbors[l] holds the ids of adjacent nodes at level l.
type node struct {
	id        uint32 // internal id == position in Index.nodes
	vector    []float32
	maxLevel  int        // highest level this node exists at (0-based)
	neighbors [][]uint32 // neighbors[l] = adjacent node ids at level l, length = maxLevel+1
}

// Config holds HNSW parameters (Malkov & Yashunin, 2016).
type Config struct {
	M                int    // max connections per node per level (non-zero levels)
	Mmax0            int    // max connections at level 0; 0 means 2*M
	EfConstruction   int    // size of dynamic candidate list during insert
	Metric           Metric // distance metric
	Seed             int64  // seed for reproducible level generation
	UseHeuristic     bool   // use heuristic (Alg. 4) vs simple (Alg. 3) neighbor selection
	ExtendCandidates bool   // consider pruned candidates when selecting (reserved)
	KeepPruned       bool   // keep pruned connections (reserved)
}

// defaultConfig returns a sensible default configuration.
func defaultConfig() Config {
	return Config{
		M:              16,
		Mmax0:          0, // resolved to 2*M at index creation
		EfConstruction: 200,
		Metric:         MetricL2,
		Seed:           0,
		UseHeuristic:   true,
	}
}

// resolve fills in defaults and validates the config.
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

// Mmax returns the max degree allowed at the given level (Mmax0 at level 0, M elsewhere).
func (c Config) Mmax(level int) int {
	if level == 0 {
		return c.Mmax0
	}
	return c.M
}

// Stats is a snapshot of index structure, used by the API /index/stats endpoint.
type Stats struct {
	Nodes           int
	MaxLevel        int
	EntryPoint      uint32
	NodesPerLevel   []int // nodesPerLevel[l] = number of nodes that exist at level l
	DegreeHistogram map[int]int
	Metric          Metric
	M               int
	Mmax0           int
	EfConstruction  int
}

// Index is the HNSW graph index. It is safe for concurrent use: searches take a
// read lock, inserts take a write lock.
type Index struct {
	mu         sync.RWMutex
	config     Config
	dist       DistanceFunc
	rng        *rand.Rand
	nodes      []*node
	dim        int
	entryPoint uint32
	maxLevel   int
	mL         float64 // 1 / ln(M)

	// external id mapping
	extToInt map[uint32]uint32 // external id -> internal index
	intToExt []uint32          // internal index -> external id
}

// New creates an empty index with the given config. Zero values are replaced
// with defaults (see defaultConfig / resolve).
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

// Len returns the number of indexed vectors.
func (idx *Index) Len() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.nodes)
}

// Config returns a copy of the index configuration.
func (idx *Index) Config() Config {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.config
}

// Stats returns a snapshot of index structure.
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

// randomLevel samples a level using l = floor(-ln(U(0,1)) * mL), mL = 1/ln(M).
// Returns a level >= 0.
func (idx *Index) randomLevel() int {
	u := idx.rng.Float64()
	if u <= 0 {
		u = 1e-18
	} else if u >= 1 {
		u = 1 - 1e-18
	}
	return int(math.Floor(-math.Log(u) * idx.mL))
}

// dim returns the vector dimension (0 if not yet set).
func (idx *Index) Dim() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.dim
}

// ensureDim checks that vector matches the index dimension, setting it on first insert.
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
