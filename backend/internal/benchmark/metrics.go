// Package benchmark provides quality metrics for approximate nearest neighbor
// search: query latency percentiles (p50/p95/p99), QPS, index build time and
// heap memory usage. It is the building block of the HNSW vs brute-force
// comparison (Stage 4).
package benchmark

import (
	"math"
	"runtime"
	"sort"
	"time"
)

// Latency summarizes query latency statistics in milliseconds.
type Latency struct {
	P50Ms float64
	P95Ms float64
	P99Ms float64
	QPS   float64 // queries per second, derived from the mean latency
}

// Percentile returns the p-th percentile (0 <= p <= 1) of samples using linear
// interpolation between the two closest ranks. An empty slice yields 0.
func Percentile(samples []float64, p float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	s := make([]float64, len(samples))
	copy(s, samples)
	sort.Float64s(s)

	if p <= 0 {
		return s[0]
	}
	if p >= 1 {
		return s[len(s)-1]
	}

	idx := p * float64(len(s)-1)
	lo := int(math.Floor(idx))
	hi := int(math.Ceil(idx))
	if lo == hi {
		return s[lo]
	}
	frac := idx - float64(lo)
	return s[lo] + (s[hi]-s[lo])*frac
}

// LatencyFromSamples computes latency statistics from per-query samples given
// in milliseconds. QPS is derived from the mean latency: 1000 / meanMs.
func LatencyFromSamples(samples []float64) Latency {
	if len(samples) == 0 {
		return Latency{}
	}

	var sum float64
	for _, s := range samples {
		sum += s
	}
	mean := sum / float64(len(samples))

	l := Latency{
		P50Ms: Percentile(samples, 0.50),
		P95Ms: Percentile(samples, 0.95),
		P99Ms: Percentile(samples, 0.99),
	}
	if mean > 0 {
		l.QPS = 1000 / mean
	}
	return l
}

// MeasureBuildTime runs build and returns how long it took. If build fails, the
// elapsed time is returned together with the error.
func MeasureBuildTime(build func() error) (time.Duration, error) {
	start := time.Now()
	if err := build(); err != nil {
		return time.Since(start), err
	}
	return time.Since(start), nil
}

// HeapInUse returns the heap memory currently in use, in bytes, after forcing a
// garbage collection. Measure an index footprint as
// HeapInUse()-before vs HeapInUse()-after building it, keeping the built index
// reachable (e.g. in a variable captured by the closure).
func HeapInUse() int64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.HeapInuse)
}
