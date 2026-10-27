package dataset

import (
	"math"
	"testing"
)

func TestGenerateUniform(t *testing.T) {
	vecs, err := Generate(100, 5, "uniform", 1)
	if err != nil {
		t.Fatalf("Generate(uniform) = %v, want nil", err)
	}
	if len(vecs) != 100 || len(vecs[0]) != 5 {
		t.Fatalf("got %d x %d, want 100 x 5", len(vecs), len(vecs[0]))
	}
	for _, v := range vecs {
		for _, x := range v {
			if x < 0 || x >= 1 {
				t.Fatalf("uniform value %v out of [0,1)", x)
			}
		}
	}
	// Determinism: same seed => same vectors.
	vecs2, err := Generate(100, 5, "uniform", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !equalVecs(vecs, vecs2) {
		t.Fatal("same seed should produce identical vectors")
	}
}

func TestGenerateGaussian(t *testing.T) {
	vecs, err := Generate(50, 4, "gaussian", 7)
	if err != nil {
		t.Fatalf("Generate(gaussian) = %v, want nil", err)
	}
	if len(vecs) != 50 || len(vecs[0]) != 4 {
		t.Fatalf("got %d x %d, want 50 x 4", len(vecs), len(vecs[0]))
	}
	for _, v := range vecs {
		for _, x := range v {
			if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
				t.Fatal("gaussian produced NaN/Inf")
			}
		}
	}
}

func TestGenerateClusters(t *testing.T) {
	vecs, err := Generate(200, 3, "clusters", 5)
	if err != nil {
		t.Fatalf("Generate(clusters) = %v, want nil", err)
	}
	if len(vecs) != 200 || len(vecs[0]) != 3 {
		t.Fatalf("got %d x %d, want 200 x 3", len(vecs), len(vecs[0]))
	}
}

func TestGenerateInvalid(t *testing.T) {
	if _, err := Generate(0, 5, "uniform", 1); err == nil {
		t.Fatal("Generate(n=0) = nil, want error")
	}
	if _, err := Generate(5, 0, "uniform", 1); err == nil {
		t.Fatal("Generate(dim=0) = nil, want error")
	}
	if _, err := Generate(5, 5, "weird", 1); err == nil {
		t.Fatal("Generate(bad dist) = nil, want error")
	}
}

func equalVecs(a, b [][]float32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}
	return true
}
