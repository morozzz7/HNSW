package benchmark

import (
	"math"
	"runtime"
	"sort"
	"time"
)

type Latency struct {
	P50Ms float64
	P95Ms float64
	P99Ms float64
	QPS   float64
}

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

func MeasureBuildTime(build func() error) (time.Duration, error) {
	start := time.Now()
	if err := build(); err != nil {
		return time.Since(start), err
	}
	return time.Since(start), nil
}

func HeapInUse() int64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.HeapInuse)
}
