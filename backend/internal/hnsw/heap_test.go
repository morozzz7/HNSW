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
	// Pop order: 0.1, 0.2, 0.3, 0.5
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

	// Full now (cap=3). Pushing 0.7 (farther than worst 0.9? No, 0.7 < 0.9, so replaces root).
	maxHeapPush(&h, Result{ID: 4, Dist: 0.7})
	// Root should now be 0.7 (farthest kept). Wait: kept items are {0.1, 0.5, 0.7}, worst=0.7.
	// Pushing 0.2 (closer than worst 0.7) replaces 0.7 -> kept {0.1, 0.2, 0.5}
	maxHeapPush(&h, Result{ID: 5, Dist: 0.2})
	// Pushing 0.8 (farther than worst 0.5) -> rejected
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
