package benchmark

import (
	"math"
	"math/rand"
	"testing"

	"hnsw-coursework/internal/bruteforce"
	"hnsw-coursework/internal/hnsw"
)

// TestHNSWQualityMetrics is an end-to-end check of the Stage 3 quality metrics
// (tasks 3.2 and 3.3): it builds an HNSW index, measures build time and heap
// usage, computes recall@k against the exact bruteforce search and collects
// query latency percentiles. Real numbers are printed with -v.
func TestHNSWQualityMetrics(t *testing.T) {
	const (
		n        = 3000
		dim      = 16
		k        = 10
		efSearch = 50
		queries  = 30
	)

	r := rand.New(rand.NewSource(42))
	vectors := make([][]float32, n)
	for i := range vectors {
		v := make([]float32, dim)
		for j := range v {
			v[j] = float32(r.Float64())
		}
		vectors[i] = v
	}
	queryVectors := make([][]float32, queries)
	for i := range queryVectors {
		v := make([]float32, dim)
		for j := range v {
			v[j] = float32(r.Float64())
		}
		queryVectors[i] = v
	}

	memBefore := HeapInUse()

	var idx *hnsw.Index
	buildTime, err := MeasureBuildTime(func() error {
		idx = hnsw.New(hnsw.Config{
			M:              16,
			EfConstruction: 100,
			Metric:         hnsw.MetricL2,
			Seed:           7,
			UseHeuristic:   true,
		})
		for i, v := range vectors {
			if _, err := idx.Insert(v, uint32(i)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("build index: %v", err)
	}

	memBytes := HeapInUse() - memBefore
	if memBytes < 0 {
		memBytes = 0
	}

	bf, err := bruteforce.New(vectors, hnsw.SquaredL2)
	if err != nil {
		t.Fatalf("bruteforce.New: %v", err)
	}

	exact, bfQPS, err := measureExact(bf, queryVectors, k)
	if err != nil {
		t.Fatalf("measureExact: %v", err)
	}
	lat, recall, err := measureSearch(idx, queryVectors, exact, k, efSearch)
	if err != nil {
		t.Fatalf("measureSearch: %v", err)
	}

	t.Logf("n=%d dim=%d M=16 efC=100 efSearch=%d k=%d queries=%d", n, dim, efSearch, k, queries)
	t.Logf("build time: %v; index memory: %.1f KiB", buildTime, float64(memBytes)/1024)
	t.Logf("recall@%d: %.4f; bruteforce QPS=%.0f", k, recall, bfQPS)
	t.Logf("latency ms: p50=%.3f p95=%.3f p99=%.3f; QPS=%.0f",
		lat.P50Ms, lat.P95Ms, lat.P99Ms, lat.QPS)

	if recall < 0.9 {
		t.Fatalf("recall@%d = %.4f, want >= 0.9", k, recall)
	}
	if buildTime <= 0 {
		t.Fatalf("build time = %v, want > 0", buildTime)
	}
	if memBytes <= 0 {
		t.Fatalf("index memory = %d bytes, want > 0", memBytes)
	}
	if lat.P50Ms <= 0 || lat.P99Ms < lat.P50Ms {
		t.Fatalf("inconsistent latency stats: %+v", lat)
	}
	for _, v := range []float64{lat.P50Ms, lat.P95Ms, lat.P99Ms, lat.QPS} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Fatalf("non-finite metric: %v", v)
		}
	}
}
