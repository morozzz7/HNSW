package hnsw

import "fmt"

// searchLayer implements Algorithm 2 (SEARCH-LAYER) from Malkov & Yashunin, 2016.
//
// It searches layer `level` starting from entry point `ep`, keeping at most `ef`
// closest candidates in a capped max-heap. It returns the best candidates sorted
// by distance ascending. If record is true, it also fills a LayerTrace with the
// visit order and closest node found at this layer.
func (idx *Index) searchLayer(q []float32, ep uint32, ef, level int, record bool) ([]item, *LayerTrace) {
	nodes := idx.nodes
	dist := idx.dist

	var trace *LayerTrace
	if record {
		trace = &LayerTrace{Level: level}
	}

	visited := acquireVisitedSet(len(nodes))
	defer releaseVisitedSet(visited)

	// W: min-heap of candidates to process (closest processed first).
	// Wc: capped max-heap of best ef candidates found (farthest at root).
	var W minHeap
	var Wc maxHeap
	Wc.cap = ef

	firstDist := dist(q, nodes[ep].vector)
	minHeapPush(&W, item{id: ep, dist: firstDist})
	maxHeapPushUncapped(&Wc, item{id: ep, dist: firstDist})
	visited.add(ep)

	var closest item
	closest = item{id: ep, dist: firstDist}

	for len(W) > 0 {
		cur := minHeapPop(&W)

		// Stop if the current node is farther than the worst kept candidate.
		if maxHeapLen(Wc) > 0 && cur.dist > maxHeapPeek(Wc).dist {
			continue
		}

		if record {
			trace.Visited = append(trace.Visited, cur.id)
		}
		if cur.dist < closest.dist {
			closest = cur
		}

		nd := nodes[cur.id]
		if level >= len(nd.neighbors) {
			continue
		}
		for _, nid := range nd.neighbors[level] {
			if visited.contains(nid) {
				continue
			}
			d := dist(q, nodes[nid].vector)
			visited.add(nid)
			if maxHeapLen(Wc) < ef || d < maxHeapPeek(Wc).dist {
				maxHeapPush(&Wc, item{id: nid, dist: d})
				minHeapPush(&W, item{id: nid, dist: d})
			}
		}
	}

	if record {
		trace.Closest = closest.id
	}

	// Return best candidates sorted by distance ascending.
	sorted := maxHeapSorted(Wc)
	return sorted, trace
}

// Search performs k-NN search using Algorithm 5 with per-query efSearch.
// ef = max(efSearch, k). It does not record a trace.
func (idx *Index) Search(q []float32, k, efSearch int) ([]item, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if len(idx.nodes) == 0 {
		return nil, nil
	}
	if len(q) != idx.dim {
		return nil, fmt.Errorf("query dimension %d does not match index dimension %d", len(q), idx.dim)
	}

	ef := efSearch
	if k > ef {
		ef = k
	}

	ep := idx.entryPoint
	// Descend from maxLevel to layer 1 using ef=1 (closest-only at each layer).
	for l := idx.maxLevel; l > 0; l-- {
		res, _ := idx.searchLayer(q, ep, 1, l, false)
		if len(res) == 0 {
			break
		}
		ep = res[0].id
	}

	// Layer 0 with full ef.
	res, _ := idx.searchLayer(q, ep, ef, 0, false)
	if len(res) > k {
		res = res[:k]
	}
	return res, nil
}

// SearchWithTrace performs k-NN search and records the per-layer search path.
// Returns (results, trace, error). The trace is used by the frontend for
// visualization of the search descent.
func (idx *Index) SearchWithTrace(q []float32, k, efSearch int) ([]item, *Trace, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if len(idx.nodes) == 0 {
		return nil, nil, nil
	}
	if len(q) != idx.dim {
		return nil, nil, fmt.Errorf("query dimension %d does not match index dimension %d", len(q), idx.dim)
	}

	ef := efSearch
	if k > ef {
		ef = k
	}

	trace := &Trace{}

	ep := idx.entryPoint
	for l := idx.maxLevel; l > 0; l-- {
		res, lt := idx.searchLayer(q, ep, 1, l, true)
		if lt != nil {
			trace.Layers = append(trace.Layers, *lt)
		}
		if len(res) == 0 {
			break
		}
		ep = res[0].id
	}

	res, lt := idx.searchLayer(q, ep, ef, 0, true)
	if lt != nil {
		trace.Layers = append(trace.Layers, *lt)
	}
	if len(res) > k {
		res = res[:k]
	}

	return res, trace, nil
}
