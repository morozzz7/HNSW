package hnsw

import (
	"math/rand"
	"testing"
)

func benchIndex(b *testing.B, n, dim, M, efC int) *Index {
	cfg := defaultConfig()
	cfg.M = M
	cfg.EfConstruction = efC
	cfg.Seed = 1
	idx := New(cfg)
	r := rand.New(rand.NewSource(7))
	for i := 0; i < n; i++ {
		v := make([]float32, dim)
		for j := range v {
			v[j] = float32(r.Float64())
		}
		if _, err := idx.Insert(v, uint32(i)); err != nil {
			b.Fatalf("insert: %v", err)
		}
	}
	return idx
}

func BenchmarkInsert(b *testing.B) {
	dim := 16
	M := 16
	efC := 200
	r := rand.New(rand.NewSource(7))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg := defaultConfig()
		cfg.M = M
		cfg.EfConstruction = efC
		cfg.Seed = int64(i)
		idx := New(cfg)
		for j := 0; j < 1000; j++ {
			v := make([]float32, dim)
			for k := range v {
				v[k] = float32(r.Float64())
			}
			if _, err := idx.Insert(v, uint32(j)); err != nil {
				b.Fatalf("insert: %v", err)
			}
		}
	}
	_ = benchIndex(b, 1000, dim, M, efC)
}

func BenchmarkSearch(b *testing.B) {
	dim := 16
	M := 16
	efC := 200
	efSearch := 100
	k := 10
	idx := benchIndex(b, 5000, dim, M, efC)
	r := rand.New(rand.NewSource(11))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q := make([]float32, dim)
		for j := range q {
			q[j] = float32(r.Float64())
		}
		if _, err := idx.Search(q, k, efSearch); err != nil {
			b.Fatalf("search: %v", err)
		}
	}
}
