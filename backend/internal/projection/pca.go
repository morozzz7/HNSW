package projection

import (
	"errors"
	"math"
	"sort"
	"sync"
)

// PCA2D projects vectors onto their first two principal components.
// It returns one [2]float64 coordinate per vector (x = projection on PC1,
// y = projection on PC2). Coordinates are centered (zero mean).
func PCA2D(vectors [][]float32) ([][2]float64, error) {
	n := len(vectors)
	if n == 0 {
		return nil, errors.New("projection: no vectors")
	}
	dim := len(vectors[0])
	if dim == 0 {
		return nil, errors.New("projection: zero-dimensional vector")
	}
	for _, v := range vectors {
		if len(v) != dim {
			return nil, errors.New("projection: inconsistent vector dimension")
		}
	}

	// 1. Column means.
	mean := make([]float64, dim)
	for _, v := range vectors {
		for j, x := range v {
			mean[j] += float64(x)
		}
	}
	for j := range mean {
		mean[j] /= float64(n)
	}

	// 2. Centered data.
	centered := make([][]float64, n)
	for i, v := range vectors {
		c := make([]float64, dim)
		for j, x := range v {
			c[j] = float64(x) - mean[j]
		}
		centered[i] = c
	}

	// 3. Covariance matrix (dim x dim), symmetric.
	cov := make([][]float64, dim)
	for i := range cov {
		cov[i] = make([]float64, dim)
	}
	divisor := float64(n)
	if n > 1 {
		divisor = float64(n - 1) // sample covariance
	}
	for i := 0; i < dim; i++ {
		for j := i; j < dim; j++ {
			var sum float64
			for k := 0; k < n; k++ {
				sum += centered[k][i] * centered[k][j]
			}
			val := sum / divisor
			cov[i][j] = val
			cov[j][i] = val
		}
	}

	// 4. Eigen-decomposition of the symmetric covariance matrix (Jacobi).
	_, eigenvectors := jacobiSymmetric(cov)

	// 5. Project onto the top-2 eigenvectors (largest eigenvalues).
	k := 2
	if dim < k {
		k = dim
	}
	coords := make([][2]float64, n)
	for i := 0; i < n; i++ {
		for j := 0; j < k; j++ {
			var acc float64
			ev := eigenvectors[j]
			for d := 0; d < dim; d++ {
				acc += centered[i][d] * ev[d]
			}
			if j == 0 {
				coords[i][0] = acc
			} else {
				coords[i][1] = acc
			}
		}
	}
	return coords, nil
}

// jacobiSymmetric computes the eigendecomposition of a symmetric matrix using
// the Jacobi eigenvalue algorithm. Returns eigenvalues sorted in descending
// order and eigenvectors as columns (eigenvectors[i] is the i-th eigenvector).
func jacobiSymmetric(a [][]float64) ([]float64, [][]float64) {
	n := len(a)
	// Working copy.
	b := make([][]float64, n)
	for i := range b {
		b[i] = append([]float64{}, a[i]...)
	}
	// Eigenvector accumulator (identity).
	v := make([][]float64, n)
	for i := range v {
		v[i] = make([]float64, n)
		v[i][i] = 1
	}
	d := make([]float64, n)
	for i := range d {
		d[i] = b[i][i]
	}
	z := make([]float64, n)

	const maxIter = 50
	for iter := 0; iter < maxIter; iter++ {
		// Off-diagonal energy.
		var sm float64
		for p := 0; p < n; p++ {
			for q := p + 1; q < n; q++ {
				sm += b[p][q] * b[p][q]
			}
		}
		if sm == 0 {
			break // converged: b is diagonal
		}

		var tresh float64
		if iter < 4 {
			tresh = 0.2 * sm / float64(n*n)
		}

		for p := 0; p < n; p++ {
			for q := p + 1; q < n; q++ {
				g := 100.0 * math.Abs(b[p][q])
				if iter > 4 && math.Abs(d[p])+g == math.Abs(d[p]) && math.Abs(d[q])+g == math.Abs(d[q]) {
					b[p][q] = 0
				} else if math.Abs(b[p][q]) > tresh {
					h := d[q] - d[p]
					var t float64
					if math.Abs(h)+g == math.Abs(h) {
						t = b[p][q] / h
					} else {
						theta := 0.5 * h / b[p][q]
						t = 1.0 / (math.Abs(theta) + math.Sqrt(1.0+theta*theta))
						if theta < 0 {
							t = -t
						}
					}
					c := 1.0 / math.Sqrt(1.0+t*t)
					s := t * c
					tau := s / (1.0 + c)
					h = t * b[p][q]
					z[p] -= h
					z[q] += h
					d[p] -= h
					d[q] += h
					b[p][q] = 0

					// Rotate rows/columns of b.
					for j := 0; j < p; j++ {
						rotateSym(b, j, p, j, q, c, s, tau)
					}
					for j := p + 1; j < q; j++ {
						rotateSym(b, p, j, j, q, c, s, tau)
					}
					for j := q + 1; j < n; j++ {
						rotateSym(b, p, j, q, j, c, s, tau)
					}
					// Rotate columns of v (eigenvectors).
					for j := 0; j < n; j++ {
						rotateSym(v, j, p, j, q, c, s, tau)
					}
				}
			}
		}
		for i := range d {
			d[i] += z[i]
			z[i] = 0
		}
	}

	// Sort eigenvalues descending, reordering eigenvector columns accordingly.
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(i, j int) bool {
		return d[order[i]] > d[order[j]]
	})
	sortedD := make([]float64, n)
	sortedV := make([][]float64, n)
	for newIdx, oldIdx := range order {
		sortedD[newIdx] = d[oldIdx]
		sortedV[newIdx] = v[oldIdx]
	}
	return sortedD, sortedV
}

// rotateSym applies the Jacobi rotation to entries a[i][j] and a[k][l].
func rotateSym(a [][]float64, i, j, k, l int, c, s, tau float64) {
	g := a[i][j]
	h := a[k][l]
	a[i][j] = g - s*(h+g*tau)
	a[k][l] = h + s*(g-h*tau)
}

// Cache is a small thread-safe cache of PCA projections keyed by dataset id.
// It is populated after an index is built (Stage 5 wires it in) and consumed
// by the /graph endpoint for 2D coordinates.
type Cache struct {
	mu    sync.RWMutex
	items map[string][][2]float64
}

// NewCache creates an empty PCA cache.
func NewCache() *Cache {
	return &Cache{items: make(map[string][][2]float64)}
}

// Get returns the cached coordinates for key, or nil if absent.
func (c *Cache) Get(key string) ([][2]float64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.items[key]
	return v, ok
}

// Set stores coordinates under key.
func (c *Cache) Set(key string, coords [][2]float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = coords
}

// Delete removes the entry for key.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}
