package benchmark

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestPercentile(t *testing.T) {
	samples := []float64{4, 1, 3, 2}

	tests := []struct {
		p    float64
		want float64
	}{
		{0, 1},
		{0.5, 2.5},
		{0.95, 3.85},
		{0.99, 3.97},
		{1, 4},
	}
	for _, tc := range tests {
		if got := Percentile(samples, tc.p); math.Abs(got-tc.want) > 1e-9 {
			t.Fatalf("Percentile(%v) = %v, want %v", tc.p, got, tc.want)
		}
	}

	if got := Percentile(nil, 0.5); got != 0 {
		t.Fatalf("Percentile(nil) = %v, want 0", got)
	}
	if got := Percentile([]float64{5}, 0.5); got != 5 {
		t.Fatalf("Percentile(single) = %v, want 5", got)
	}
}

func TestLatencyFromSamples(t *testing.T) {
	lat := LatencyFromSamples([]float64{10, 20, 30})
	if math.Abs(lat.P50Ms-20) > 1e-9 {
		t.Fatalf("P50 = %v, want 20", lat.P50Ms)
	}
	if math.Abs(lat.P95Ms-29) > 1e-9 {
		t.Fatalf("P95 = %v, want 29", lat.P95Ms)
	}
	if math.Abs(lat.P99Ms-29.8) > 1e-9 {
		t.Fatalf("P99 = %v, want 29.8", lat.P99Ms)
	}
	if math.Abs(lat.QPS-50) > 1e-9 {
		t.Fatalf("QPS = %v, want 50", lat.QPS)
	}

	if got := LatencyFromSamples(nil); got != (Latency{}) {
		t.Fatalf("LatencyFromSamples(nil) = %+v, want zero value", got)
	}
}

func TestMeasureBuildTime(t *testing.T) {
	const delay = 2 * time.Millisecond
	elapsed, err := MeasureBuildTime(func() error {
		time.Sleep(delay)
		return nil
	})
	if err != nil {
		t.Fatalf("MeasureBuildTime: %v", err)
	}
	if elapsed < delay {
		t.Fatalf("elapsed = %v, want >= %v", elapsed, delay)
	}

	want := errors.New("boom")
	if _, err := MeasureBuildTime(func() error { return want }); !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}

func TestHeapInUse(t *testing.T) {
	if got := HeapInUse(); got <= 0 {
		t.Fatalf("HeapInUse() = %d, want > 0", got)
	}
}
