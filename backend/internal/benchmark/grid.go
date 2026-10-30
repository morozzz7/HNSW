package benchmark

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"hnsw-coursework/internal/bruteforce"
	"hnsw-coursework/internal/dataset"
	"hnsw-coursework/internal/hnsw"
)

type GridConfig struct {
	Vectors           [][]float32
	Queries           [][]float32
	M                 []int
	EfConstruction    []int
	EfSearch          []int
	Metric            []hnsw.Metric
	UseHeuristic      []bool
	K                 int
	Seed              int64
	BruteforceWorkers int
	Progress          func(done, total int)
}

type Result struct {
	M              int         `json:"M"`
	EfConstruction int         `json:"efConstruction"`
	EfSearch       int         `json:"efSearch"`
	Metric         hnsw.Metric `json:"metric"`
	UseHeuristic   bool        `json:"heuristic"`
	K              int         `json:"k"`
	Recall         float64     `json:"recall"`
	QPS            float64     `json:"qps"`
	LatencyP50Ms   float64     `json:"latencyP50Ms"`
	LatencyP95Ms   float64     `json:"latencyP95Ms"`
	LatencyP99Ms   float64     `json:"latencyP99Ms"`
	BuildTimeMs    float64     `json:"buildTimeMs"`
	BruteforceQPS  float64     `json:"bruteforceQps"`
	Speedup        float64     `json:"speedup"`
}

func (r Result) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"M":              r.M,
		"efConstruction": r.EfConstruction,
		"efSearch":       r.EfSearch,
		"metric":         r.Metric,
		"heuristic":      r.UseHeuristic,
		"k":              r.K,
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

func recallKey(k int) string {
	if k <= 0 {
		return "recall"
	}
	return "recall@" + strconv.Itoa(k)
}

type exactTable struct {
	results [][]bruteforce.Result
	qps     float64
}

func (c GridConfig) withDefaults() GridConfig {
	if len(c.M) == 0 {
		c.M = []int{16}
	}
	if len(c.EfConstruction) == 0 {
		c.EfConstruction = []int{200}
	}
	if len(c.EfSearch) == 0 {
		c.EfSearch = []int{100}
	}
	if len(c.Metric) == 0 {
		c.Metric = []hnsw.Metric{hnsw.MetricL2}
	}
	if len(c.UseHeuristic) == 0 {
		c.UseHeuristic = []bool{true}
	}
	if c.K <= 0 {
		c.K = 10
	}
	return c
}

func (c GridConfig) validate() error {
	if len(c.Vectors) == 0 {
		return fmt.Errorf("grid: no vectors")
	}
	if len(c.Queries) == 0 {
		return fmt.Errorf("grid: no queries")
	}
	if err := dataset.Validate(c.Vectors); err != nil {
		return fmt.Errorf("grid: vectors: %w", err)
	}
	if err := dataset.Validate(c.Queries); err != nil {
		return fmt.Errorf("grid: queries: %w", err)
	}
	dim := len(c.Vectors[0])
	if len(c.Queries[0]) != dim {
		return fmt.Errorf("grid: query dimension %d does not match vector dimension %d", len(c.Queries[0]), dim)
	}
	if c.K <= 0 {
		return fmt.Errorf("grid: k must be positive, got %d", c.K)
	}
	if c.K > len(c.Vectors) {
		return fmt.Errorf("grid: k=%d exceeds number of vectors %d", c.K, len(c.Vectors))
	}
	for _, m := range c.M {
		if m <= 0 {
			return fmt.Errorf("grid: M must be positive, got %d", m)
		}
	}
	for _, efC := range c.EfConstruction {
		if efC <= 0 {
			return fmt.Errorf("grid: efConstruction must be positive, got %d", efC)
		}
	}
	for _, efS := range c.EfSearch {
		if efS <= 0 {
			return fmt.Errorf("grid: efSearch must be positive, got %d", efS)
		}
	}
	for _, metric := range c.Metric {
		if !validMetric(metric) {
			return fmt.Errorf("grid: unknown metric %q", metric)
		}
	}
	return nil
}

func validMetric(m hnsw.Metric) bool {
	switch m {
	case hnsw.MetricL2, hnsw.MetricCosine, hnsw.MetricInnerProduct:
		return true
	default:
		return false
	}
}

