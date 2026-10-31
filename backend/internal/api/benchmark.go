package api

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"hnsw-coursework/internal/benchmark"
	"hnsw-coursework/internal/hnsw"
	"hnsw-coursework/internal/indexmanager"
)

type benchmarkRequest struct {
	IndexID        string `json:"indexId"`
	M              []int  `json:"M"`
	EfConstruction []int  `json:"efConstruction"`
	EfSearch       []int  `json:"efSearch"`
	K              int    `json:"k"`
	Queries        int    `json:"queries"`
}
type benchmarkJob struct {
	ID       string
	Status   string
	Progress float64
	Results  []benchmark.Result
	Error    string
	Created  time.Time
}
type benchmarkStore struct {
	sync.RWMutex
	jobs map[string]*benchmarkJob
	next uint64
}

func (s *Server) benchmarkStoreInit() {
	if s.bench == nil {
		s.bench = &benchmarkStore{jobs: make(map[string]*benchmarkJob)}
	}
}

func (s *Server) handleBenchmark(w http.ResponseWriter, r *http.Request) {
	s.benchmarkStoreInit()
	var req benchmarkRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, err.Error())
		return
	}
	snap, err := s.mgr.BenchmarkSnapshot()
	if err != nil {
		writeManagerError(w, err, codeSearchFailed, http.StatusConflict)
		return
	}
	if req.IndexID != "" && req.IndexID != snap.IndexID {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, "indexId does not match the current index")
		return
	}
	if req.K <= 0 {
		req.K = 10
	}
	if req.Queries <= 0 {
		req.Queries = 100
	}
	if req.K > hnsw.MaxK || req.Queries > 10000 {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, "k or queries exceeds the limit")
		return
	}
	if req.K > len(snap.Vectors) {
		writeError(w, http.StatusBadRequest, codeKExceedsN, "k exceeds the number of indexed vectors")
		return
	}
	for _, value := range req.M {
		if value < hnsw.MinM || value > hnsw.MaxM {
			writeError(w, http.StatusBadRequest, codeInvalidRequest, "M is outside the allowed range")
			return
		}
	}
	for _, value := range req.EfConstruction {
		if value < 1 || value > hnsw.MaxEfConstruction {
			writeError(w, http.StatusBadRequest, codeInvalidRequest, "efConstruction is outside the allowed range")
			return
		}
	}
	for _, value := range req.EfSearch {
		if value < 1 || value > hnsw.MaxEfSearch {
			writeError(w, http.StatusBadRequest, codeInvalidRequest, "efSearch is outside the allowed range")
			return
		}
	}
	if len(req.M)*max1(len(req.EfConstruction))*max1(len(req.EfSearch)) > 256 {
		writeError(w, http.StatusBadRequest, codeInvalidRequest, "benchmark grid is too large")
		return
	}
	s.bench.Lock()
	s.bench.next++
	id := fmt.Sprintf("bench_%x", s.bench.next)
	job := &benchmarkJob{ID: id, Status: "queued", Created: time.Now()}
	s.bench.jobs[id] = job
	if len(s.bench.jobs) > 100 {
		var oldestID string
		var oldest time.Time
		for candidateID, candidate := range s.bench.jobs {
			if candidateID == id {
				continue
			}
			if oldestID == "" || candidate.Created.Before(oldest) {
				oldestID, oldest = candidateID, candidate.Created
			}
		}
		if oldestID != "" {
			delete(s.bench.jobs, oldestID)
		}
	}
	s.bench.Unlock()
	go s.runBenchmark(job, snap, req)
	writeJSON(w, http.StatusAccepted, map[string]string{"id": id, "status": "queued"})
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

func (s *Server) runBenchmark(job *benchmarkJob, snap indexmanager.BenchmarkSnapshot, req benchmarkRequest) {
	queries := make([][]float32, 0, minInt(req.Queries, len(snap.Vectors)))
	for i := 0; i < req.Queries && i < len(snap.Vectors); i++ {
		queries = append(queries, snap.Vectors[i])
	}
	jobStatus := func(status, message string) {
		s.bench.Lock()
		job.Status, job.Error = status, message
		s.bench.Unlock()
	}
	jobStatus("running", "")
	results, err := benchmark.RunGrid(benchmark.GridConfig{Vectors: snap.Vectors, Queries: queries, M: req.M, EfConstruction: req.EfConstruction, EfSearch: req.EfSearch, Metric: []hnsw.Metric{snap.Metric}, UseHeuristic: []bool{true}, K: req.K, Seed: snap.Seed, Progress: func(done, total int) {
		s.bench.Lock()
		if total > 0 {
			job.Progress = float64(done) / float64(total)
		}
		s.bench.Unlock()
	}})
	if err != nil {
		jobStatus("error", err.Error())
		return
	}
	s.bench.Lock()
	job.Results, job.Progress, job.Status = results, 1, "done"
	s.bench.Unlock()
}

func (s *Server) handleBenchmarkStatus(w http.ResponseWriter, r *http.Request) {
	s.benchmarkStoreInit()
	id := strings.TrimPrefix(r.URL.Path, basePath+"/benchmark/")
	s.bench.RLock()
	job, ok := s.bench.jobs[id]
	var cp benchmarkJob
	if ok {
		cp = *job
		cp.Results = append([]benchmark.Result(nil), job.Results...)
	}
	s.bench.RUnlock()
	if !ok {
		writeError(w, http.StatusNotFound, codeNotFound, "benchmark job not found")
		return
	}
	resp := map[string]any{"id": cp.ID, "status": cp.Status, "progress": cp.Progress}
	if cp.Status == "error" {
		resp["error"] = cp.Error
	}
	if cp.Status == "done" {
		resp["results"] = cp.Results
	}
	writeJSON(w, http.StatusOK, resp)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
