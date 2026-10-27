package hnsw

import (
	"fmt"
	"math/rand"
	"testing"
)

func randVec(r *rand.Rand, dim int) []float32 {
	v := make([]float32, dim)
	for i := range v {
		v[i] = float32(r.Float64())
	}
	return v
}

func TestInsertAndSearchBasic(t *testing.T) {
	cfg := defaultConfig()
	cfg.Seed = 42
	cfg.M = 8
	cfg.EfConstruction = 100
	idx := New(cfg)

	dim := 10
	r := rand.New(rand.NewSource(1))
	n := 100
	vecs := make([][]float32, n)
	for i := 0; i < n; i++ {
		vecs[i] = randVec(r, dim)
		if _, err := idx.Insert(vecs[i], uint32(i)); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}

	if got := idx.Len(); got != n {
		t.Fatalf("Len = %d, want %d", got, n)
	}

	// Search for each vector: it should be its own nearest neighbor (distance 0).
	for i := 0; i < n; i++ {
		res, err := idx.Search(vecs[i], 1, 100)
		if err != nil {
			t.Fatalf("search %d: %v", i, err)
		}
		if len(res) == 0 {
			t.Fatalf("search %d: no results", i)
		}
		if res[0].id != uint32(i) {
			t.Fatalf("search %d: nearest = %d, want %d", i, res[0].id, i)
		}
	}
}

func TestSearchKResults(t *testing.T) {
	cfg := defaultConfig()
	cfg.Seed = 7
	idx := New(cfg)

	dim := 8
	r := rand.New(rand.NewSource(3))
	n := 50
	for i := 0; i < n; i++ {
		if _, err := idx.Insert(randVec(r, dim), uint32(i)); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}

	q := randVec(r, dim)
	k := 10
	res, err := idx.Search(q, k, 100)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(res) != k {
		t.Fatalf("len = %d, want %d", len(res), k)
	}
	// Distances should be non-decreasing.
	for i := 1; i < len(res); i++ {
		if res[i].dist < res[i-1].dist {
			t.Fatalf("results not sorted: %v > %v at %d", res[i-1].dist, res[i].dist, i)
		}
	}
}

func TestDeterminism(t *testing.T) {
	build := func(seed int64) *Index {
		cfg := defaultConfig()
		cfg.Seed = seed
		cfg.M = 12
		cfg.EfConstruction = 150
		idx := New(cfg)
		r := rand.New(rand.NewSource(99))
		dim := 12
		n := 200
		for i := 0; i < n; i++ {
			if _, err := idx.Insert(randVec(r, dim), uint32(i)); err != nil {
				t.Fatalf("insert: %v", err)
			}
		}
		return idx
	}

	idx1 := build(123)
	idx2 := build(123)

	s1 := idx1.Stats()
	s2 := idx2.Stats()
	if s1.MaxLevel != s2.MaxLevel {
		t.Fatalf("maxLevel differs: %d vs %d", s1.MaxLevel, s2.MaxLevel)
	}
	if s1.Nodes != s2.Nodes {
		t.Fatalf("nodes differs: %d vs %d", s1.Nodes, s2.Nodes)
	}
	for l := 0; l <= s1.MaxLevel; l++ {
		if s1.NodesPerLevel[l] != s2.NodesPerLevel[l] {
			t.Fatalf("nodesPerLevel[%d] differs: %d vs %d", l, s1.NodesPerLevel[l], s2.NodesPerLevel[l])
		}
	}
}

