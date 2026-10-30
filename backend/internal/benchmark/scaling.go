package benchmark

import (
	"encoding/json"
	"fmt"
	"time"

	"hnsw-coursework/internal/bruteforce"
	"hnsw-coursework/internal/dataset"
	"hnsw-coursework/internal/hnsw"
)

type ScalingConfig struct {
	N                 []int
	Dim               int
	QueryCount        int
	K                 int
	EfSearch          int
	M                 int
	EfConstruction    int
	Metric            hnsw.Metric
	UseHeuristic      bool
	Seed              int64
	Distribution      string
	BruteforceWorkers int
	Progress          func(done, total int)
}

type ScalingResult struct {
	N              int         `json:"n"`
	Dim            int         `json:"dim"`
	M              int         `json:"M"`
	EfConstruction int         `json:"efConstruction"`
	EfSearch       int         `json:"efSearch"`
	Metric         hnsw.Metric `json:"metric"`
	UseHeuristic   bool        `json:"heuristic"`
	K              int         `json:"k"`
	Queries        int         `json:"queries"`
	Recall         float64     `json:"recall"`
	QPS            float64     `json:"qps"`
	LatencyP50Ms   float64     `json:"latencyP50Ms"`
	LatencyP95Ms   float64     `json:"latencyP95Ms"`
	LatencyP99Ms   float64     `json:"latencyP99Ms"`
	BuildTimeMs    float64     `json:"buildTimeMs"`
	BruteforceQPS  float64     `json:"bruteforceQps"`
	Speedup        float64     `json:"speedup"`
}

func (r ScalingResult) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"n":              r.N,
		"dim":            r.Dim,
		"M":              r.M,
		"efConstruction": r.EfConstruction,
		"efSearch":       r.EfSearch,
		"metric":         r.Metric,
		"heuristic":      r.UseHeuristic,
		"k":              r.K,
		"queries":        r.Queries,
		recallKey(r.K):   r.Recall,
		"qps":            r.QPS,
		"latencyP50Ms":   r.LatencyP50Ms,
		"latencyP95Ms":   r.LatencyP95Ms,
		"latencyP99Ms":   r.LatencyP99Ms,
		"buildTimeMs":    r.BuildTimeMs,
		"bruteforceQps":  r.BruteforceQPS,
		"speedup":        r.Speedup,
	})
}

func DefaultScalingConfig() ScalingConfig {
	return ScalingConfig{
		N:              []int{1_000, 10_000, 100_000, 1_000_000},
		Dim:            16,
		QueryCount:     100,
		K:              10,
		EfSearch:       100,
		M:              16,
		EfConstruction: 200,
		Metric:         hnsw.MetricL2,
		UseHeuristic:   true,
		Seed:           42,
		Distribution:   "uniform",
	}
}

func (c ScalingConfig) withDefaults() ScalingConfig {
	def := DefaultScalingConfig()
	if len(c.N) == 0 {
		c.N = def.N
	}
	if c.Dim <= 0 {
		c.Dim = def.Dim
	}
	if c.QueryCount <= 0 {
		c.QueryCount = def.QueryCount
	}
	if c.K <= 0 {
		c.K = def.K
	}
	if c.EfSearch <= 0 {
		c.EfSearch = def.EfSearch
	}
	if c.M <= 0 {
		c.M = def.M
	}
	if c.EfConstruction <= 0 {
		c.EfConstruction = def.EfConstruction
	}
	if c.Metric == "" {
		c.Metric = def.Metric
	}
	if c.Distribution == "" {
		c.Distribution = def.Distribution
	}
	return c
}

