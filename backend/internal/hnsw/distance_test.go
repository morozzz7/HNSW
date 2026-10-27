package hnsw

import (
	"testing"
)

func TestSquaredL2(t *testing.T) {
	a := []float32{1, 2, 3}
	b := []float32{4, 6, 8}
	// (1-4)^2 + (2-6)^2 + (3-8)^2 = 9 + 16 + 25 = 50
	if got := SquaredL2(a, b); got != 50 {
		t.Fatalf("SquaredL2 = %v, want 50", got)
	}
	// identical vectors -> 0
	if got := SquaredL2(a, a); got != 0 {
		t.Fatalf("SquaredL2 identical = %v, want 0", got)
	}
}

func TestCosine(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{1, 0, 0}
	if got := Cosine(a, b); got != 0 {
		t.Fatalf("Cosine identical = %v, want 0", got)
	}
	// orthogonal
	c := []float32{0, 1, 0}
	if got := Cosine(a, c); got != 1 {
		t.Fatalf("Cosine orthogonal = %v, want 1", got)
	}
	// zero vector -> distance 1
	z := []float32{0, 0, 0}
	if got := Cosine(a, z); got != 1 {
		t.Fatalf("Cosine zero = %v, want 1", got)
	}
}

func TestInnerProduct(t *testing.T) {
	a := []float32{1, 2, 3}
	b := []float32{4, 5, 6}
	// dot = 4 + 10 + 18 = 32, negative inner product = -32
	if got := InnerProduct(a, b); got != -32 {
		t.Fatalf("InnerProduct = %v, want -32", got)
	}
}

func TestDistanceDispatch(t *testing.T) {
	a := []float32{1, 2, 3}
	b := []float32{4, 5, 6}

	// l2 -> squared L2
	if got := Distance(MetricL2)(a, b); got != SquaredL2(a, b) {
		t.Error("Distance(l2) should return SquaredL2 behavior")
	}
	// cosine -> cosine
	if got := Distance(MetricCosine)(a, b); got != Cosine(a, b) {
		t.Error("Distance(cosine) should return Cosine behavior")
	}
	// inner_product -> inner product
	if got := Distance(MetricInnerProduct)(a, b); got != InnerProduct(a, b) {
		t.Error("Distance(inner_product) should return InnerProduct behavior")
	}
	// unknown -> default squared L2
	if got := Distance("unknown")(a, b); got != SquaredL2(a, b) {
		t.Error("Distance(unknown) should default to SquaredL2")
	}
}
