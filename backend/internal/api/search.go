package api

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"hnsw-coursework/internal/hnsw"
	"hnsw-coursework/internal/indexmanager"
)

type searchRequest struct {
	Vector   []float32 `json:"vector"`
	VectorID *FlexID   `json:"vectorId"`
	K        int       `json:"k"`
	EfSearch int       `json:"efSearch"`
	Mode     string    `json:"mode"`
	Trace    bool      `json:"trace"`
}

type searchResultResponse struct {
	ID       uint32    `json:"id"`
	VectorID string    `json:"vectorId"`
	Distance float64   `json:"distance"`
	Vector   []float32 `json:"vector"`
}

type traceLayerResponse struct {
	Level   int      `json:"level"`
	Visited []uint32 `json:"visited"`
	Closest uint32   `json:"closest"`
}

type traceResponse struct {
	Layers []traceLayerResponse `json:"layers"`
}

type searchResponse struct {
	Results []searchResultResponse `json:"results"`
	Found   int                    `json:"found"`
	Trace   *traceResponse         `json:"trace,omitempty"`
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	var req searchRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, err.Error())
		return
	}

	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = "hnsw"
	}
	if mode != "hnsw" && mode != "bruteforce" {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, `mode must be "hnsw" or "bruteforce"`)
		return
	}
	if req.K <= 0 {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, "k must be positive")
		return
	}
	if req.EfSearch < 0 {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, "efSearch must not be negative")
		return
	}

	hasVector := len(req.Vector) > 0
	hasVectorID := req.VectorID != nil && req.VectorID.String() != ""
	if hasVector && hasVectorID {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, "provide either vector or vectorId, not both")
		return
	}
	if !hasVector && !hasVectorID {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, "either vector or vectorId is required")
		return
	}

	st := s.mgr.Status()
	switch st.Status {
	case indexmanager.StatusIdle:
		writeError(w, http.StatusNotFound, codeIndexNotFound, "no index has been built yet")
		return
	case indexmanager.StatusBuilding:
		writeError(w, http.StatusConflict, codeIndexNotReady, "index is still building")
		return
	case indexmanager.StatusError:
		msg := st.Error
		if msg == "" {
			msg = "index build failed"
		}
		writeError(w, http.StatusConflict, codeIndexNotReady, msg)
		return
	}

	n := s.mgr.Len()
	if req.K > n {
		writeError(w, http.StatusBadRequest, codeKExceedsN,
			fmt.Sprintf("k=%d exceeds the number of indexed vectors %d", req.K, n))
		return
	}

	var query []float32
	if hasVectorID {
		q, ok := s.mgr.VectorByID(req.VectorID.String())
		if !ok {
			writeError(w, http.StatusBadRequest, codeInvalidRequest,
				fmt.Sprintf("unknown vectorId %q", req.VectorID.String()))
			return
		}
		query = q
	} else {
		query = req.Vector
	}
	if dim := s.mgr.Dim(); len(query) != dim {
		writeError(w, http.StatusBadRequest, codeInvalidDimension,
			fmt.Sprintf("query dimension %d does not match index dimension %d", len(query), dim))
		return
	}

	ef := req.EfSearch
	if ef < req.K {
		ef = req.K
	}
	metric := s.mgr.Metric()

	var (
		results []indexmanager.SearchResult
		trace   *hnsw.Trace
		err     error
	)
	if mode == "bruteforce" {
		results, err = s.mgr.BruteForceSearch(query, req.K)
	} else {
		results, trace, err = s.mgr.HNSWSearch(query, req.K, ef, req.Trace)
	}
	if err != nil {
		if errors.Is(err, indexmanager.ErrNoIndex) ||
			errors.Is(err, indexmanager.ErrNotReady) ||
			errors.Is(err, indexmanager.ErrDimension) {
			writeManagerError(w, err, codeSearchFailed, http.StatusInternalServerError)
			return
		}
		writeError(w, http.StatusInternalServerError, codeSearchFailed, err.Error())
		return
	}

	out := make([]searchResultResponse, len(results))
	for i, res := range results {
		out[i] = searchResultResponse{
			ID:       res.Internal,
			VectorID: res.VectorID,
			Distance: outputDistance(metric, res.Distance),
			Vector:   res.Vector,
		}
	}
	resp := searchResponse{Results: out, Found: len(out), Trace: mapTrace(trace)}
	writeJSON(w, http.StatusOK, resp)
}

// outputDistance converts an internal distance to the API representation:
//   - l2: sqrt(squared L2) => actual Euclidean distance;
//   - cosine: already 1 - cosine similarity;
//   - inner_product: negative dot product (smaller means closer).
func outputDistance(metric hnsw.Metric, d float32) float64 {
	if metric == hnsw.MetricL2 {
		return math.Sqrt(float64(d))
	}
	return float64(d)
}

// mapTrace converts the core trace to the API shape. It returns nil when the
// core returned no trace (e.g. brute-force mode).
func mapTrace(tr *hnsw.Trace) *traceResponse {
	if tr == nil {
		return nil
	}
	layers := make([]traceLayerResponse, len(tr.Layers))
	for i, lt := range tr.Layers {
		visited := make([]uint32, len(lt.Visited))
		copy(visited, lt.Visited)
		layers[i] = traceLayerResponse{
			Level:   lt.Level,
			Visited: visited,
			Closest: lt.Closest,
		}
	}
	return &traceResponse{Layers: layers}
}
