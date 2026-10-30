package dataset

import (
	"errors"
	"fmt"
	"math/rand"
)

func Generate(n, dim int, distribution string, seed int64) ([][]float32, error) {
	if n <= 0 {
		return nil, errors.New("generate: n must be positive")
	}
	if dim <= 0 {
		return nil, errors.New("generate: dim must be positive")
	}
	r := rand.New(rand.NewSource(seed))

	switch distribution {
	case "uniform":
		return genUniform(r, n, dim), nil
	case "gaussian":
		return genGaussian(r, n, dim), nil
	case "clusters":
		return genClusters(r, n, dim), nil
	default:
		return nil, fmt.Errorf("generate: unknown distribution %q (want uniform, gaussian, clusters)", distribution)
	}
}

func genUniform(r *rand.Rand, n, dim int) [][]float32 {
	vecs := make([][]float32, n)
	for i := 0; i < n; i++ {
		v := make([]float32, dim)
		for j := 0; j < dim; j++ {
			v[j] = float32(r.Float64())
		}
		vecs[i] = v
	}
	return vecs
}

func genGaussian(r *rand.Rand, n, dim int) [][]float32 {
	vecs := make([][]float32, n)
	for i := 0; i < n; i++ {
		v := make([]float32, dim)
		for j := 0; j < dim; j++ {
			v[j] = float32(r.NormFloat64())
		}
		vecs[i] = v
	}
	return vecs
}

func genClusters(r *rand.Rand, n, dim int) [][]float32 {
	clusterCount := 4
	if n < clusterCount {
		clusterCount = n
	}
	centers := make([][]float32, clusterCount)
	for c := 0; c < clusterCount; c++ {
		centers[c] = genUniform(r, 1, dim)[0]
	}

	vecs := make([][]float32, n)
	for i := 0; i < n; i++ {
		center := centers[i%clusterCount]
		v := make([]float32, dim)
		for j := 0; j < dim; j++ {
			v[j] = center[j] + float32(0.05*r.NormFloat64())
		}
		vecs[i] = v
	}
	return vecs
}
