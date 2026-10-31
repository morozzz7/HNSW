package hnsw

import "sort"

func selectNeighbors(q []float32, candidates []Result, nodes []*node, dist DistanceFunc,
	Mmax int, useHeuristic bool, owner uint32, level int, extendCandidates, keepPruned bool) []uint32 {

	if useHeuristic {
		return selectNeighborsHeuristic(q, candidates, nodes, dist, Mmax, owner, level, extendCandidates, keepPruned)
	}
	return selectNeighborsSimple(q, candidates, nodes, dist, Mmax, owner)
}

func selectNeighborsSimple(q []float32, candidates []Result, nodes []*node, dist DistanceFunc,
	Mmax int, owner uint32) []uint32 {

	sorted := make([]Result, len(candidates))
	copy(sorted, candidates)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Dist < sorted[j].Dist })

	seen := make(map[uint32]bool)
	out := make([]uint32, 0, Mmax)
	for _, it := range sorted {
		if it.ID == owner || seen[it.ID] {
			continue
		}
		seen[it.ID] = true
		out = append(out, it.ID)
		if len(out) >= Mmax {
			break
		}
	}
	return out
}

func selectNeighborsHeuristic(q []float32, candidates []Result, nodes []*node, dist DistanceFunc,
	Mmax int, owner uint32, level int, extendCandidates, keepPruned bool) []uint32 {

	work := make([]Result, 0, len(candidates))
	seen := make(map[uint32]bool, len(candidates))
	for _, it := range candidates {
		if it.ID == owner || seen[it.ID] {
			continue
		}
		seen[it.ID] = true
		work = append(work, it)
	}

	if extendCandidates {
		base := len(work)
		for i := 0; i < base; i++ {
			nd := nodes[work[i].ID]
			if level >= len(nd.neighbors) {
				continue
			}
			for _, nid := range nd.neighbors[level] {
				if nid == owner || seen[nid] {
					continue
				}
				seen[nid] = true
				work = append(work, Result{ID: nid, Dist: dist(q, nodes[nid].vector)})
			}
		}
	}

	sorted := make([]Result, len(work))
	copy(sorted, work)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Dist < sorted[j].Dist })

	out := make([]uint32, 0, Mmax)
	pickedVecs := make([][]float32, 0, Mmax)
	discarded := make([]Result, 0, len(sorted))
	for _, it := range sorted {
		if len(out) >= Mmax {
			break
		}
		good := true
		for _, pv := range pickedVecs {
			if dist(nodes[it.ID].vector, pv) < it.Dist {
				good = false
				break
			}
		}
		if good {
			out = append(out, it.ID)
			pickedVecs = append(pickedVecs, nodes[it.ID].vector)
		} else {
			discarded = append(discarded, it)
		}
	}

	if keepPruned {
		for _, it := range discarded {
			if len(out) >= Mmax {
				break
			}
			out = append(out, it.ID)
		}
	}
	return out
}
