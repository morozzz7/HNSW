package hnsw

import "sort"

// selectNeighbors chooses up to Mmax neighbor ids from candidates, ordered by
// distance to q ascending. It implements either Algorithm 3 (simple) or
// Algorithm 4 (heuristic) from Malkov & Yashunin, 2016, depending on useHeuristic.
//
// Candidates must not include q itself; if it does, it is skipped.
func selectNeighbors(q []float32, candidates []Result, nodes []*node, dist DistanceFunc,
	Mmax int, useHeuristic bool) []uint32 {

	if useHeuristic {
		return selectNeighborsHeuristic(q, candidates, nodes, dist, Mmax)
	}
	return selectNeighborsSimple(q, candidates, nodes, dist, Mmax)
}

// selectNeighborsSimple implements Algorithm 3: keep the Mmax closest candidates.
func selectNeighborsSimple(q []float32, candidates []Result, nodes []*node, dist DistanceFunc,
	Mmax int) []uint32 {

	// Sort by distance ascending.
	sorted := make([]Result, len(candidates))
	copy(sorted, candidates)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Dist < sorted[j].Dist })

	seen := make(map[uint32]bool)
	out := make([]uint32, 0, Mmax)
	for _, it := range sorted {
		if seen[it.ID] {
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

// selectNeighborsHeuristic implements Algorithm 4: greedily pick candidates that
// are mutually far (each new pick must be farther from every already-picked
// neighbor than from q), up to Mmax picks.
func selectNeighborsHeuristic(q []float32, candidates []Result, nodes []*node, dist DistanceFunc,
	Mmax int) []uint32 {

	sorted := make([]Result, len(candidates))
	copy(sorted, candidates)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Dist < sorted[j].Dist })

	out := make([]uint32, 0, Mmax)
	// vectors of already-picked neighbors for fast distance checks
	pickedVecs := make([][]float32, 0, Mmax)
	for _, it := range sorted {
		if len(out) >= Mmax {
			break
		}
		good := true
		qdist := it.Dist
		for _, pv := range pickedVecs {
			if dist(nodes[it.ID].vector, pv) < qdist {
				good = false
				break
			}
		}
		if good {
			out = append(out, it.ID)
			pickedVecs = append(pickedVecs, nodes[it.ID].vector)
		}
	}
	return out
}
