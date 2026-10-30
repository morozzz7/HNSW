package hnsw

type Result struct {
	ID   uint32
	Dist float32
}

type minHeap []Result

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

func minHeapPeek(h minHeap) Result { return h[0] }

type maxHeap struct {
	items []Result
	cap   int
}

func maxHeapPush(h *maxHeap, it Result) {
	if len(h.items) < h.cap {
		maxHeapPushUncapped(h, it)
		return
	}
	if h.cap == 0 {
		return
	}
	if it.Dist >= h.items[0].Dist {
		return
	}
	h.items[0] = it
	maxHeapSiftDown(h, 0)
}

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

func maxHeapPeek(h maxHeap) Result { return h.items[0] }

func maxHeapLen(h maxHeap) int { return len(h.items) }

func maxHeapSorted(h maxHeap) []Result {
	out := make([]Result, len(h.items))
	copy(out, h.items)
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