func RunGrid(cfg GridConfig) ([]Result, error) {
	cfg = cfg.withDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	exact, err := buildExactTables(cfg)
	if err != nil {
		return nil, err
	}

	total := len(cfg.M) * len(cfg.EfConstruction) * len(cfg.EfSearch) *
		len(cfg.Metric) * len(cfg.UseHeuristic)
	results := make([]Result, 0, total)
	if cfg.Progress != nil {
		cfg.Progress(0, total)
	}
	done := 0

	for _, m := range cfg.M {
		for _, efC := range cfg.EfConstruction {
			for _, metric := range cfg.Metric {
				gt := exact[metric]
				for _, heuristic := range cfg.UseHeuristic {
					idx := hnsw.New(hnsw.Config{
						M:              m,
						EfConstruction: efC,
						Metric:         metric,
						Seed:           cfg.Seed,
						UseHeuristic:   heuristic,
					})

					buildTime, err := MeasureBuildTime(func() error {
						for i, v := range cfg.Vectors {
							if _, err := idx.Insert(v, uint32(i)); err != nil {
								return fmt.Errorf("insert vector %d: %w", i, err)
							}
						}
						return nil
					})
					if err != nil {
						return nil, fmt.Errorf("grid M=%d efConstruction=%d metric=%s heuristic=%t: build: %w",
							m, efC, metric, heuristic, err)
					}
					buildMs := float64(buildTime) / float64(time.Millisecond)

					for _, efS := range cfg.EfSearch {
						lat, recall, err := measureSearch(idx, cfg.Queries, gt.results, cfg.K, efS)
						if err != nil {
							return nil, fmt.Errorf("grid M=%d efConstruction=%d efSearch=%d metric=%s heuristic=%t: %w",
								m, efC, efS, metric, heuristic, err)
						}

						speedup := 0.0
						if gt.qps > 0 {
							speedup = lat.QPS / gt.qps
						}
						results = append(results, Result{
							M:              m,
							EfConstruction: efC,
							EfSearch:       efS,
							Metric:         metric,
							UseHeuristic:   heuristic,
							K:              cfg.K,
							Recall:         recall,
							QPS:            lat.QPS,
							LatencyP50Ms:   lat.P50Ms,
							LatencyP95Ms:   lat.P95Ms,
							LatencyP99Ms:   lat.P99Ms,
							BuildTimeMs:    buildMs,
							BruteforceQPS:  gt.qps,
							Speedup:        speedup,
						})
						done++
						if cfg.Progress != nil {
							cfg.Progress(done, total)
						}
					}
				}
			}
		}
	}

	return results, nil
}

func buildExactTables(cfg GridConfig) (map[hnsw.Metric]exactTable, error) {
	tables := make(map[hnsw.Metric]exactTable, len(cfg.Metric))
	for _, metric := range cfg.Metric {
		if _, ok := tables[metric]; ok {
			continue
		}
		bf, err := bruteforce.NewWithWorkers(cfg.Vectors, hnsw.Distance(metric), cfg.BruteforceWorkers)
		if err != nil {
			return nil, fmt.Errorf("grid: bruteforce index (%s): %w", metric, err)
		}
		exact, qps, err := measureExact(bf, cfg.Queries, cfg.K)
		if err != nil {
			return nil, fmt.Errorf("grid: bruteforce search (%s): %w", metric, err)
		}
		tables[metric] = exactTable{results: exact, qps: qps}
	}
	return tables, nil
}

const (
	minSampleDuration  = time.Millisecond
	maxCalibrationReps = 4096
)

func calibrateReps(run func() error) (int, error) {
	const trials = 5
	reps := 1
	for {
		shortest := time.Duration(1<<63 - 1)
		for t := 0; t < trials; t++ {
			start := time.Now()
			for i := 0; i < reps; i++ {
				if err := run(); err != nil {
					return 0, err
				}
			}
			if elapsed := time.Since(start); elapsed < shortest {
				shortest = elapsed
			}
		}
		if shortest >= minSampleDuration || reps >= maxCalibrationReps {
			return reps, nil
		}
		reps *= 2
	}
}

func measureExact(bf *bruteforce.Index, queries [][]float32, k int) ([][]bruteforce.Result, float64, error) {
	if len(queries) == 0 {
		return nil, 0, nil
	}

	reps, err := calibrateReps(func() error {
		_, err := bf.Search(queries[0], k)
		return err
	})
	if err != nil {
		return nil, 0, fmt.Errorf("calibrate: %w", err)
	}

	exact := make([][]bruteforce.Result, len(queries))
	samples := make([]float64, len(queries))
	for i, q := range queries {
		start := time.Now()
		var res []bruteforce.Result
		for r := 0; r < reps; r++ {
			res, err = bf.Search(q, k)
			if err != nil {
				return nil, 0, fmt.Errorf("query %d: %w", i, err)
			}
		}
		samples[i] = elapsedMs(start) / float64(reps)
		exact[i] = res
	}
	return exact, LatencyFromSamples(samples).QPS, nil
}

func measureSearch(idx *hnsw.Index, queries [][]float32, exact [][]bruteforce.Result, k, efSearch int) (Latency, float64, error) {
	if len(queries) == 0 {
		return Latency{}, 0, nil
	}

	reps, err := calibrateReps(func() error {
		_, err := idx.Search(queries[0], k, efSearch)
		return err
	})
	if err != nil {
		return Latency{}, 0, fmt.Errorf("calibrate: %w", err)
	}

	samples := make([]float64, len(queries))
	var recallSum float64
	for i, q := range queries {
		start := time.Now()
		var res []hnsw.Result
		for r := 0; r < reps; r++ {
			res, err = idx.Search(q, k, efSearch)
			if err != nil {
				return Latency{}, 0, fmt.Errorf("query %d: %w", i, err)
			}
		}
		samples[i] = elapsedMs(start) / float64(reps)

		approx := make([]bruteforce.Result, len(res))
		for j := range res {
			approx[j] = bruteforce.Result{ID: res[j].ID, Dist: res[j].Dist}
		}
		recallSum += bruteforce.Recall(exact[i], approx, k)
	}
	return LatencyFromSamples(samples), recallSum / float64(len(queries)), nil
}

func elapsedMs(start time.Time) float64 {
	return float64(time.Since(start)) / float64(time.Millisecond)
}
