package bruteforce

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

type DistFunc = func(a, b []float32) float32

type Result struct {
	ID   uint32
	Dist float32
}

type Index struct {
	vectors [][]float32
	dim     int
	dist    DistFunc
	workers int
}

func New(vectors [][]float32, dist DistFunc) (*Index, error) {
	return newIndex(vectors, dist, 0)
}

func NewWithWorkers(vectors [][]float32, dist DistFunc, workers int) (*Index, error) {
	if workers < 1 {
		workers = 1
	}
	return newIndex(vectors, dist, workers)
}

func newIndex(vectors [][]float32, dist DistFunc, workers int) (*Index, error) {
	if len(vectors) == 0 {
		return nil, errors.New("bruteforce: no vectors")
	}
	if dist == nil {
		return nil, errors.New("bruteforce: nil distance function")
	}
	dim := len(vectors[0])
	if dim == 0 {
		return nil, errors.New("bruteforce: zero-dimensional vector")
	}
	for i, v := range vectors {
		if len(v) != dim {
			return nil, fmt.Errorf("bruteforce: vector %d has dimension %d, want %d", i, len(v), dim)
		}
	}
	if workers > len(vectors) {
		workers = len(vectors)
	}
	return &Index{vectors: vectors, dim: dim, dist: dist, workers: workers}, nil
}

func (ix *Index) Len() int { return len(ix.vectors) }

func (ix *Index) Dim() int { return ix.dim }

func (ix *Index) Search(q []float32, k int) ([]Result, error) {
	if len(q) != ix.dim {
		return nil, fmt.Errorf("bruteforce: query dimension %d does not match index dimension %d", len(q), ix.dim)
	}
	if k <= 0 {
		return []Result{}, nil
	}

	n := len(ix.vectors)
	if k > n {
		k = n
	}

	if ix.workers <= 1 || n == 1 {
		return ix.scan(q, 0, n, k), nil
	}

	chunk := (n + ix.workers - 1) / ix.workers
	parts := make([][]Result, 0, ix.workers)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for start := 0; start < n; start += chunk {
		end := start + chunk
		if end > n {
			end = n
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			res := ix.scan(q, start, end, k)
			mu.Lock()
			parts = append(parts, res)
			mu.Unlock()
		}(start, end)
	}
	wg.Wait()

	all := make([]Result, 0, len(parts)*k)
	for _, p := range parts {
		all = append(all, p...)
	}
	sortResults(all)
	if len(all) > k {
		all = all[:k:k]
	}
	return all, nil
}

func (ix *Index) scan(q []float32, start, end, k int) []Result {
	res := make([]Result, 0, end-start)
	for i := start; i < end; i++ {
		res = append(res, Result{ID: uint32(i), Dist: ix.dist(q, ix.vectors[i])})
	}
	sortResults(res)
	if len(res) > k {
		res = res[:k:k]
	}
	return res
}

func sortResults(r []Result) {
	sort.Slice(r, func(i, j int) bool {
		if r[i].Dist != r[j].Dist {
			return r[i].Dist < r[j].Dist
		}
		return r[i].ID < r[j].ID
	})
}

func Recall(exact, approx []Result, k int) float64 {
	if k <= 0 || len(exact) == 0 {
		return 0
	}

	exactTop := exact
	if len(exactTop) > k {
		exactTop = exactTop[:k]
	}
	want := make(map[uint32]struct{}, len(exactTop))
	for _, r := range exactTop {
		want[r.ID] = struct{}{}
	}

	approxTop := approx
	if len(approxTop) > k {
		approxTop = approxTop[:k]
	}
	found := 0
	seen := make(map[uint32]struct{}, len(approxTop))
	for _, r := range approxTop {
		if _, ok := want[r.ID]; !ok {
			continue
		}
		if _, dup := seen[r.ID]; dup {
			continue
		}
		seen[r.ID] = struct{}{}
		found++
	}

	return float64(found) / float64(len(exactTop))
}
