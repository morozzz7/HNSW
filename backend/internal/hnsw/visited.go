package hnsw

import "sync"

// visitedSet marks nodes visited during a single search using a generation
// counter. Marks from previous searches are implicitly invalid because the
// generation changed, avoiding an O(n) clear between searches.
type visitedSet struct {
	marks []int32 // marks[id] == gen means id was visited in the current generation
	gen   int32   // current generation; incremented on each reset
}

var visitedPool = sync.Pool{
	New: func() any { return &visitedSet{} },
}

// acquireVisitedSet returns a visitedSet sized for n nodes. The returned set
// must be released with releaseVisitedSet when done.
func acquireVisitedSet(n int) *visitedSet {
	v := visitedPool.Get().(*visitedSet)
	if v.marks == nil || len(v.marks) < n {
		v.marks = make([]int32, n)
	}
	v.gen++
	if v.gen == 0 {
		// Wrapped around: invalidate all marks explicitly.
		v.gen = 1
		for i := range v.marks {
			v.marks[i] = 0
		}
	}
	return v
}

func releaseVisitedSet(v *visitedSet) {
	visitedPool.Put(v)
}

// contains reports whether id was visited in the current generation.
func (v *visitedSet) contains(id uint32) bool {
	return int(id) < len(v.marks) && v.marks[id] == v.gen
}

// add marks id as visited in the current generation.
func (v *visitedSet) add(id uint32) {
	v.marks[id] = v.gen
}
