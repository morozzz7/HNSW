package api

import (
	"net/http"

	"hnsw-coursework/internal/indexmanager"
)

type insertVectorRequest struct {
	VectorID *FlexID   `json:"vectorId"`
	Vector   []float32 `json:"vector"`
}

type insertVectorResponse struct {
	ID       uint32 `json:"id"`
	VectorID string `json:"vectorId"`
}

func (s *Server) handleInsertVector(w http.ResponseWriter, r *http.Request) {
	var req insertVectorRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, err.Error())
		return
	}
	if len(req.Vector) == 0 {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, "vector is required")
		return
	}

	st := s.mgr.Status()
	switch st.Status {
	case indexmanager.StatusIdle:
		writeError(w, http.StatusNotFound, codeIndexNotFound, "no index has been built yet")
		return
	case indexmanager.StatusBuilding, indexmanager.StatusError:
		writeError(w, http.StatusConflict, codeIndexNotReady, "index is not ready")
		return
	}

	vectorID := ""
	if req.VectorID != nil {
		vectorID = req.VectorID.String()
	}
	internal, assigned, err := s.mgr.InsertVector(req.Vector, vectorID)
	if err != nil {
		writeManagerError(w, err, codeInternalError, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, insertVectorResponse{ID: internal, VectorID: assigned})
}
