package hnsw

type LayerTrace struct {
	Level   int
	Visited []uint32
	Closest uint32
}

type Trace struct {
	Layers []LayerTrace
}
