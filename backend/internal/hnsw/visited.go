package hnsw

import "sync"

type visitedSet struct {
	marks []int32
	gen   int32
}

var visitedPool = sync.Pool{
	New: func() any { return &visitedSet{} },
}

func acquireVisitedSet(n int) *visitedSet {
	v := visitedPool.Get().(*visitedSet)
	if v.marks == nil || len(v.marks) < n {
		v.marks = make([]int32, n)
	}
	v.gen++
	if v.gen == 0 {
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

func (v *visitedSet) contains(id uint32) bool {
	return int(id) < len(v.marks) && v.marks[id] == v.gen
}

func (v *visitedSet) add(id uint32) {
	v.marks[id] = v.gen
}
