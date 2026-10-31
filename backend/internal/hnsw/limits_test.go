package hnsw

import (
	"math/rand"
	"testing"
)

func TestNewValidatesConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{"M too small", Config{M: 1}},
		{"M too large", Config{M: MaxM + 1}},
		{"efConstruction too large", Config{EfConstruction: MaxEfConstruction + 1}},
		{"Mmax0 below M", Config{M: 8, Mmax0: 4}},
		{"Mmax0 above limit", Config{M: 8, Mmax0: 33}},
		{"unknown metric", Config{Metric: "manhattan"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(tc.cfg); err == nil {
				t.Fatalf("New(%+v) = nil error, want validation error", tc.cfg)
			}
		})
	}
}

func TestNewResolvesDefaults(t *testing.T) {
	idx, err := New(Config{})
	if err != nil {
		t.Fatalf("New(default) = %v", err)
	}
	cfg := idx.Config()
	if cfg.M != 16 || cfg.Mmax0 != 32 || cfg.EfConstruction != 200 || cfg.Metric != MetricL2 {
		t.Fatalf("resolved config = %+v", cfg)
	}
}

func TestInsertRejectsZeroDimension(t *testing.T) {
	idx := MustNew(defaultConfig())
	if _, err := idx.Insert(nil, 0); err == nil {
		t.Fatal("Insert(nil) = nil error, want error")
	}
	if got := idx.Dim(); got != 0 {
		t.Fatalf("Dim = %d, want 0", got)
	}
	if _, err := idx.Insert([]float32{1, 2}, 1); err != nil {
		t.Fatalf("Insert after rejected vector = %v", err)
	}
	if got := idx.Dim(); got != 2 {
		t.Fatalf("Dim = %d, want 2", got)
	}
}

func TestInsertClonesVector(t *testing.T) {
	idx := MustNew(defaultConfig())
	v := []float32{1, 2, 3}
	if _, err := idx.Insert(v, 0); err != nil {
		t.Fatalf("Insert = %v", err)
	}
	v[0], v[1], v[2] = -100, -100, -100

	res, err := idx.Search([]float32{1, 2, 3}, 1, 10)
	if err != nil {
		t.Fatalf("Search = %v", err)
	}
	if len(res) != 1 || res[0].Dist != 0 {
		t.Fatalf("Search after mutation = %+v, want exact match", res)
	}
}

func TestSearchRaisesEfFloor(t *testing.T) {
	idx := MustNew(defaultConfig())
	r := rand.New(rand.NewSource(5))
	for i := 0; i < 100; i++ {
		if _, err := idx.Insert(randVec(r, 4), uint32(i)); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}
	res, err := idx.Search(randVec(r, 4), 5, 0)
	if err != nil {
		t.Fatalf("Search = %v", err)
	}
	if len(res) != 5 {
		t.Fatalf("len(results) = %d, want 5", len(res))
	}
}

func TestAlgorithm4Options(t *testing.T) {
	for _, tc := range []struct {
		name     string
		extend   bool
		keepPrun bool
	}{
		{"extend candidates", true, false},
		{"keep pruned", false, true},
		{"both", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := defaultConfig()
			cfg.M = 6
			cfg.Seed = 11
			cfg.ExtendCandidates = tc.extend
			cfg.KeepPruned = tc.keepPrun
			idx := MustNew(cfg)

			r := rand.New(rand.NewSource(17))
			for i := 0; i < 300; i++ {
				if _, err := idx.Insert(randVec(r, 8), uint32(i)); err != nil {
					t.Fatalf("insert %d: %v", i, err)
				}
			}

			idx.mu.RLock()
			defer idx.mu.RUnlock()
			resolved := idx.config
			for _, nd := range idx.nodes {
				for level := 0; level < len(nd.neighbors); level++ {
					seen := make(map[uint32]bool, len(nd.neighbors[level]))
					for _, nid := range nd.neighbors[level] {
						if nid == nd.id {
							t.Fatalf("node %d links to itself at level %d", nd.id, level)
						}
						if seen[nid] {
							t.Fatalf("node %d has duplicate neighbor %d at level %d", nd.id, nid, level)
						}
						seen[nid] = true
					}
					if limit := resolved.Mmax(level); len(nd.neighbors[level]) > limit {
						t.Fatalf("node %d degree %d at level %d exceeds %d",
							nd.id, len(nd.neighbors[level]), level, limit)
					}
				}
			}
		})
	}
}
