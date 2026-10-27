package hnsw

import (
	"math"
)

// Metric is a distance metric used by the index.
type Metric string

const (
	// MetricL2 is squared Euclidean distance (no sqrt on the hot path).
	MetricL2 Metric = "l2"
	// MetricCosine is cosine distance = 1 - cosine similarity.
	MetricCosine Metric = "cosine"
	// MetricInnerProduct is negative inner product (smaller = closer).
	MetricInnerProduct Metric = "inner_product"
)

// DistanceFunc computes the distance between two equal-length vectors.
type DistanceFunc func(a, b []float32) float32

// SquaredL2 returns the squared L2 distance between a and b.
// Sqrt is intentionally not applied; callers apply it only when reporting distances.
func SquaredL2(a, b []float32) float32 {
	var sum float32
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return sum
}

// Cosine returns the cosine distance 1 - (a.b)/(|a||b|).
// Zero-length vectors are treated as maximally distant (distance 1).
func Cosine(a, b []float32) float32 {
	var dot, na, nb float32
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 1
	}
	return 1 - dot/(float32(math.Sqrt(float64(na*nb))))
}

// InnerProduct returns the negative dot product so that smaller means closer.
func InnerProduct(a, b []float32) float32 {
	var dot float32
	for i := range a {
		dot += a[i] * b[i]
	}
	return -dot
}

// Distance returns the distance function for the given metric.
func Distance(m Metric) DistanceFunc {
	switch m {
	case MetricCosine:
		return Cosine
	case MetricInnerProduct:
		return InnerProduct
	default:
		return SquaredL2
	}
}
