package bruteforce

import (
	"math"
	"math/rand"
	"reflect"
	"testing"

	"hnsw-coursework/internal/hnsw"
)

func randVec(r *rand.Rand, dim int) []float32 {
	v := make([]float32, dim)
	for i := range v {
		v[i] = float32(r.Float64())
	}
	return v
}

func TestNewValidation(t *testing.T) {
	tests := []struct {
		name    string
		vectors [][]float32
		dist    DistFunc
	}{
		{"no vectors", nil, hnsw.SquaredL2},
		{"zero dimension", [][]float32{{}}, hnsw.SquaredL2},
		{"inconsistent dimension", [][]float32{{1, 2}, {3}}, hnsw.SquaredL2},
		{"nil distance", [][]float32{{1, 2}}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(tc.vectors, tc.dist); err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}

func TestSearchExact(t *testing.T) {
	vectors := [][]float32{
		{0, 0},
		{1, 0},
		{0, 1},
		{3, 3},
	}
	idx, err := New(vectors, hnsw.SquaredL2)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res, err := idx.Search([]float32{0, 0}, 2)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("len = %d, want 2", len(res))
	}
	// Distances: id0=0, id1=1, id2=1, id3=18. Ties are broken by id.
	if res[0].ID != 0 || res[0].Dist != 0 {
		t.Fatalf("res[0] = {%d, %v}, want {0, 0}", res[0].ID, res[0].Dist)
	}
	if res[1].ID != 1 || res[1].Dist != 1 {
		t.Fatalf("res[1] = {%d, %v}, want {1, 1}", res[1].ID, res[1].Dist)
	}
}

func TestSearchSelfQuery(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	const n, dim = 200, 8
	vectors := make([][]float32, n)
	for i := range vectors {
		vectors[i] = randVec(r, dim)
	}
	idx, err := New(vectors, hnsw.SquaredL2)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for i, v := range vectors {
		res, err := idx.Search(v, 1)
		if err != nil {
			t.Fatalf("search %d: %v", i, err)
		}
		if len(res) != 1 {
			t.Fatalf("search %d: len = %d, want 1", i, len(res))
		}
		if res[0].ID != uint32(i) {
			t.Fatalf("search %d: nearest = %d, want %d", i, res[0].ID, i)
		}
		if math.Abs(float64(res[0].Dist)) > 1e-6 {
			t.Fatalf("search %d: distance to self = %v, want 0", i, res[0].Dist)
		}
	}
}

func TestSearchParallelMatchesSequential(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	const n, dim, k = 2000, 16, 10
	vectors := make([][]float32, n)
	for i := range vectors {
		vectors[i] = randVec(r, dim)
	}

	seq, err := New(vectors, hnsw.SquaredL2)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	par, err := NewWithWorkers(vectors, hnsw.SquaredL2, 4)
	if err != nil {
		t.Fatalf("NewWithWorkers: %v", err)
	}

	for q := 0; q < 10; q++ {
		query := randVec(r, dim)
		gotSeq, err := seq.Search(query, k)
		if err != nil {
			t.Fatalf("sequential search: %v", err)
		}
		gotPar, err := par.Search(query, k)
		if err != nil {
			t.Fatalf("parallel search: %v", err)
		}
		if !reflect.DeepEqual(gotSeq, gotPar) {
			t.Fatalf("query %d: parallel results differ from sequential:\nseq=%v\npar=%v",
				q, gotSeq, gotPar)
		}
	}
}

func TestSearchKExceedsN(t *testing.T) {
	vectors := [][]float32{{0}, {1}, {2}, {3}, {4}}
	idx, err := New(vectors, hnsw.SquaredL2)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	res, err := idx.Search([]float32{0}, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res) != len(vectors) {
		t.Fatalf("len = %d, want %d", len(res), len(vectors))
	}
}

func TestSearchNonPositiveK(t *testing.T) {
	idx, err := New([][]float32{{0}, {1}}, hnsw.SquaredL2)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for _, k := range []int{0, -1} {
		res, err := idx.Search([]float32{0}, k)
		if err != nil {
			t.Fatalf("k=%d: %v", k, err)
		}
		if len(res) != 0 {
			t.Fatalf("k=%d: len = %d, want 0", k, len(res))
		}
	}
}

func TestSearchDimensionMismatch(t *testing.T) {
	idx, err := New([][]float32{{1, 2}}, hnsw.SquaredL2)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := idx.Search([]float32{1, 2, 3}, 1); err == nil {
		t.Fatal("expected a dimension mismatch error, got nil")
	}
}

func TestRecall(t *testing.T) {
	exact := []Result{{ID: 1, Dist: 0}, {ID: 2, Dist: 1}, {ID: 3, Dist: 2}}
	approx := []Result{{ID: 2, Dist: 0}, {ID: 9, Dist: 1}, {ID: 1, Dist: 2}}

	if got := Recall(exact, approx, 3); math.Abs(got-2.0/3.0) > 1e-9 {
		t.Fatalf("recall = %v, want %v", got, 2.0/3.0)
	}
	if got := Recall(exact, exact, 3); got != 1 {
		t.Fatalf("perfect recall = %v, want 1", got)
	}
	if got := Recall(exact, approx, 0); got != 0 {
		t.Fatalf("recall with k=0 = %v, want 0", got)
	}

	// Only the exact top-k is considered.
	longer := append([]Result{{ID: 10, Dist: 6}}, exact...)
	if got := Recall(longer, approx, 3); math.Abs(got-2.0/3.0) > 1e-9 {
		t.Fatalf("recall with longer exact list = %v, want %v", got, 2.0/3.0)
	}
}
