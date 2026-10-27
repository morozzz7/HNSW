package hnsw

// Result pairs a node id with its distance to a query. Distances are compared
// directly on the hot path without any interface{} or reflection. It is the
// element type returned by searches and stored in the typed heaps below.
type Result struct {
	ID   uint32
	Dist float32
}

// minHeap is a min-heap ordered by Dist (closest first). It stores results in a
// flat slice using the standard binary-heap array layout.
type minHeap []Result

// minHeapPush inserts it into h, restoring the min-heap property.
func minHeapPush(h *minHeap, it Result) {
	*h = append(*h, it)
	i := len(*h) - 1
	for i > 0 {
		parent := (i - 1) / 2
		if !((*h)[i].Dist < (*h)[parent].Dist) {
			break
		}
		(*h)[i], (*h)[parent] = (*h)[parent], (*h)[i]
		i = parent
	}
}

// minHeapPop removes and returns the minimum (closest) result, restoring the heap.
func minHeapPop(h *minHeap) Result {
	n := len(*h) - 1
	(*h)[0], (*h)[n] = (*h)[n], (*h)[0]
	it := (*h)[n]
	*h = (*h)[:n]
	i := 0
	for {
		left := 2*i + 1
		right := 2*i + 2
		smallest := i
		if left < n && (*h)[left].Dist < (*h)[smallest].Dist {
			smallest = left
		}
		if right < n && (*h)[right].Dist < (*h)[smallest].Dist {
			smallest = right
		}
		if smallest == i {
			break
		}
		(*h)[i], (*h)[smallest] = (*h)[smallest], (*h)[i]
		i = smallest
	}
	return it
}

// minHeapPeek returns the minimum result without removing it. h must be non-empty.
func minHeapPeek(h minHeap) Result { return h[0] }

// maxHeap is a max-heap ordered by Dist (farthest first). Used as a capped
// "best-so-far" container: it keeps at most cap results, evicting the farthest.
type maxHeap struct {
	items []Result
	cap   int
}

// maxHeapPush inserts it into h. If h is full and it is farther than the
// current worst (root), it is rejected. If h is full and it is closer, the
// root is evicted and it is inserted.
func maxHeapPush(h *maxHeap, it Result) {
	if len(h.items) < h.cap {
		maxHeapPushUncapped(h, it)
		return
	}
	if h.cap == 0 {
		return
	}
	if it.Dist >= h.items[0].Dist {
		return // farther than (or equal to) the worst kept: discard
	}
	// Replace the root (farthest kept) with the closer result and sift down.
	h.items[0] = it
	maxHeapSiftDown(h, 0)
}

// maxHeapPushUncapped inserts it without capacity check.
func maxHeapPushUncapped(h *maxHeap, it Result) {
	h.items = append(h.items, it)
	i := len(h.items) - 1
	for i > 0 {
		parent := (i - 1) / 2
		if !(h.items[i].Dist > h.items[parent].Dist) {
			break
		}
		h.items[i], h.items[parent] = h.items[parent], h.items[i]
		i = parent
	}
}

// maxHeapSiftDown restores the max-heap property starting at index i.
func maxHeapSiftDown(h *maxHeap, i int) {
	n := len(h.items)
	for {
		left := 2*i + 1
		right := 2*i + 2
		largest := i
		if left < n && h.items[left].Dist > h.items[largest].Dist {
			largest = left
		}
		if right < n && h.items[right].Dist > h.items[largest].Dist {
			largest = right
		}
		if largest == i {
			break
		}
		h.items[i], h.items[largest] = h.items[largest], h.items[i]
		i = largest
	}
}

// maxHeapPeek returns the farthest kept result without removing it. h must be non-empty.
func maxHeapPeek(h maxHeap) Result { return h.items[0] }

// maxHeapLen returns the number of results in h.
func maxHeapLen(h maxHeap) int { return len(h.items) }

// maxHeapSorted returns a copy of h.items sorted by Dist ascending (closest first).
func maxHeapSorted(h maxHeap) []Result {
	out := make([]Result, len(h.items))
	copy(out, h.items)
	// Selection sort is fine for small ef; correctness matters more than speed here.
	for i := range out {
		best := i
		for j := i + 1; j < len(out); j++ {
			if out[j].Dist < out[best].Dist {
				best = j
			}
		}
		out[i], out[best] = out[best], out[i]
	}
	return out
}
