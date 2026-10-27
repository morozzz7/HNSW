package hnsw

// Insert adds vector to the index using Algorithm 1 (INSERT) from
// Malkov & Yashunin, 2016. externalID is the caller-supplied id; it is stored
// in the external-id mapping table. Returns the internal id assigned to the
// inserted vector.
//
// The caller must NOT hold idx.mu; Insert takes the write lock itself.
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

	// Append nd to the slice now so that neighbor lookups (during overflow
	// re-selection) can resolve nodes[nd.id]. It is not yet linked into the
	// graph; links are added below.
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
	maxL := idx.maxLevel // snapshot before we potentially raise maxLevel

	// Descend from maxLevel down to level+1 using ef=1 (closest-only per layer).
	// Layers above level(x) only contain already-inserted high-level nodes.
	for l := maxL; l > level; l-- {
		res, _ := idx.searchLayer(vector, ep, 1, l, false)
		if len(res) == 0 {
			break
		}
		ep = res[0].id
	}

	// Insert at levels min(level, maxL) down to 0. Levels above maxL have no
	// other nodes yet, so x is inserted there only as the new entry point.
	insertMaxL := level
	if insertMaxL > maxL {
		insertMaxL = maxL
	}
	for l := insertMaxL; l >= 0; l-- {
		Mmax := idx.config.Mmax(l)
		W, _ := idx.searchLayer(vector, ep, idx.config.EfConstruction, l, false)

		neighbors := selectNeighbors(vector, W, idx.nodes, idx.dist, Mmax, idx.config.UseHeuristic)

		// Add bidirectional links between x and each selected neighbor.
		for _, nid := range neighbors {
			addLinkLocked(nd, idx.nodes[nid], l)
		}

		// Shrink overflowing neighbors: re-select their neighbor lists at this level.
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

		// Closest candidate becomes the entry point for the next lower layer.
		if len(W) > 0 {
			ep = W[0].id
		}
	}

	// Update entry point and maxLevel when x has a higher level.
	if level > maxL {
		idx.entryPoint = internalID
		idx.maxLevel = level
	}

	idx.extToInt[externalID] = internalID
	idx.intToExt = append(idx.intToExt, externalID)

	return internalID, nil
}

// addLinkLocked adds a bidirectional link between a and b at the given level,
// avoiding duplicate entries. Caller must hold idx.mu (write lock).
func addLinkLocked(a, b *node, level int) {
	if !containsUint32(a.neighbors[level], b.id) {
		a.neighbors[level] = append(a.neighbors[level], b.id)
	}
	if !containsUint32(b.neighbors[level], a.id) {
		b.neighbors[level] = append(b.neighbors[level], a.id)
	}
}

// removeLinkLocked removes b from a's neighbor list at the given level.
// Caller must hold idx.mu (write lock).
func removeLinkLocked(a *node, b uint32, level int) {
	a.neighbors[level] = removeUint32(a.neighbors[level], b)
}

// neighborCandidates builds item list (id, distance from v) for each id.
// Caller must hold idx.mu.
func neighborCandidates(v []float32, ids []uint32, nodes []*node, dist DistanceFunc) []item {
	res := make([]item, 0, len(ids))
	for _, id := range ids {
		res = append(res, item{id: id, dist: dist(v, nodes[id].vector)})
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
