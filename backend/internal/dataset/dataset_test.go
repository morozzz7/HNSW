package dataset

import (
	"math"
	"testing"
)

func TestValidateOK(t *testing.T) {
	vectors := [][]float32{{1, 2, 3}, {4, 5, 6}}
	if err := Validate(vectors); err != nil {
		t.Fatalf("Validate = %v, want nil", err)
	}
}

func TestValidateEmpty(t *testing.T) {
	if err := Validate([][]float32{}); err == nil {
		t.Fatal("Validate(empty) = nil, want error")
	}
}

func TestValidateZeroDim(t *testing.T) {
	if err := Validate([][]float32{{}}); err == nil {
		t.Fatal("Validate(zero-dim) = nil, want error")
	}
}

func TestValidateDimMismatch(t *testing.T) {
	vectors := [][]float32{{1, 2}, {3}}
	if err := Validate(vectors); err == nil {
		t.Fatal("Validate(dim mismatch) = nil, want error")
	}
}

func TestValidateNaN(t *testing.T) {
	vectors := [][]float32{{1, float32(math.NaN())}}
	if err := Validate(vectors); err == nil {
		t.Fatal("Validate(NaN) = nil, want error")
	}
}

func TestValidateInf(t *testing.T) {
	vectors := [][]float32{{1, float32(math.Inf(1))}}
	if err := Validate(vectors); err == nil {
		t.Fatal("Validate(Inf) = nil, want error")
	}
}

func TestValidateTooLarge(t *testing.T) {
	vectors := make([][]float32, MaxVectors+1)
	for i := range vectors {
		vectors[i] = []float32{1}
	}
	if err := Validate(vectors); err == nil {
		t.Fatal("Validate(oversized) = nil, want error")
	}
}

func TestNewDatasetStatsAndPreview(t *testing.T) {
	vectors := [][]float32{{0, 10}, {2, 8}, {4, 6}, {6, 4}, {8, 2}, {10, 0}}
	ds, err := NewDataset("csv", "", vectors)
	if err != nil {
		t.Fatalf("NewDataset = %v, want nil", err)
	}
	if ds.N != 6 || ds.Dim != 2 {
		t.Fatalf("ds.N=%d ds.Dim=%d, want 6 2", ds.N, ds.Dim)
	}
	if len(ds.Preview) != min(len(vectors), previewRows) {
		t.Fatalf("len(preview)=%d, want %d", len(ds.Preview), min(len(vectors), previewRows))
	}
	if ds.Stats == nil {
		t.Fatal("stats is nil")
	}
	if ds.Stats.Mean[0] != 5 {
		t.Fatalf("mean[0] = %v, want 5", ds.Stats.Mean[0])
	}
	if ds.Stats.Min[0] != 0 || ds.Stats.Max[1] != 10 {
		t.Fatalf("min/max = %v %v", ds.Stats.Min, ds.Stats.Max)
	}
}

func TestStoreCRUD(t *testing.T) {
	s := NewStore()
	if s.Count() != 0 {
		t.Fatalf("Count = %d, want 0", s.Count())
	}

	ds1, _ := NewDataset("generate", "uniform", [][]float32{{1, 2}})
	ds2, _ := NewDataset("generate", "gaussian", [][]float32{{3, 4}})
	id1 := s.Add(ds1)
	id2 := s.Add(ds2)
	if id1 == id2 {
		t.Fatal("ids should differ")
	}

	if got := s.Count(); got != 2 {
		t.Fatalf("Count = %d, want 2", got)
	}
	if got := s.Get(id1); got == nil || got.Dim != 2 {
		t.Fatalf("Get(%s) = %v", id1, got)
	}
	if s.Get("nope") != nil {
		t.Fatal("Get(missing) should be nil")
	}

	list := s.List()
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}
	for _, ds := range list {
		if ds.Vectors != nil {
			t.Fatal("List should not expose full vectors")
		}
	}

	if !s.Remove(id1) {
		t.Fatal("Remove should return true")
	}
	if s.Remove(id1) {
		t.Fatal("Remove twice should return false")
	}
	if got := s.Count(); got != 1 {
		t.Fatalf("Count after remove = %d, want 1", got)
	}
}

func TestNewDatasetValidationError(t *testing.T) {
	_, err := NewDataset("csv", "", [][]float32{{1, 2}, {3}})
	if err == nil {
		t.Fatal("NewDataset(dim mismatch) = nil, want error")
	}
}
