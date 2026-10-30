package benchmark

import (
	"math"
	"strings"
	"testing"

	"hnsw-coursework/internal/dataset"
	"hnsw-coursework/internal/hnsw"
)

const dim = 8

func mustGenerate(t *testing.T, n, dim int, seed int64) [][]float32 {
	t.Helper()
	vectors, err := dataset.Generate(n, dim, "uniform", seed)
	if err != nil {
		t.Fatalf("dataset.Generate: %v", err)
	}
	return vectors
}

func TestRunGridSmall(t *testing.T) {
	const (
		n       = 300
		queries = 10
		k       = 5
	)
	vectors := mustGenerate(t, n, dim, 1)
	queryVectors := mustGenerate(t, queries, dim, 2)

	cfg := GridConfig{
		Vectors:        vectors,
		Queries:        queryVectors,
		M:              []int{4, 8},
		EfConstruction: []int{16, 32},
		EfSearch:       []int{8, 16},
		Metric:         []hnsw.Metric{hnsw.MetricL2},
		UseHeuristic:   []bool{true, false},
		K:              k,
		Seed:           7,
	}

	const wantRows = 2 * 2 * 2 * 1 * 2
	var progressCalls, lastDone int
	cfg.Progress = func(done, total int) {
		progressCalls++
		if total != wantRows {
			t.Fatalf("progress total = %d, want %d", total, wantRows)
		}
		if done < lastDone {
			t.Fatalf("progress went backwards: %d after %d", done, lastDone)
		}
		lastDone = done
	}

	results, err := RunGrid(cfg)
	if err != nil {
		t.Fatalf("RunGrid: %v", err)
	}
	if len(results) != wantRows {
		t.Fatalf("len(results) = %d, want %d", len(results), wantRows)
	}
	if progressCalls != wantRows+1 {
		t.Fatalf("progress calls = %d, want %d", progressCalls, wantRows+1)
	}
	if lastDone != wantRows {
		t.Fatalf("final progress = %d, want %d", lastDone, wantRows)
	}

	seen := make(map[[2]any]bool)
	for i, r := range results {
		if r.K != k {
			t.Errorf("row %d: K = %d, want %d", i, r.K, k)
		}
		if r.Metric != hnsw.MetricL2 {
			t.Errorf("row %d: metric = %q, want l2", i, r.Metric)
		}
		if r.Recall < 0 || r.Recall > 1 {
			t.Errorf("row %d: recall = %v, want [0, 1]", i, r.Recall)
		}
		if r.QPS <= 0 {
			t.Errorf("row %d: QPS = %v, want > 0", i, r.QPS)
		}
		if r.BruteforceQPS <= 0 {
			t.Errorf("row %d: bruteforce QPS = %v, want > 0", i, r.BruteforceQPS)
		}
		if r.Speedup <= 0 {
			t.Errorf("row %d: speedup = %v, want > 0", i, r.Speedup)
		}
		if r.BuildTimeMs <= 0 {
			t.Errorf("row %d: build time = %v, want > 0", i, r.BuildTimeMs)
		}
		if !(r.LatencyP50Ms <= r.LatencyP95Ms && r.LatencyP95Ms <= r.LatencyP99Ms) {
			t.Errorf("row %d: latencies out of order: p50=%v p95=%v p99=%v",
				i, r.LatencyP50Ms, r.LatencyP95Ms, r.LatencyP99Ms)
		}
		for _, v := range []float64{r.Recall, r.QPS, r.BruteforceQPS, r.Speedup, r.BuildTimeMs,
			r.LatencyP50Ms, r.LatencyP95Ms, r.LatencyP99Ms} {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				t.Errorf("row %d: non-finite metric %v", i, v)
			}
		}
		seen[[2]any{r.M, r.UseHeuristic}] = true
	}
	if len(seen) != 4 {
		t.Fatalf("distinct M/heuristic groups = %d, want 4", len(seen))
	}
}

