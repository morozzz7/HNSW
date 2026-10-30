package hnsw

func (idx *Index) Insert(vector []float32, externalID uint32) (uint32, error) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if err := idx.ensureDim(vector); err != nil {
		return 0, err
	}

	level := idx.randomLevel()
	internalID := uint32(len(idx.nodes))

	nd := &node{
		id:        internalID,
		vector:    vector,
		maxLevel:  level,
		neighbors: make([][]uint32, level+1),
	}

	idx.nodes = append(idx.nodes, nd)

	if len(idx.nodes) == 1 {
		idx.entryPoint = internalID
		if level > idx.maxLevel {
			idx.maxLevel = level
		}
		idx.extToInt[externalID] = internalID
		idx.intToExt = append(idx.intToExt, externalID)
		return internalID, nil
	}

	ep := idx.entryPoint
	maxL := idx.maxLevel

	for l := maxL; l > level; l-- {
		res, _ := idx.searchLayer(vector, ep, 1, l, false)
		if len(res) == 0 {
			break
		}
		ep = res[0].ID
	}

	insertMaxL := level
	if insertMaxL > maxL {
		insertMaxL = maxL
	}
	for l := insertMaxL; l >= 0; l-- {
		Mmax := idx.config.Mmax(l)
		W, _ := idx.searchLayer(vector, ep, idx.config.EfConstruction, l, false)

		neighbors := selectNeighbors(vector, W, idx.nodes, idx.dist, Mmax, idx.config.UseHeuristic)

		for _, nid := range neighbors {
			addLinkLocked(nd, idx.nodes[nid], l)
		}

		for _, nid := range neighbors {
			e := idx.nodes[nid]
			if len(e.neighbors[l]) > Mmax {
				cand := neighborCandidates(e.vector, e.neighbors[l], idx.nodes, idx.dist)
				newNeighbors := selectNeighbors(e.vector, cand, idx.nodes, idx.dist, Mmax, idx.config.UseHeuristic)
				oldNeighbors := e.neighbors[l]
				e.neighbors[l] = newNeighbors

				newSet := make(map[uint32]bool, len(newNeighbors))
				for _, nn := range newNeighbors {
					newSet[nn] = true
				}
				for _, old := range oldNeighbors {
					if !newSet[old] {
						removeLinkLocked(idx.nodes[old], e.id, l)
					}
				}
			}
		}

		if len(W) > 0 {
			ep = W[0].ID
		}
	}

	if level > maxL {
		idx.entryPoint = internalID
		idx.maxLevel = level
	}

	idx.extToInt[externalID] = internalID
	idx.intToExt = append(idx.intToExt, externalID)

	return internalID, nil
}

func addLinkLocked(a, b *node, level int) {
	if !containsUint32(a.neighbors[level], b.id) {
		a.neighbors[level] = append(a.neighbors[level], b.id)
	}
	if !containsUint32(b.neighbors[level], a.id) {
		b.neighbors[level] = append(b.neighbors[level], a.id)
	}
}

func removeLinkLocked(a *node, b uint32, level int) {
	a.neighbors[level] = removeUint32(a.neighbors[level], b)
}

func neighborCandidates(v []float32, ids []uint32, nodes []*node, dist DistanceFunc) []Result {
	res := make([]Result, 0, len(ids))
	for _, id := range ids {
		res = append(res, Result{ID: id, Dist: dist(v, nodes[id].vector)})
	}
	return res
}

func containsUint32(s []uint32, x uint32) bool {
	for _, v := range s {
		if v == x {
			return true
		}
	}
	return false
}

func removeUint32(s []uint32, x uint32) []uint32 {
	for i, v := range s {
		if v == x {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}
