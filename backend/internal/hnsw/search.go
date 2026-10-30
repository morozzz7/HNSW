package hnsw

import "fmt"

func (idx *Index) searchLayer(q []float32, ep uint32, ef, level int, record bool) ([]Result, *LayerTrace) {
	nodes := idx.nodes
	dist := idx.dist

	var trace *LayerTrace
	if record {
		trace = &LayerTrace{Level: level}
	}

	visited := acquireVisitedSet(len(nodes))
	defer releaseVisitedSet(visited)

	var W minHeap
	var Wc maxHeap
	Wc.cap = ef

	firstDist := dist(q, nodes[ep].vector)
	minHeapPush(&W, Result{ID: ep, Dist: firstDist})
	maxHeapPushUncapped(&Wc, Result{ID: ep, Dist: firstDist})
	visited.add(ep)

	closest := Result{ID: ep, Dist: firstDist}

	for len(W) > 0 {
		cur := minHeapPop(&W)

		if maxHeapLen(Wc) > 0 && cur.Dist > maxHeapPeek(Wc).Dist {
			continue
		}

		if record {
			trace.Visited = append(trace.Visited, cur.ID)
		}
		if cur.Dist < closest.Dist {
			closest = cur
		}

		nd := nodes[cur.ID]
		if level >= len(nd.neighbors) {
			continue
		}
		for _, nid := range nd.neighbors[level] {
			if visited.contains(nid) {
				continue
			}
			d := dist(q, nodes[nid].vector)
			visited.add(nid)
			if maxHeapLen(Wc) < ef || d < maxHeapPeek(Wc).Dist {
				maxHeapPush(&Wc, Result{ID: nid, Dist: d})
				minHeapPush(&W, Result{ID: nid, Dist: d})
			}
		}
	}

	if record {
		trace.Closest = closest.ID
	}

	sorted := maxHeapSorted(Wc)
	return sorted, trace
}

func (idx *Index) Search(q []float32, k, efSearch int) ([]Result, error) {
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
	for l := idx.maxLevel; l > 0; l-- {
		res, _ := idx.searchLayer(q, ep, 1, l, false)
		if len(res) == 0 {
			break
		}
		ep = res[0].ID
	}

	res, _ := idx.searchLayer(q, ep, ef, 0, false)
	if len(res) > k {
		res = res[:k]
	}
	return res, nil
}

func (idx *Index) SearchWithTrace(q []float32, k, efSearch int) ([]Result, *Trace, error) {
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
		ep = res[0].ID
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