func (c ScalingConfig) validate() error {
	if len(c.N) == 0 {
		return fmt.Errorf("scaling: no N values")
	}
	for i, n := range c.N {
		if n <= 0 {
			return fmt.Errorf("scaling: N[%d] must be positive, got %d", i, n)
		}
		if c.K > n {
			return fmt.Errorf("scaling: k=%d exceeds N=%d", c.K, n)
		}
	}
	if c.Dim <= 0 {
		return fmt.Errorf("scaling: dim must be positive, got %d", c.Dim)
	}
	if c.QueryCount <= 0 {
		return fmt.Errorf("scaling: query count must be positive, got %d", c.QueryCount)
	}
	if c.K <= 0 {
		return fmt.Errorf("scaling: k must be positive, got %d", c.K)
	}
	if c.EfSearch <= 0 {
		return fmt.Errorf("scaling: efSearch must be positive, got %d", c.EfSearch)
	}
	if c.M <= 0 {
		return fmt.Errorf("scaling: M must be positive, got %d", c.M)
	}
	if c.EfConstruction <= 0 {
		return fmt.Errorf("scaling: efConstruction must be positive, got %d", c.EfConstruction)
	}
	if !validMetric(c.Metric) {
		return fmt.Errorf("scaling: unknown metric %q", c.Metric)
	}
	return nil
}

func RunScaling(cfg ScalingConfig) ([]ScalingResult, error) {
	cfg = cfg.withDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	queries, err := dataset.Generate(cfg.QueryCount, cfg.Dim, cfg.Distribution, cfg.Seed+1)
	if err != nil {
		return nil, fmt.Errorf("scaling: generate queries: %w", err)
	}

	results := make([]ScalingResult, 0, len(cfg.N))
	if cfg.Progress != nil {
		cfg.Progress(0, len(cfg.N))
	}

	for i, n := range cfg.N {
		vectors, err := dataset.Generate(n, cfg.Dim, cfg.Distribution, cfg.Seed)
		if err != nil {
			return nil, fmt.Errorf("scaling: generate n=%d: %w", n, err)
		}

		idx := hnsw.New(hnsw.Config{
			M:              cfg.M,
			EfConstruction: cfg.EfConstruction,
			Metric:         cfg.Metric,
			Seed:           cfg.Seed,
			UseHeuristic:   cfg.UseHeuristic,
		})
		buildTime, err := MeasureBuildTime(func() error {
			for j, v := range vectors {
				if _, err := idx.Insert(v, uint32(j)); err != nil {
					return fmt.Errorf("insert vector %d: %w", j, err)
				}
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("scaling n=%d: build: %w", n, err)
		}

		bf, err := bruteforce.NewWithWorkers(vectors, hnsw.Distance(cfg.Metric), cfg.BruteforceWorkers)
		if err != nil {
			return nil, fmt.Errorf("scaling n=%d: bruteforce: %w", n, err)
		}
		exact, bfQPS, err := measureExact(bf, queries, cfg.K)
		if err != nil {
			return nil, fmt.Errorf("scaling n=%d: bruteforce search: %w", n, err)
		}

		lat, recall, err := measureSearch(idx, queries, exact, cfg.K, cfg.EfSearch)
		if err != nil {
			return nil, fmt.Errorf("scaling n=%d: hnsw search: %w", n, err)
		}

		speedup := 0.0
		if bfQPS > 0 {
			speedup = lat.QPS / bfQPS
		}
		results = append(results, ScalingResult{
			N:              n,
			Dim:            cfg.Dim,
			M:              cfg.M,
			EfConstruction: cfg.EfConstruction,
			EfSearch:       cfg.EfSearch,
			Metric:         cfg.Metric,
			UseHeuristic:   cfg.UseHeuristic,
			K:              cfg.K,
			Queries:        cfg.QueryCount,
			Recall:         recall,
			QPS:            lat.QPS,
			LatencyP50Ms:   lat.P50Ms,
			LatencyP95Ms:   lat.P95Ms,
			LatencyP99Ms:   lat.P99Ms,
			BuildTimeMs:    float64(buildTime) / float64(time.Millisecond),
			BruteforceQPS:  bfQPS,
			Speedup:        speedup,
		})

		if cfg.Progress != nil {
			cfg.Progress(i+1, len(cfg.N))
		}
	}

	return results, nil
}
