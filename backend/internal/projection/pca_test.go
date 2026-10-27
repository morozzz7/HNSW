package projection

import (
	"math"
	"testing"
)

func approx(a, b float64) bool {
	return math.Abs(a-b) < 1e-6
}

func TestPCA2DLength(t *testing.T) {
	vectors := [][]float32{{1, 2}, {3, 4}, {5, 6}}
	coords, err := PCA2D(vectors)
	if err != nil {
		t.Fatalf("PCA2D = %v, want nil", err)
	}
	if len(coords) != 3 {
		t.Fatalf("len = %d, want 3", len(coords))
	}
}

func TestPCA2DDeterministic(t *testing.T) {
	vectors := [][]float32{{1, 2}, {3, 4}, {5, 6}, {7, 8}}
	a, err := PCA2D(vectors)
	if err != nil {
		t.Fatal(err)
	}
	b, err := PCA2D(vectors)
	if err != nil {
		t.Fatal(err)
	}
	for i := range a {
		if !approx(a[i][0], b[i][0]) || !approx(a[i][1], b[i][1]) {
			t.Fatalf("PCA2D not deterministic at %d: %v vs %v", i, a[i], b[i])
		}
	}
}

func TestPCA2DConstantVectors(t *testing.T) {
	// Zero variance: all vectors identical -> projection is all zeros.
	vectors := [][]float32{{1, 2}, {1, 2}, {1, 2}}
	coords, err := PCA2D(vectors)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range coords {
		if !approx(c[0], 0) || !approx(c[1], 0) {
			t.Fatalf("constant vectors projected to %v, want [0,0]", c)
		}
	}
}

func TestPCA2DErrors(t *testing.T) {
	if _, err := PCA2D(nil); err == nil {
		t.Fatal("PCA2D(nil) = nil, want error")
	}
	if _, err := PCA2D([][]float32{{}}); err == nil {
		t.Fatal("PCA2D(zero-dim) = nil, want error")
	}
	if _, err := PCA2D([][]float32{{1, 2}, {3}}); err == nil {
		t.Fatal("PCA2D(mismatched dim) = nil, want error")
	}
}

func TestPCA2DAxisAligned(t *testing.T) {
	// Variance along x is larger than along y; PC1 should track x.
	vectors := [][]float32{
		{0, 0}, {2, 0}, {4, 0}, // spread on x
		{0, 0}, {0, 1}, {0, 2}, // less spread on y
	}
	coords, err := PCA2D(vectors)
	if err != nil {
		t.Fatal(err)
	}
	// The total spread along PC1 should exceed that along PC2.
	var spreadX, spreadY float64
	for _, c := range coords {
		spreadX += math.Abs(c[0])
		spreadY += math.Abs(c[1])
	}
	if spreadX <= spreadY {
		t.Fatalf("PC1 spread %v not larger than PC2 spread %v", spreadX, spreadY)
	}
}

func TestCache(t *testing.T) {
	c := NewCache()
	if _, ok := c.Get("missing"); ok {
		t.Fatal("Get(missing) should not be ok")
	}
	coords := [][2]float64{{1, 2}, {3, 4}}
	c.Set("ds_000001", coords)
	got, ok := c.Get("ds_000001")
	if !ok {
		t.Fatal("Get after Set should be ok")
	}
	if len(got) != 2 || !approx(got[0][0], 1) || !approx(got[1][1], 4) {
		t.Fatalf("unexpected cached coords: %v", got)
	}
	c.Delete("ds_000001")
	if _, ok := c.Get("ds_000001"); ok {
		t.Fatal("Get after Delete should not be ok")
	}
}
