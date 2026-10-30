package hnsw

import "sort"

func selectNeighbors(q []float32, candidates []Result, nodes []*node, dist DistanceFunc,
	Mmax int, useHeuristic bool) []uint32 {

	if useHeuristic {
		return selectNeighborsHeuristic(q, candidates, nodes, dist, Mmax)
	}
	return selectNeighborsSimple(q, candidates, nodes, dist, Mmax)
}

func selectNeighborsSimple(q []float32, candidates []Result, nodes []*node, dist DistanceFunc,
	Mmax int) []uint32 {

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

func selectNeighborsHeuristic(q []float32, candidates []Result, nodes []*node, dist DistanceFunc,
	Mmax int) []uint32 {

	sorted := make([]Result, len(candidates))
	copy(sorted, candidates)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Dist < sorted[j].Dist })

	out := make([]uint32, 0, Mmax)
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
