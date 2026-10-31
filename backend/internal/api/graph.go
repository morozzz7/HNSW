package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"hnsw-coursework/internal/indexmanager"
)

type graphVertexResponse struct {
	ID           uint32  `json:"id"`
	VectorID     string  `json:"vectorId"`
	Level        int     `json:"level"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	IsEntryPoint bool    `json:"isEntryPoint"`
}

type graphEdgeResponse struct {
	From  uint32 `json:"from"`
	To    uint32 `json:"to"`
	Level int    `json:"level"`
}
type graphResponse struct {
	Level         int                   `json:"level"`
	Limit         int                   `json:"limit"`
	Vertices      []graphVertexResponse `json:"vertices"`
	Edges         []graphEdgeResponse   `json:"edges"`
	TotalVertices int                   `json:"totalVertices"`
}

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	level, limit := 0, 500
	var err error
	if q.Get("level") != "" {
		level, err = strconv.Atoi(q.Get("level"))
		if err != nil {
			writeError(w, http.StatusBadRequest, codeInvalidRequest, "level must be an integer")
			return
		}
	}
	if q.Get("limit") != "" {
		limit, err = strconv.Atoi(q.Get("limit"))
		if err != nil {
			writeError(w, http.StatusBadRequest, codeInvalidRequest, "limit must be an integer")
			return
		}
	}
	if limit < 1 || limit > 5000 {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, "limit must be between 1 and 5000")
		return
	}
	g, err := s.mgr.Graph(level, limit, q.Get("around"))
	if err != nil {
		if errors.Is(err, indexmanager.ErrNoIndex) {
			writeError(w, http.StatusNotFound, codeIndexNotFound, "no index has been built yet")
			return
		}
		if errors.Is(err, indexmanager.ErrNotReady) {
			writeError(w, http.StatusConflict, codeIndexNotReady, "index is not ready")
			return
		}
		writeError(w, http.StatusBadRequest, codeInvalidRequest, fmt.Sprint(err))
		return
	}
	resp := graphResponse{Level: g.Level, Limit: limit, TotalVertices: g.TotalVertices, Vertices: make([]graphVertexResponse, len(g.Vertices)), Edges: make([]graphEdgeResponse, len(g.Edges))}
	for i, v := range g.Vertices {
		resp.Vertices[i] = graphVertexResponse{ID: v.ID, VectorID: v.VectorID, Level: v.Level, X: v.X, Y: v.Y, IsEntryPoint: v.IsEntryPoint}
	}
	for i, e := range g.Edges {
		resp.Edges[i] = graphEdgeResponse{From: e.From, To: e.To, Level: e.Level}
	}
	writeJSON(w, http.StatusOK, resp)
}
