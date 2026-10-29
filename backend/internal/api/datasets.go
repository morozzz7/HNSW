package api

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"hnsw-coursework/internal/dataset"
)

// maxGenerateValues caps the number of float components produced by a single
// generate request to avoid accidental OOM (50M float32 ~= 200 MiB).
const maxGenerateValues = 50_000_000

type createDatasetResponse struct {
	ID      string      `json:"id"`
	N       int         `json:"n"`
	Dim     int         `json:"dim"`
	Preview [][]float32 `json:"preview"`
}

type datasetListItem struct {
	ID        string    `json:"id"`
	N         int       `json:"n"`
	Dim       int       `json:"dim"`
	CreatedAt time.Time `json:"createdAt"`
}

type listDatasetsResponse struct {
	Datasets []datasetListItem `json:"datasets"`
}

type generateRequest struct {
	Action       string `json:"action"`
	N            int    `json:"n"`
	Dim          int    `json:"dim"`
	Distribution string `json:"distribution"`
	Seed         int64  `json:"seed"`
}

func (s *Server) handleCreateDataset(w http.ResponseWriter, r *http.Request) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidRequest,
			"Content-Type is required (multipart/form-data or application/json)")
		return
	}

	switch mediaType {
	case "multipart/form-data":
		s.createDatasetFromUpload(w, r)
	case "application/json":
		s.createDatasetFromGenerate(w, r)
	default:
		writeError(w, http.StatusBadRequest, codeInvalidRequest,
			"Content-Type must be multipart/form-data or application/json")
	}
}

func (s *Server) createDatasetFromUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, codeFileTooLarge,
				fmt.Sprintf("upload exceeds %d bytes", int64(maxUploadBytes)))
			return
		}
		writeError(w, http.StatusBadRequest, codeInvalidRequest, "invalid multipart form: "+err.Error())
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, `missing form field "file"`)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	var vectors [][]float32
	switch ext {
	case ".csv":
		vectors, err = dataset.LoadCSV(file)
	case ".json":
		vectors, err = dataset.LoadJSON(file)
	default:
		writeError(w, http.StatusUnsupportedMediaType, codeUnsupportedFileType,
			fmt.Sprintf("unsupported file type %q (want .csv or .json)", ext))
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, "load dataset: "+err.Error())
		return
	}

	source := strings.TrimPrefix(ext, ".")
	ds, err := dataset.NewDataset(source, "", vectors)
	if err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, err.Error())
		return
	}
	s.store.Add(ds)
	writeJSON(w, http.StatusCreated, createDatasetResponse{
		ID: ds.ID, N: ds.N, Dim: ds.Dim, Preview: ds.Preview,
	})
}

func (s *Server) createDatasetFromGenerate(w http.ResponseWriter, r *http.Request) {
	var req generateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, err.Error())
		return
	}
	if req.Action != "generate" {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, `action must be "generate"`)
		return
	}
	if req.Distribution == "" {
		req.Distribution = "uniform"
	}
	switch req.Distribution {
	case "uniform", "gaussian", "clusters":
	default:
		writeError(w, http.StatusBadRequest, codeInvalidRequest,
			fmt.Sprintf("unknown distribution %q (want uniform, gaussian, clusters)", req.Distribution))
		return
	}
	if req.N <= 0 {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, "n must be positive")
		return
	}
	if req.Dim <= 0 {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, "dim must be positive")
		return
	}
	if req.N > dataset.MaxVectors {
		writeError(w, http.StatusBadRequest, codeInvalidRequest,
			fmt.Sprintf("n=%d exceeds limit %d", req.N, dataset.MaxVectors))
		return
	}
	if req.Dim > dataset.MaxDim {
		writeError(w, http.StatusBadRequest, codeInvalidRequest,
			fmt.Sprintf("dim=%d exceeds limit %d", req.Dim, dataset.MaxDim))
		return
	}
	if int64(req.N)*int64(req.Dim) > maxGenerateValues {
		writeError(w, http.StatusBadRequest, codeInvalidRequest,
			fmt.Sprintf("n*dim=%d exceeds generation limit %d", int64(req.N)*int64(req.Dim), int64(maxGenerateValues)))
		return
	}

	vectors, err := dataset.Generate(req.N, req.Dim, req.Distribution, req.Seed)
	if err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, err.Error())
		return
	}
	ds, err := dataset.NewDataset("generate", req.Distribution, vectors)
	if err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, err.Error())
		return
	}
	s.store.Add(ds)
	writeJSON(w, http.StatusCreated, createDatasetResponse{
		ID: ds.ID, N: ds.N, Dim: ds.Dim, Preview: ds.Preview,
	})
}

func (s *Server) handleListDatasets(w http.ResponseWriter, _ *http.Request) {
	items := s.store.List()
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	out := make([]datasetListItem, len(items))
	for i, ds := range items {
		out[i] = datasetListItem{ID: ds.ID, N: ds.N, Dim: ds.Dim, CreatedAt: ds.CreatedAt}
	}
	writeJSON(w, http.StatusOK, listDatasetsResponse{Datasets: out})
}

func (s *Server) handleGetDataset(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ds := s.store.Get(id)
	if ds == nil {
		writeError(w, http.StatusNotFound, codeDatasetNotFound, fmt.Sprintf("dataset %q not found", id))
		return
	}
	writeJSON(w, http.StatusOK, ds)
}