func TestGraphInvariants(t *testing.T) {
	cfg := defaultConfig()
	cfg.Seed = 55
	cfg.M = 8
	cfg.Mmax0 = 16
	cfg.EfConstruction = 200
	idx := New(cfg)

	r := rand.New(rand.NewSource(11))
	dim := 10
	n := 500
	for i := 0; i < n; i++ {
		if _, err := idx.Insert(randVec(r, dim), uint32(i)); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}

	stats := idx.Stats()

	// No self-loops.
	for _, nd := range idx.nodes {
		for l := 0; l <= nd.maxLevel; l++ {
			for _, nb := range nd.neighbors[l] {
				if nb == nd.id {
					t.Fatalf("self-loop at node %d level %d", nd.id, l)
				}
			}
		}
	}

	// Degree <= Mmax at each level.
	for _, nd := range idx.nodes {
		for l := 0; l <= nd.maxLevel; l++ {
			mmax := idx.config.Mmax(l)
			if deg := len(nd.neighbors[l]); deg > mmax {
				t.Fatalf("node %d level %d degree %d > Mmax %d", nd.id, l, deg, mmax)
			}
		}
	}

	// Bidirectional links: if a->b at level l, then b->a at level l.
	for _, nd := range idx.nodes {
		for l := 0; l <= nd.maxLevel; l++ {
			for _, nb := range nd.neighbors[l] {
				found := false
				for _, nb2 := range idx.nodes[nb].neighbors[l] {
					if nb2 == nd.id {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("missing back-link: %d->%d at level %d", nd.id, nb, l)
				}
			}
		}
	}

	// Entry point exists at maxLevel.
	if stats.MaxLevel >= 0 {
		ep := idx.nodes[stats.EntryPoint]
		if ep.maxLevel < stats.MaxLevel {
			t.Fatalf("entry point %d maxLevel %d < index maxLevel %d",
				stats.EntryPoint, ep.maxLevel, stats.MaxLevel)
		}
		if !containsUint32(ep.neighbors[stats.MaxLevel], stats.EntryPoint) && stats.MaxLevel > 0 {
			// Entry point at maxLevel may or may not have neighbors at that level,
			// but it must exist at that level (it does since maxLevel <= ep.maxLevel).
			_ = ep
		}
	}
}

func TestSearchWithTrace(t *testing.T) {
	cfg := defaultConfig()
	cfg.Seed = 3
	idx := New(cfg)

	r := rand.New(rand.NewSource(5))
	dim := 6
	n := 80
	for i := 0; i < n; i++ {
		if _, err := idx.Insert(randVec(r, dim), uint32(i)); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}

	q := randVec(r, dim)
	res, trace, err := idx.SearchWithTrace(q, 10, 100)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(res) == 0 {
		t.Fatal("no results")
	}
	if trace == nil {
		t.Fatal("trace is nil")
	}
	if len(trace.Layers) == 0 {
		t.Fatal("trace has no layers")
	}
	// Layer 0 should be present.
	hasLayer0 := false
	for _, lt := range trace.Layers {
		if lt.Level == 0 {
			hasLayer0 = true
			if lt.Closest == 0 && len(res) > 0 {
				// Closest should be a valid node id.
				if int(lt.Closest) >= len(idx.nodes) {
					t.Fatalf("trace closest %d out of range", lt.Closest)
				}
			}
		}
	}
	if !hasLayer0 {
		t.Fatal("trace missing layer 0")
	}
	// Every visited id must be valid.
	for _, lt := range trace.Layers {
		for _, id := range lt.Visited {
			if int(id) >= len(idx.nodes) {
				t.Fatalf("visited id %d out of range", id)
			}
		}
	}
}

func TestQualityRecall(t *testing.T) {
	// Quality test: recall@10 >= 0.95 on 10k random vectors with M=16, efC=200, efSearch=100.
	cfg := defaultConfig()
	cfg.Seed = 2026
	cfg.M = 16
	cfg.Mmax0 = 32
	cfg.EfConstruction = 200
	idx := New(cfg)

	dim := 16
	n := 10000
	r := rand.New(rand.NewSource(42))
	dataset := make([][]float32, n)
	for i := 0; i < n; i++ {
		dataset[i] = randVec(r, dim)
		if _, err := idx.Insert(dataset[i], uint32(i)); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}

	// Brute-force ground truth for 100 random queries.
	qCount := 100
	recallSum := 0.0
	for q := 0; q < qCount; q++ {
		query := randVec(r, dim)

		// Brute force: top 10 by squared L2.
		type cand struct {
			id   uint32
			dist float32
		}
		gt := make([]cand, n)
		for i := 0; i < n; i++ {
			gt[i] = cand{id: uint32(i), dist: SquaredL2(query, dataset[i])}
		}
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				if gt[j].dist < gt[i].dist {
					gt[i], gt[j] = gt[j], gt[i]
				}
			}
		}
		gtSet := make(map[uint32]bool)
		for i := 0; i < 10; i++ {
			gtSet[gt[i].id] = true
		}

		// HNSW search.
		res, err := idx.Search(query, 10, 100)
		if err != nil {
			t.Fatalf("search %d: %v", q, err)
		}
		found := 0
		for _, it := range res {
			if gtSet[it.id] {
				found++
			}
		}
		recallSum += float64(found) / 10.0
	}
	recall := recallSum / float64(qCount)
	fmt.Printf("recall@10 = %.4f (M=16, efC=200, efSearch=100, n=%d)\n", recall, n)
	if recall < 0.95 {
		t.Fatalf("recall@10 = %.4f, want >= 0.95", recall)
	}
}

func TestWrongDimensionRejected(t *testing.T) {
	cfg := defaultConfig()
	idx := New(cfg)

	if _, err := idx.Insert([]float32{1, 2, 3}, 0); err != nil {
		t.Fatalf("first insert should set dim: %v", err)
	}
	if _, err := idx.Insert([]float32{1, 2, 3, 4}, 1); err == nil {
		t.Fatal("wrong dimension should be rejected")
	}
}

func TestEmptyIndexSearch(t *testing.T) {
	idx := New(defaultConfig())
	res, err := idx.Search([]float32{1, 2, 3}, 5, 10)
	if err != nil {
		t.Fatalf("search on empty index: %v", err)
	}
	if len(res) != 0 {
		t.Fatalf("empty index search should return no results, got %d", len(res))
	}
}
