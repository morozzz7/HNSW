package api

import (
	"errors"
	"fmt"
	"net/http"

	"hnsw-coursework/internal/hnsw"
	"hnsw-coursework/internal/indexmanager"
)

type buildIndexRequest struct {
	DatasetID      string `json:"datasetId"`
	M              int    `json:"M"`
	EfConstruction int    `json:"efConstruction"`
	Metric         string `json:"metric"`
	Heuristic      *bool  `json:"heuristic"`
	Seed           *int64 `json:"seed"`
}

type buildIndexResponse struct {
	IndexID string `json:"indexId"`
	Status  string `json:"status"`
}

type indexStatusResponse struct {
	Status    string  `json:"status"`
	Progress  float64 `json:"progress"`
	ElapsedMs int64   `json:"elapsedMs"`
	IndexID   string  `json:"indexId"`
	Error     *string `json:"error"`
}

type degreeDistributionResponse struct {
	Min  int     `json:"min"`
	Max  int     `json:"max"`
	Mean float64 `json:"mean"`
	P50  int     `json:"p50"`
}

type indexStatsResponse struct {
	Nodes              int                        `json:"nodes"`
	MaxLevel           int                        `json:"maxLevel"`
	EntryPoint         uint32                     `json:"entryPoint"`
	NodesPerLevel      []int                      `json:"nodesPerLevel"`
	DegreeDistribution degreeDistributionResponse `json:"degreeDistribution"`
	Metric             string                     `json:"metric"`
	M                  int                        `json:"M"`
	EfConstruction     int                        `json:"efConstruction"`
	IndexID            string                     `json:"indexId,omitempty"`
	DatasetID          string                     `json:"datasetId,omitempty"`
}

func (s *Server) handleBuildIndex(w http.ResponseWriter, r *http.Request) {
	var req buildIndexRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, err.Error())
		return
	}
	if req.DatasetID == "" {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, "datasetId is required")
		return
	}

	// Resolve documented defaults for omitted zero values.
	if req.M < 0 {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, "M must be positive")
		return
	}
	if req.EfConstruction < 0 {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, "efConstruction must be positive")
		return
	}
	if req.M == 0 {
		req.M = 16
	}
	if req.EfConstruction == 0 {
		req.EfConstruction = 200
	}
	metric := hnsw.MetricL2
	if req.Metric != "" {
		metric = hnsw.Metric(req.Metric)
	}
	useHeuristic := true
	if req.Heuristic != nil {
		useHeuristic = *req.Heuristic
	}
	var seed int64
	if req.Seed != nil {
		seed = *req.Seed
	}

	indexID, err := s.mgr.StartBuild(indexmanager.BuildConfig{
		DatasetID:      req.DatasetID,
		M:              req.M,
		EfConstruction: req.EfConstruction,
		Metric:         metric,
		UseHeuristic:   useHeuristic,
		Seed:           seed,
	})
	if err != nil {
		switch {
		case errors.Is(err, indexmanager.ErrDatasetNotFound):
			writeError(w, http.StatusNotFound, codeDatasetNotFound,
				fmt.Sprintf("dataset %q not found", req.DatasetID))
		case errors.Is(err, indexmanager.ErrBuildInProgress):
			writeError(w, http.StatusConflict, codeIndexNotReady, "an index build is already in progress")
		case errors.Is(err, indexmanager.ErrInvalid):
			writeError(w, http.StatusBadRequest, codeInvalidRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, codeBuildFailed, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusAccepted, buildIndexResponse{IndexID: indexID, Status: string(indexmanager.StatusBuilding)})
}

func (s *Server) handleIndexStatus(w http.ResponseWriter, _ *http.Request) {
	st := s.mgr.Status()
	resp := indexStatusResponse{
		Status:    string(st.Status),
		Progress:  st.Progress,
		ElapsedMs: st.ElapsedMs,
		IndexID:   st.IndexID,
	}
	if st.Status == indexmanager.StatusError && st.Error != "" {
		msg := st.Error
		resp.Error = &msg
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleIndexStats(w http.ResponseWriter, _ *http.Request) {
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
		writeError(w, http.StatusInternalServerError, codeBuildFailed, msg)
		return
	}

	stats, err := s.mgr.Stats()
	if err != nil {
		writeManagerError(w, err, codeIndexNotReady, http.StatusConflict)
		return
	}
	writeJSON(w, http.StatusOK, indexStatsResponse{
		Nodes:         stats.Nodes,
		MaxLevel:      stats.MaxLevel,
		EntryPoint:    stats.EntryPoint,
		NodesPerLevel: stats.NodesPerLevel,
		DegreeDistribution: degreeDistributionResponse{
			Min:  stats.DegreeDistribution.Min,
			Max:  stats.DegreeDistribution.Max,
			Mean: stats.DegreeDistribution.Mean,
			P50:  stats.DegreeDistribution.P50,
		},
		Metric:         string(stats.Metric),
		M:              stats.M,
		EfConstruction: stats.EfConstruction,
		IndexID:        stats.IndexID,
		DatasetID:      stats.DatasetID,
	})
}
