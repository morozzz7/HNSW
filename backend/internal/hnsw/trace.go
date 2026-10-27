package hnsw

// LayerTrace records what happened during a single SEARCH-LAYER call.
// It is used by SearchWithTrace to visualize the per-layer search path.
type LayerTrace struct {
	Level   int      // graph layer this trace belongs to
	Visited []uint32 // sequence of node ids visited (in visit order)
	Closest uint32   // node id closest to the query at this layer
}

// Trace records the per-layer path of a k-NN search, one entry per layer.
type Trace struct {
	Layers []LayerTrace
}
