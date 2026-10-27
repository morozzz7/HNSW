// Package bruteforce provides an exact k-NN search over a fixed set of vectors
// by scanning all of them (linear scan, optionally parallel). It is used as the
// ground truth for measuring recall of the approximate HNSW index and as a
// baseline for speed comparisons.
package bruteforce

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

// DistFunc is the signature of a distance function: smaller values mean closer
// vectors. It is a type alias, so hnsw.DistanceFunc values can be passed
// directly. Both arguments must have the same length.
type DistFunc = func(a, b []float32) float32

// Result is a single exact nearest-neighbor hit. ID is the vector's row index
// in the set given to New.
type Result struct {
	ID   uint32
	Dist float32
}

// Index performs exact k-NN search by scanning all vectors. It is safe for
// concurrent searches; the underlying vectors must not be modified after New.
type Index struct {
	vectors [][]float32
	dim     int
	dist    DistFunc
	workers int // number of parallel scan goroutines; 0 or 1 = sequential
}

// New creates a sequential exact-search index over vectors. dist must be a
// distance function consistent with the one used by the approximate index.
func New(vectors [][]float32, dist DistFunc) (*Index, error) {
	return newIndex(vectors, dist, 0)
}

// NewWithWorkers creates an exact-search index that scans vectors in parallel
// using the given number of goroutines. workers <= 1 means sequential scanning.
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

// Len returns the number of indexed vectors.
func (ix *Index) Len() int { return len(ix.vectors) }

// Dim returns the vector dimension.
func (ix *Index) Dim() int { return ix.dim }

// Search returns the k vectors closest to q, sorted by distance ascending (ties
// broken by id). If k is larger than the index, all vectors are returned; if
// k <= 0 the result is empty.
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

// scan computes distances for vectors in [start, end) and returns the best k
// of them, sorted by distance ascending. It is the inner loop of the linear scan.
func (ix *Index) scan(q []float32, start, end, k int) []Result {
	res := make([]Result, 0, end-start)
	for i := start; i < end; i++ {
		res = append(res, Result{ID: uint32(i), Dist: ix.dist(q, ix.vectors[i])})
	}
	sortResults(res)
	if len(res) > k {
		res = res[:k:k] // cap capacity so callers do not retain the whole scan buffer
	}
	return res
}

// sortResults sorts by distance ascending, breaking ties by id for deterministic
// output (important when comparing approximate results with exact ones).
func sortResults(r []Result) {
	sort.Slice(r, func(i, j int) bool {
		if r[i].Dist != r[j].Dist {
			return r[i].Dist < r[j].Dist
		}
		return r[i].ID < r[j].ID
	})
}

// Recall computes recall@k: the fraction of the exact top-k results that also
// appear in the approximate top-k. Ids are compared as sets, so result order
// does not matter. The denominator is min(k, len(exact)); both lists must be
// sorted by distance ascending. Returns a value in [0, 1].
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
