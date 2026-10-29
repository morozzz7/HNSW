// Package api implements the HTTP API described in docs/API.md.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"hnsw-coursework/internal/dataset"
	"hnsw-coursework/internal/indexmanager"
)

const (
	basePath      = "/api/v1"
	allowedOrigin = "http://localhost:5173"

	maxJSONBodyBytes = 8 << 20   // 8 MiB for JSON bodies
	maxUploadBytes   = 256 << 20 // 256 MiB for dataset uploads
)

// API error codes (see docs/API.md).
const (
	codeInvalidRequest      = "INVALID_REQUEST"
	codeDatasetNotFound     = "DATASET_NOT_FOUND"
	codeIndexNotFound       = "INDEX_NOT_FOUND"
	codeIndexNotReady       = "INDEX_NOT_READY"
	codeInvalidDimension    = "INVALID_DIMENSION"
	codeKExceedsN           = "K_EXCEEDS_N"
	codeBuildFailed         = "BUILD_FAILED"
	codeSearchFailed        = "SEARCH_FAILED"
	codeFileTooLarge        = "FILE_TOO_LARGE"
	codeUnsupportedFileType = "UNSUPPORTED_FILE_TYPE"
	codeNotFound            = "NOT_FOUND"
	codeMethodNotAllowed    = "METHOD_NOT_ALLOWED"
	codeInternalError       = "INTERNAL_ERROR"
)

// apiError is the inner object of the unified error envelope.
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorEnvelope struct {
	Error apiError `json:"error"`
}

// Server wires HTTP handlers to the dataset store and index manager.
type Server struct {
	store *dataset.Store
	mgr   *indexmanager.Manager
	mux   *http.ServeMux
}

// NewServer creates the API server and registers all routes.
func NewServer(store *dataset.Store, mgr *indexmanager.Manager) *Server {
	s := &Server{
		store: store,
		mgr:   mgr,
		mux:   http.NewServeMux(),
	}
	s.routes()
	return s
}

// Handler returns the fully wrapped HTTP handler (CORS, panic recovery).
func (s *Server) Handler() http.Handler {
	return s.withMiddleware(s.mux)
}

// Routes is an alias for Handler, kept for callers that prefer routing
// terminology.
func (s *Server) Routes() http.Handler { return s.Handler() }

func (s *Server) routes() {
	s.mux.HandleFunc("GET "+basePath+"/health", s.handleHealth)
	s.mux.HandleFunc("POST "+basePath+"/datasets", s.handleCreateDataset)
	s.mux.HandleFunc("GET "+basePath+"/datasets", s.handleListDatasets)
	s.mux.HandleFunc("GET "+basePath+"/datasets/{id}", s.handleGetDataset)
	s.mux.HandleFunc("POST "+basePath+"/index", s.handleBuildIndex)
	s.mux.HandleFunc("GET "+basePath+"/index/status", s.handleIndexStatus)
	s.mux.HandleFunc("GET "+basePath+"/index/events", s.handleIndexEvents)
	s.mux.HandleFunc("GET "+basePath+"/index/stats", s.handleIndexStats)
	s.mux.HandleFunc("POST "+basePath+"/vectors", s.handleInsertVector)
	s.mux.HandleFunc("POST "+basePath+"/search", s.handleSearch)

	// Method-less patterns act as fallbacks for wrong methods on known paths.
	// Go 1.22 ServeMux prefers the method-specific pattern when it matches.
	for _, path := range []string{
		basePath + "/health",
		basePath + "/datasets",
		basePath + "/datasets/{id}",
		basePath + "/index",
		basePath + "/index/status",
		basePath + "/index/events",
		basePath + "/index/stats",
		basePath + "/vectors",
		basePath + "/search",
	} {
		s.mux.HandleFunc(path, s.handleMethodNotAllowed)
	}
	s.mux.HandleFunc("/", s.handleNotFound)
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("api: panic serving %s %s: %v", r.Method, r.URL.Path, rec)
				writeError(w, http.StatusInternalServerError, codeInternalError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleNotFound(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotFound, codeNotFound, "route not found")
}

func (s *Server) handleMethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusMethodNotAllowed, codeMethodNotAllowed, "method not allowed")
}

// writeJSON marshals v and writes it with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		log.Printf("api: marshal response: %v", err)
		body = []byte(`{"error":{"code":"INTERNAL_ERROR","message":"failed to encode response"}}`)
		status = http.StatusInternalServerError
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// writeError writes the unified error envelope.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorEnvelope{Error: apiError{Code: code, Message: message}})
}

// decodeJSON decodes a single JSON value from the request body and rejects
// trailing data. It returns a plain error; handlers decide the API code.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if r.Body == nil {
		return errors.New("request body is empty")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("invalid JSON body: unexpected trailing data")
	}
	return nil
}

// FlexID accepts a JSON string or number for vector ids. Responses always use
// strings, but the contract example shows a numeric vectorId in /search.
type FlexID string

// UnmarshalJSON implements json.Unmarshaler.
func (id *FlexID) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*id = ""
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*id = FlexID(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return errors.New("vectorId must be a string or a number")
	}
	*id = FlexID(n.String())
	return nil
}

// String returns the id as a plain string.
func (id FlexID) String() string { return string(id) }

// writeManagerError maps manager sentinel errors to API error responses.
func writeManagerError(w http.ResponseWriter, err error, fallbackCode string, fallbackStatus int) {
	switch {
	case errors.Is(err, indexmanager.ErrNoIndex):
		writeError(w, http.StatusNotFound, codeIndexNotFound, "no index has been built yet")
	case errors.Is(err, indexmanager.ErrNotReady):
		writeError(w, http.StatusConflict, codeIndexNotReady, "index is not ready")
	case errors.Is(err, indexmanager.ErrDatasetNotFound):
		writeError(w, http.StatusNotFound, codeDatasetNotFound, "dataset not found")
	case errors.Is(err, indexmanager.ErrDimension):
		writeError(w, http.StatusBadRequest, codeInvalidDimension, "vector dimension does not match the index")
	case errors.Is(err, indexmanager.ErrDuplicateID):
		writeError(w, http.StatusBadRequest, codeInvalidRequest, err.Error())
	case errors.Is(err, indexmanager.ErrBuildInProgress):
		writeError(w, http.StatusConflict, codeIndexNotReady, "an index build is already in progress")
	case errors.Is(err, indexmanager.ErrInvalid):
		writeError(w, http.StatusBadRequest, codeInvalidRequest, err.Error())
	default:
		writeError(w, fallbackStatus, fallbackCode, err.Error())
	}
}
