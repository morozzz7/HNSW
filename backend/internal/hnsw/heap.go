package hnsw

// item pairs a node id with its distance to a query. Distances are compared
// directly on the hot path without any interface{} or reflection.
type item struct {
	id   uint32
	dist float32
}

// minHeap is a min-heap ordered by dist (closest first). It stores items in a
// flat slice using the standard binary-heap array layout.
type minHeap []item

// minHeapPush inserts it into h, restoring the min-heap property.
func minHeapPush(h *minHeap, it item) {
	*h = append(*h, it)
	i := len(*h) - 1
	for i > 0 {
		parent := (i - 1) / 2
		if !((*h)[i].dist < (*h)[parent].dist) {
			break
		}
		(*h)[i], (*h)[parent] = (*h)[parent], (*h)[i]
		i = parent
	}
}

// minHeapPop removes and returns the minimum (closest) item, restoring the heap.
func minHeapPop(h *minHeap) item {
	n := len(*h) - 1
	(*h)[0], (*h)[n] = (*h)[n], (*h)[0]
	it := (*h)[n]
	*h = (*h)[:n]
	i := 0
	for {
		left := 2*i + 1
		right := 2*i + 2
		smallest := i
		if left < n && (*h)[left].dist < (*h)[smallest].dist {
			smallest = left
		}
		if right < n && (*h)[right].dist < (*h)[smallest].dist {
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

// minHeapPeek returns the minimum item without removing it. h must be non-empty.
func minHeapPeek(h minHeap) item { return h[0] }

// maxHeap is a max-heap ordered by dist (farthest first). Used as a capped
// "best-so-far" container: it keeps at most cap items, evicting the farthest.
type maxHeap struct {
	items []item
	cap   int
}

// maxHeapPush inserts it into h. If h is full and it is farther than the
// current worst (root), it is rejected. If h is full and it is closer, the
// root is evicted and it is inserted.
func maxHeapPush(h *maxHeap, it item) {
	if len(h.items) < h.cap {
		maxHeapPushUncapped(h, it)
		return
	}
	if h.cap == 0 {
		return
	}
	if it.dist >= h.items[0].dist {
		return // farther than (or equal to) the worst kept: discard
	}
	// Replace the root (farthest kept) with the closer item and sift down.
	h.items[0] = it
	maxHeapSiftDown(h, 0)
}

// maxHeapPushUncapped inserts it without capacity check.
func maxHeapPushUncapped(h *maxHeap, it item) {
	h.items = append(h.items, it)
	i := len(h.items) - 1
	for i > 0 {
		parent := (i - 1) / 2
		if !(h.items[i].dist > h.items[parent].dist) {
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
		if left < n && h.items[left].dist > h.items[largest].dist {
			largest = left
		}
		if right < n && h.items[right].dist > h.items[largest].dist {
			largest = right
		}
		if largest == i {
			break
		}
		h.items[i], h.items[largest] = h.items[largest], h.items[i]
		i = largest
	}
}

// maxHeapPeek returns the farthest kept item without removing it. h must be non-empty.
func maxHeapPeek(h maxHeap) item { return h.items[0] }

// maxHeapLen returns the number of items in h.
func maxHeapLen(h maxHeap) int { return len(h.items) }

// maxHeapSorted returns a copy of h.items sorted by dist ascending (closest first).
func maxHeapSorted(h maxHeap) []item {
	out := make([]item, len(h.items))
	copy(out, h.items)
	// Selection sort is fine for small ef; correctness matters more than speed here.
	for i := range out {
		best := i
		for j := i + 1; j < len(out); j++ {
			if out[j].dist < out[best].dist {
				best = j
			}
		}
		out[i], out[best] = out[best], out[i]
	}
	return out
}
