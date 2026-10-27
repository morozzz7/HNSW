package hnsw

import (
	"sync"
	"testing"
)

func TestVisitedSet(t *testing.T) {
	v := acquireVisitedSet(10)
	defer releaseVisitedSet(v)

	if v.contains(5) {
		t.Fatal("contains should be false initially")
	}
	v.add(5)
	if !v.contains(5) {
		t.Fatal("contains should be true after add")
	}
	if v.contains(6) {
		t.Fatal("contains should be false for unadded id")
	}

	// Acquire a new set: generation changes, old marks invalid.
	v2 := acquireVisitedSet(10)
	defer releaseVisitedSet(v2)
	if v2.contains(5) {
		t.Fatal("new generation should not contain old mark")
	}
}

func TestVisitedSetConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				v := acquireVisitedSet(64)
				for id := 0; id < 64; id++ {
					v.add(uint32(id))
				}
				for id := 0; id < 64; id++ {
					if !v.contains(uint32(id)) {
						t.Errorf("concurrent contains failed for id %d", id)
					}
				}
				releaseVisitedSet(v)
			}
		}()
	}
	wg.Wait()
}
