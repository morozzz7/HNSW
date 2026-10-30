package hnsw

import (
	"testing"
)

func TestMinHeapOrdering(t *testing.T) {
	var h minHeap
	minHeapPush(&h, Result{ID: 1, Dist: 0.5})
	minHeapPush(&h, Result{ID: 2, Dist: 0.1})
	minHeapPush(&h, Result{ID: 3, Dist: 0.3})
	minHeapPush(&h, Result{ID: 4, Dist: 0.2})

	if len(h) != 4 {
		t.Fatalf("len = %d, want 4", len(h))
	}
	want := []uint32{2, 4, 3, 1}
	for i, wid := range want {
		it := minHeapPop(&h)
		if it.ID != wid {
			t.Fatalf("pop[%d] = %d, want %d", i, it.ID, wid)
		}
	}
	if len(h) != 0 {
		t.Fatalf("len after pop = %d, want 0", len(h))
	}
}

func TestMaxHeapCapped(t *testing.T) {
	h := maxHeap{cap: 3}
	maxHeapPush(&h, Result{ID: 1, Dist: 0.9})
	maxHeapPush(&h, Result{ID: 2, Dist: 0.1})
	maxHeapPush(&h, Result{ID: 3, Dist: 0.5})

	maxHeapPush(&h, Result{ID: 4, Dist: 0.7})
	maxHeapPush(&h, Result{ID: 5, Dist: 0.2})
	maxHeapPush(&h, Result{ID: 6, Dist: 0.8})

	if maxHeapLen(h) != 3 {
		t.Fatalf("len = %d, want 3", maxHeapLen(h))
	}
	sorted := maxHeapSorted(h)
	want := []float32{0.1, 0.2, 0.5}
	for i, d := range want {
		if sorted[i].Dist != d {
			t.Fatalf("sorted[%d].Dist = %v, want %v", i, sorted[i].Dist, d)
		}
	}
}