func TestRunGridDefaults(t *testing.T) {
	vectors := mustGenerate(t, 120, dim, 3)
	queryVectors := mustGenerate(t, 3, dim, 4)

	results, err := RunGrid(GridConfig{
		Vectors: vectors,
		Queries: queryVectors,
		K:       3,
	})
	if err != nil {
		t.Fatalf("RunGrid: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	r := results[0]
	if r.M != 16 || r.EfConstruction != 200 || r.EfSearch != 100 {
		t.Fatalf("defaults = M %d, efC %d, efSearch %d; want 16, 200, 100",
			r.M, r.EfConstruction, r.EfSearch)
	}
	if r.Metric != hnsw.MetricL2 || !r.UseHeuristic {
		t.Fatalf("defaults = metric %q, heuristic %t; want l2, true", r.Metric, r.UseHeuristic)
	}
	if r.K != 3 {
		t.Fatalf("K = %d, want 3", r.K)
	}
}

func TestRunGridDeterministic(t *testing.T) {
	vectors := mustGenerate(t, 200, dim, 5)
	queryVectors := mustGenerate(t, 6, dim, 6)
	cfg := GridConfig{
		Vectors:        vectors,
		Queries:        queryVectors,
		M:              []int{4},
		EfConstruction: []int{16},
		EfSearch:       []int{10, 20},
		Metric:         []hnsw.Metric{hnsw.MetricL2, hnsw.MetricCosine},
		UseHeuristic:   []bool{true, false},
		K:              4,
		Seed:           99,
	}

	first, err := RunGrid(cfg)
	if err != nil {
		t.Fatalf("first RunGrid: %v", err)
	}
	second, err := RunGrid(cfg)
	if err != nil {
		t.Fatalf("second RunGrid: %v", err)
	}
	if len(first) != len(second) {
		t.Fatalf("row counts differ: %d vs %d", len(first), len(second))
	}
	for i := range first {
		a, b := first[i], second[i]
		if a.M != b.M || a.EfConstruction != b.EfConstruction || a.EfSearch != b.EfSearch ||
			a.Metric != b.Metric || a.UseHeuristic != b.UseHeuristic {
			t.Fatalf("row %d: parameters differ: %+v vs %+v", i, a, b)
		}
		if a.Recall != b.Recall {
			t.Fatalf("row %d: recall not deterministic: %v vs %v", i, a.Recall, b.Recall)
		}
	}
}

func TestRunGridValidation(t *testing.T) {
	vectors := mustGenerate(t, 100, dim, 7)
	queryVectors := mustGenerate(t, 3, dim, 8)

	base := func() GridConfig {
		return GridConfig{
			Vectors: vectors,
			Queries: queryVectors,
			K:       3,
		}
	}

	tests := []struct {
		name    string
		mutate  func(*GridConfig)
		wantErr string
	}{
		{"no vectors", func(c *GridConfig) { c.Vectors = nil }, "no vectors"},
		{"no queries", func(c *GridConfig) { c.Queries = nil }, "no queries"},
		{"k exceeds n", func(c *GridConfig) { c.K = 101 }, "exceeds number of vectors"},
		{"bad M", func(c *GridConfig) { c.M = []int{0} }, "M must be positive"},
		{"bad efConstruction", func(c *GridConfig) { c.EfConstruction = []int{0} }, "efConstruction must be positive"},
		{"bad efSearch", func(c *GridConfig) { c.EfSearch = []int{0} }, "efSearch must be positive"},
		{"unknown metric", func(c *GridConfig) { c.Metric = []hnsw.Metric{"manhattan"} }, "unknown metric"},
		{"query dim mismatch", func(c *GridConfig) {
			c.Queries = mustGenerate(t, 3, dim+1, 9)
		}, "query dimension"},
		{"ragged vectors", func(c *GridConfig) {
			c.Vectors = [][]float32{{1, 2}, {1, 2, 3}}
		}, "dimension"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base()
			tc.mutate(&cfg)
			_, err := RunGrid(cfg)
			if err == nil {
				t.Fatalf("RunGrid: want error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %q, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}
