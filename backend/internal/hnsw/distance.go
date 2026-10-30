package hnsw

import (
	"math"
)

type Metric string

const (
	MetricL2           Metric = "l2"
	MetricCosine       Metric = "cosine"
	MetricInnerProduct Metric = "inner_product"
)

type DistanceFunc func(a, b []float32) float32

func SquaredL2(a, b []float32) float32 {
	var sum float32
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return sum
}

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

func InnerProduct(a, b []float32) float32 {
	var dot float32
	for i := range a {
		dot += a[i] * b[i]
	}
	return -dot
}

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
