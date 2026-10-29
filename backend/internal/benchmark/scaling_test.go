package benchmark

import (
	"math"
	"strings"
	"testing"

	"hnsw-coursework/internal/hnsw"
)

func TestDefaultScalingConfig(t *testing.T) {
	cfg := DefaultScalingConfig()
	wantN := []int{1000, 10000, 100000, 1000000}
	if len(cfg.N) != len(wantN) {
		t.Fatalf("N = %v, want %v", cfg.N, wantN)
	}
	for i := range wantN {
		if cfg.N[i] != wantN[i] {
			t.Fatalf("N[%d] = %d, want %d", i, cfg.N[i], wantN[i])
		}
	}
	if cfg.M != 16 || cfg.EfConstruction != 200 || cfg.EfSearch != 100 {
		t.Fatalf("defaults = M %d, efC %d, efSearch %d; want 16, 200, 100",
			cfg.M, cfg.EfConstruction, cfg.EfSearch)
	}
	if !cfg.UseHeuristic || cfg.Metric != hnsw.MetricL2 || cfg.Distribution != "uniform" {
		t.Fatalf("defaults = heuristic %t, metric %q, distribution %q; want true, l2, uniform",
			cfg.UseHeuristic, cfg.Metric, cfg.Distribution)
	}
}

func TestRunScaling(t *testing.T) {
	cfg := ScalingConfig{
		N:              []int{300, 1200},
		Dim:            dim,
		QueryCount:     5,
		K:              3,
		EfSearch:       20,
		M:              8,
		EfConstruction: 32,
		Metric:         hnsw.MetricL2,
		UseHeuristic:   true,
		Seed:           11,
		Distribution:   "uniform",
	}

	var progressCalls, lastDone int
	cfg.Progress = func(done, total int) {
		progressCalls++
		if total != len(cfg.N) {
			t.Fatalf("progress total = %d, want %d", total, len(cfg.N))
		}
		if done < lastDone {
			t.Fatalf("progress went backwards: %d after %d", done, lastDone)
		}
		lastDone = done
	}

	results, err := RunScaling(cfg)
	if err != nil {
		t.Fatalf("RunScaling: %v", err)
	}
	if len(results) != len(cfg.N) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(cfg.N))
	}
	if progressCalls != len(cfg.N)+1 {
		t.Fatalf("progress calls = %d, want %d", progressCalls, len(cfg.N)+1)
	}

	for i, r := range results {
		if r.N != cfg.N[i] {
			t.Errorf("row %d: N = %d, want %d", i, r.N, cfg.N[i])
		}
		if r.Dim != cfg.Dim || r.M != cfg.M || r.EfConstruction != cfg.EfConstruction ||
			r.EfSearch != cfg.EfSearch || r.K != cfg.K || r.Queries != cfg.QueryCount {
			t.Errorf("row %d: parameters = %+v, want dim %d, M %d, efC %d, efSearch %d, k %d, queries %d",
				i, r, cfg.Dim, cfg.M, cfg.EfConstruction, cfg.EfSearch, cfg.K, cfg.QueryCount)
		}
		if r.Metric != cfg.Metric || !r.UseHeuristic {
			t.Errorf("row %d: metric = %q, heuristic = %t; want l2, true", i, r.Metric, r.UseHeuristic)
		}
		if r.Recall < 0 || r.Recall > 1 {
			t.Errorf("row %d: recall = %v, want [0, 1]", i, r.Recall)
		}
		if r.QPS <= 0 || r.BruteforceQPS <= 0 || r.Speedup <= 0 {
			t.Errorf("row %d: non-positive qps/speedup: %+v", i, r)
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
	}
}

func TestRunScalingValidation(t *testing.T) {
	base := func() ScalingConfig {
		return ScalingConfig{
			N:              []int{50, 100},
			Dim:            dim,
			QueryCount:     3,
			K:              2,
			EfSearch:       10,
			M:              4,
			EfConstruction: 16,
			Metric:         hnsw.MetricL2,
			UseHeuristic:   true,
			Seed:           1,
			Distribution:   "uniform",
		}
	}

	tests := []struct {
		name    string
		mutate  func(*ScalingConfig)
		wantErr string
	}{
		{"zero N value", func(c *ScalingConfig) { c.N = []int{0} }, "must be positive"},
		{"k exceeds N", func(c *ScalingConfig) { c.N = []int{2}; c.K = 5 }, "exceeds N"},
		{"unknown metric", func(c *ScalingConfig) { c.Metric = "hamming" }, "unknown metric"},
		{"unknown distribution", func(c *ScalingConfig) { c.Distribution = "zipf" }, "unknown distribution"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base()
			tc.mutate(&cfg)
			_, err := RunScaling(cfg)
			if err == nil {
				t.Fatalf("RunScaling: want error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %q, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}
