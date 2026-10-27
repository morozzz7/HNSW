package bruteforce

import (
	"math/rand"
	"testing"

	"hnsw-coursework/internal/hnsw"
)

func benchIndex(b *testing.B, n, dim, workers int) *Index {
	b.Helper()
	r := rand.New(rand.NewSource(7))
	vectors := make([][]float32, n)
	for i := range vectors {
		vectors[i] = randVec(r, dim)
	}
	idx, err := NewWithWorkers(vectors, hnsw.SquaredL2, workers)
	if err != nil {
		b.Fatalf("NewWithWorkers: %v", err)
	}
	return idx
}

func BenchmarkSearch(b *testing.B) {
	const n, dim, k = 10000, 16, 10
	idx := benchIndex(b, n, dim, 0)
	r := rand.New(rand.NewSource(11))
	query := randVec(r, dim)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := idx.Search(query, k); err != nil {
			b.Fatalf("search: %v", err)
		}
	}
}

func BenchmarkSearchParallel(b *testing.B) {
	const n, dim, k = 10000, 16, 10
	idx := benchIndex(b, n, dim, 4)
	r := rand.New(rand.NewSource(11))
	query := randVec(r, dim)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := idx.Search(query, k); err != nil {
			b.Fatalf("search: %v", err)
		}
	}
}
