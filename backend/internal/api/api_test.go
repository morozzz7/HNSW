package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hnsw-coursework/internal/dataset"
	"hnsw-coursework/internal/indexmanager"
)

const (
	testTimeout  = 10 * time.Second
	pollInterval = 2 * time.Millisecond
	buildM       = 8
	buildEfC     = 50
	testDatasetN = 64
	testDatasetD = 8
	testSearchK  = 5
)

// newTestAPI creates an isolated store/manager/server pair.
func newTestAPI(t *testing.T) (*Server, *indexmanager.Manager) {
	t.Helper()
	store := dataset.NewStore()
	mgr := indexmanager.New(store)
	srv := NewServer(store, mgr)
	t.Cleanup(mgr.Close)
	return srv, mgr
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		r = bytes.NewReader(data)
	}
	req := httptest.NewRequest(method, path, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeMap(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return out
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) (string, string) {
	t.Helper()
	var env struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope %q: %v", rec.Body.String(), err)
	}
	return env.Error.Code, env.Error.Message
}

func expectError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, status, rec.Body.String())
	}
	gotCode, message := errorCode(t, rec)
	if gotCode != code {
		t.Fatalf("error code = %q, want %q (message: %s)", gotCode, code, message)
	}
	if message == "" {
		t.Fatal("error message is empty")
	}
}

func createGeneratedDataset(t *testing.T, h http.Handler, n, dim int) string {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, basePath+"/datasets", map[string]any{
		"action":       "generate",
		"n":            n,
		"dim":          dim,
		"distribution": "uniform",
		"seed":         1,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create dataset: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	m := decodeMap(t, rec)
	id, _ := m["id"].(string)
	if id == "" {
		t.Fatalf("create dataset: missing id in %s", rec.Body.String())
	}
	return id
}

func startBuild(t *testing.T, h http.Handler, datasetID string) string {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, basePath+"/index", map[string]any{
		"datasetId":      datasetID,
		"M":              buildM,
		"efConstruction": buildEfC,
		"metric":         "l2",
		"heuristic":      true,
		"seed":           42,
	})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("build index: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	m := decodeMap(t, rec)
	id, _ := m["indexId"].(string)
	if id == "" {
		t.Fatalf("build index: missing indexId in %s", rec.Body.String())
	}
	return id
}

func waitReady(t *testing.T, mgr *indexmanager.Manager) indexmanager.State {
	t.Helper()
	deadline := time.Now().Add(testTimeout)
	for time.Now().Before(deadline) {
		st := mgr.Status()
		switch st.Status {
		case indexmanager.StatusReady:
			return st
		case indexmanager.StatusError:
			t.Fatalf("build failed: %s", st.Error)
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("index did not become ready within %s", testTimeout)
	return indexmanager.State{}
}

func readyIndex(t *testing.T, h http.Handler, mgr *indexmanager.Manager) string {
	t.Helper()
	datasetID := createGeneratedDataset(t, h, testDatasetN, testDatasetD)
	startBuild(t, h, datasetID)
	waitReady(t, mgr)
	return datasetID
}

func TestHealthAndUnifiedRoutingErrors(t *testing.T) {
	srv, _ := newTestAPI(t)
	h := srv.Handler()

	rec := doJSON(t, h, http.MethodGet, basePath+"/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200", rec.Code)
	}
	if got := decodeMap(t, rec)["status"]; got != "ok" {
		t.Fatalf("health status field = %v, want ok", got)
	}

	rec = doJSON(t, h, http.MethodGet, basePath+"/does-not-exist", nil)
	expectError(t, rec, http.StatusNotFound, codeNotFound)

	rec = doJSON(t, h, http.MethodDelete, basePath+"/datasets", nil)
	expectError(t, rec, http.StatusMethodNotAllowed, codeMethodNotAllowed)

	rec = doJSON(t, h, http.MethodPost, basePath+"/search", nil)
	expectError(t, rec, http.StatusBadRequest, codeInvalidRequest)
}

func TestCORSPreflight(t *testing.T) {
	srv, _ := newTestAPI(t)
	h := srv.Handler()

	req := httptest.NewRequest(http.MethodOptions, basePath+"/datasets", nil)
	req.Header.Set("Origin", allowedOrigin)
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != allowedOrigin {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, allowedOrigin)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "POST") {
		t.Fatalf("Access-Control-Allow-Methods = %q, want POST", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Content-Type") {
		t.Fatalf("Access-Control-Allow-Headers = %q, want Content-Type", got)
	}

	req = httptest.NewRequest(http.MethodGet, basePath+"/health", nil)
	req.Header.Set("Origin", allowedOrigin)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != allowedOrigin {
		t.Fatalf("simple request CORS header = %q, want %q", got, allowedOrigin)
	}

	req = httptest.NewRequest(http.MethodOptions, basePath+"/health", nil)
	req.Header.Set("Origin", "http://evil.example")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("disallowed origin header = %q, want empty", got)
	}
}

func TestDatasetsGenerateListGet(t *testing.T) {
	srv, _ := newTestAPI(t)
	h := srv.Handler()

	rec := doJSON(t, h, http.MethodPost, basePath+"/datasets", map[string]any{
		"action":       "generate",
		"n":            24,
		"dim":          4,
		"distribution": "gaussian",
		"seed":         7,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("generate status = %d, body = %s", rec.Code, rec.Body.String())
	}
	created := decodeMap(t, rec)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("generate response missing id")
	}
	if got := int(created["n"].(float64)); got != 24 {
		t.Fatalf("n = %d, want 24", got)
	}
	if got := int(created["dim"].(float64)); got != 4 {
		t.Fatalf("dim = %d, want 4", got)
	}
	preview, ok := created["preview"].([]any)
	if !ok || len(preview) != 10 {
		t.Fatalf("preview len = %d, want 10", len(preview))
	}

	rec = doJSON(t, h, http.MethodGet, basePath+"/datasets", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	list := decodeMap(t, rec)
	datasets, _ := list["datasets"].([]any)
	if len(datasets) != 1 {
		t.Fatalf("list len = %d, want 1", len(datasets))
	}
	item := datasets[0].(map[string]any)
	for _, key := range []string{"id", "n", "dim", "createdAt"} {
		if _, ok := item[key]; !ok {
			t.Fatalf("list item missing %q: %v", key, item)
		}
	}

	rec = doJSON(t, h, http.MethodGet, basePath+"/datasets/"+id, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d", rec.Code)
	}
	meta := decodeMap(t, rec)
	stats, ok := meta["stats"].(map[string]any)
	if !ok {
		t.Fatalf("get response missing stats: %v", meta)
	}
	for _, key := range []string{"min", "max", "mean"} {
		arr, _ := stats[key].([]any)
		if len(arr) != 4 {
			t.Fatalf("stats.%s len = %d, want 4", key, len(arr))
		}
	}
	if got := int(meta["n"].(float64)); got != 24 {
		t.Fatalf("get n = %d, want 24", got)
	}

	rec = doJSON(t, h, http.MethodGet, basePath+"/datasets/ds_999999", nil)
	expectError(t, rec, http.StatusNotFound, codeDatasetNotFound)

	rec = doJSON(t, h, http.MethodPost, basePath+"/datasets", map[string]any{
		"action": "generate", "n": 10, "dim": 3, "distribution": "poisson",
	})
	expectError(t, rec, http.StatusBadRequest, codeInvalidRequest)

	rec = doJSON(t, h, http.MethodPost, basePath+"/datasets", map[string]any{
		"action": "generate", "n": 0, "dim": 3, "distribution": "uniform",
	})
	expectError(t, rec, http.StatusBadRequest, codeInvalidRequest)
}

func multipartUpload(t *testing.T, h http.Handler, filename, content string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write([]byte(content)); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, basePath+"/datasets", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestDatasetUploadCSVAndJSON(t *testing.T) {
	srv, _ := newTestAPI(t)
	h := srv.Handler()

	csv := "a,b,c\n1,2,3\n4,5,6\n7,8,9\n"
	rec := multipartUpload(t, h, "vectors.csv", csv)
	if rec.Code != http.StatusCreated {
		t.Fatalf("csv upload status = %d, body = %s", rec.Code, rec.Body.String())
	}
	m := decodeMap(t, rec)
	if got := int(m["n"].(float64)); got != 3 {
		t.Fatalf("csv n = %d, want 3", got)
	}
	if got := int(m["dim"].(float64)); got != 3 {
		t.Fatalf("csv dim = %d, want 3", got)
	}

	rec = multipartUpload(t, h, "vectors.json", `{"vectors":[[1,2],[3,4]]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("json upload status = %d, body = %s", rec.Code, rec.Body.String())
	}
	m = decodeMap(t, rec)
	if got := int(m["n"].(float64)); got != 2 {
		t.Fatalf("json n = %d, want 2", got)
	}

	rec = multipartUpload(t, h, "vectors.txt", "1,2,3")
	expectError(t, rec, http.StatusUnsupportedMediaType, codeUnsupportedFileType)

	rec = multipartUpload(t, h, "broken.csv", "1,2,3\n4,5\n")
	expectError(t, rec, http.StatusBadRequest, codeInvalidRequest)

	req := httptest.NewRequest(http.MethodPost, basePath+"/datasets", strings.NewReader("x"))
	req.Header.Set("Content-Type", "text/plain")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	expectError(t, rec, http.StatusBadRequest, codeInvalidRequest)
}

func TestIndexBuildStatusStats(t *testing.T) {
	srv, mgr := newTestAPI(t)
	h := srv.Handler()

	rec := doJSON(t, h, http.MethodGet, basePath+"/index/status", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("idle status code = %d", rec.Code)
	}
	idle := decodeMap(t, rec)
	if idle["status"] != "idle" {
		t.Fatalf("idle status = %v", idle["status"])
	}
	if idle["error"] != nil {
		t.Fatalf("idle error = %v, want null", idle["error"])
	}

	rec = doJSON(t, h, http.MethodGet, basePath+"/index/stats", nil)
	expectError(t, rec, http.StatusNotFound, codeIndexNotFound)

	datasetID := createGeneratedDataset(t, h, testDatasetN, testDatasetD)
	indexID := startBuild(t, h, datasetID)

	st := waitReady(t, mgr)
	if st.IndexID != indexID {
		t.Fatalf("ready indexId = %q, want %q", st.IndexID, indexID)
	}

	rec = doJSON(t, h, http.MethodGet, basePath+"/index/status", nil)
	ready := decodeMap(t, rec)
	if ready["status"] != "ready" {
		t.Fatalf("status = %v, want ready", ready["status"])
	}
	if got := ready["progress"].(float64); got != 1 {
		t.Fatalf("progress = %v, want 1", got)
	}
	if ready["indexId"] != indexID {
		t.Fatalf("status indexId = %v, want %s", ready["indexId"], indexID)
	}
	if ready["error"] != nil {
		t.Fatalf("ready error = %v, want null", ready["error"])
	}

	rec = doJSON(t, h, http.MethodGet, basePath+"/index/stats", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("stats status = %d, body = %s", rec.Code, rec.Body.String())
	}
	stats := decodeMap(t, rec)
	if got := int(stats["nodes"].(float64)); got != testDatasetN {
		t.Fatalf("nodes = %d, want %d", got, testDatasetN)
	}
	maxLevel := int(stats["maxLevel"].(float64))
	if maxLevel < 0 {
		t.Fatalf("maxLevel = %d, want >= 0", maxLevel)
	}
	npl, _ := stats["nodesPerLevel"].([]any)
	if len(npl) != maxLevel+1 {
		t.Fatalf("nodesPerLevel len = %d, want %d", len(npl), maxLevel+1)
	}
	if got := int(npl[0].(float64)); got != testDatasetN {
		t.Fatalf("nodesPerLevel[0] = %d, want %d", got, testDatasetN)
	}
	entry := int(stats["entryPoint"].(float64))
	if entry < 0 || entry >= testDatasetN {
		t.Fatalf("entryPoint = %d, out of range", entry)
	}
	dd := stats["degreeDistribution"].(map[string]any)
	if got := int(dd["max"].(float64)); got > 2*buildM {
		t.Fatalf("degree max = %d, want <= %d", got, 2*buildM)
	}
	if got := dd["mean"].(float64); got <= 0 {
		t.Fatalf("degree mean = %v, want > 0", got)
	}
	if stats["metric"] != "l2" {
		t.Fatalf("metric = %v, want l2", stats["metric"])
	}
	if got := int(stats["M"].(float64)); got != buildM {
		t.Fatalf("M = %d, want %d", got, buildM)
	}
	if got := int(stats["efConstruction"].(float64)); got != buildEfC {
		t.Fatalf("efConstruction = %d, want %d", got, buildEfC)
	}

	rec = doJSON(t, h, http.MethodPost, basePath+"/index", map[string]any{
		"datasetId": "ds_999999",
	})
	expectError(t, rec, http.StatusNotFound, codeDatasetNotFound)

	rec = doJSON(t, h, http.MethodPost, basePath+"/index", map[string]any{
		"datasetId": datasetID,
		"metric":    "manhattan",
	})
	expectError(t, rec, http.StatusBadRequest, codeInvalidRequest)

	// Rebuild replaces the previous index.
	newIndexID := startBuild(t, h, datasetID)
	if newIndexID == indexID {
		t.Fatal("rebuild reused the previous index id")
	}
	waitReady(t, mgr)
}

func TestVectorInsertAndSearchByID(t *testing.T) {
	srv, mgr := newTestAPI(t)
	h := srv.Handler()
	readyIndex(t, h, mgr)

	vec := []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}
	rec := doJSON(t, h, http.MethodPost, basePath+"/vectors", map[string]any{
		"vectorId": "custom",
		"vector":   vec,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("insert status = %d, body = %s", rec.Code, rec.Body.String())
	}
	inserted := decodeMap(t, rec)
	if got := int(inserted["id"].(float64)); got != testDatasetN {
		t.Fatalf("insert id = %d, want %d", got, testDatasetN)
	}
	if inserted["vectorId"] != "custom" {
		t.Fatalf("insert vectorId = %v, want custom", inserted["vectorId"])
	}

	vec2 := make([]float32, len(vec))
	for i := range vec {
		vec2[i] = vec[i] + 0.01
	}
	rec = doJSON(t, h, http.MethodPost, basePath+"/vectors", map[string]any{"vector": vec2})
	if rec.Code != http.StatusCreated {
		t.Fatalf("auto-id insert status = %d, body = %s", rec.Code, rec.Body.String())
	}
	auto := decodeMap(t, rec)
	if auto["vectorId"] != "ext_0" {
		t.Fatalf("auto vectorId = %v, want ext_0", auto["vectorId"])
	}

	rec = doJSON(t, h, http.MethodPost, basePath+"/vectors", map[string]any{
		"vectorId": "custom", "vector": vec,
	})
	expectError(t, rec, http.StatusBadRequest, codeInvalidRequest)

	rec = doJSON(t, h, http.MethodPost, basePath+"/vectors", map[string]any{
		"vector": []float32{1, 2},
	})
	expectError(t, rec, http.StatusBadRequest, codeInvalidDimension)

	rec = doJSON(t, h, http.MethodPost, basePath+"/search", map[string]any{
		"vectorId": "custom", "k": 3, "efSearch": 10, "mode": "hnsw",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("search by id status = %d, body = %s", rec.Code, rec.Body.String())
	}
	res := decodeMap(t, rec)
	results := res["results"].([]any)
	if len(results) != 3 {
		t.Fatalf("found %d results, want 3", len(results))
	}
	top := results[0].(map[string]any)
	if top["vectorId"] != "custom" {
		t.Fatalf("top vectorId = %v, want custom", top["vectorId"])
	}
	if got := top["distance"].(float64); got != 0 {
		t.Fatalf("top distance = %v, want 0", got)
	}

	// Numeric vectorId must resolve the built dataset's external id "0".
	rec = doJSON(t, h, http.MethodPost, basePath+"/search", map[string]any{
		"vectorId": 0, "k": 3, "mode": "hnsw",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("numeric vectorId search status = %d, body = %s", rec.Code, rec.Body.String())
	}
	res = decodeMap(t, rec)
	top = res["results"].([]any)[0].(map[string]any)
	if got := int(top["id"].(float64)); got != 0 {
		t.Fatalf("numeric vectorId top id = %d, want 0", got)
	}

	// Brute force must see the inserted vector.
	rec = doJSON(t, h, http.MethodPost, basePath+"/search", map[string]any{
		"vectorId": "custom", "k": 1, "mode": "bruteforce",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("bruteforce search status = %d, body = %s", rec.Code, rec.Body.String())
	}
	res = decodeMap(t, rec)
	top = res["results"].([]any)[0].(map[string]any)
	if top["vectorId"] != "custom" {
		t.Fatalf("bruteforce top vectorId = %v, want custom", top["vectorId"])
	}

	rec = doJSON(t, h, http.MethodPost, basePath+"/search", map[string]any{
		"vectorId": "nope", "k": 1, "mode": "hnsw",
	})
	expectError(t, rec, http.StatusBadRequest, codeInvalidRequest)
}

func TestSearchModesTraceAndValidation(t *testing.T) {
	srv, mgr := newTestAPI(t)
	h := srv.Handler()
	readyIndex(t, h, mgr)

	// Use dataset vector 0 as the query; HNSW and brute force must agree on the
	// exact top-1 hit (the query itself).
	meta := decodeMap(t, doJSON(t, h, http.MethodGet, basePath+"/datasets", nil))
	firstID := meta["datasets"].([]any)[0].(map[string]any)["id"].(string)
	ds := decodeMap(t, doJSON(t, h, http.MethodGet, basePath+"/datasets/"+firstID, nil))
	preview := ds["preview"].([]any)
	query := make([]float32, testDatasetD)
	for i, x := range preview[0].([]any) {
		query[i] = float32(x.(float64))
	}

	rec := doJSON(t, h, http.MethodPost, basePath+"/search", map[string]any{
		"vector": query, "k": testSearchK, "efSearch": 20, "mode": "hnsw",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("hnsw search status = %d, body = %s", rec.Code, rec.Body.String())
	}
	hnswRes := decodeMap(t, rec)
	hnswResults := hnswRes["results"].([]any)
	if len(hnswResults) != testSearchK {
		t.Fatalf("hnsw found = %d, want %d", len(hnswResults), testSearchK)
	}
	if got := int(hnswRes["found"].(float64)); got != testSearchK {
		t.Fatalf("hnsw found field = %d, want %d", got, testSearchK)
	}
	prev := -1.0
	for _, item := range hnswResults {
		r := item.(map[string]any)
		d := r["distance"].(float64)
		if d < prev {
			t.Fatalf("results not sorted by distance: %v after %v", d, prev)
		}
		prev = d
		if len(r["vector"].([]any)) != testDatasetD {
			t.Fatalf("result vector len = %d, want %d", len(r["vector"].([]any)), testDatasetD)
		}
	}
	hnswTopID := int(hnswResults[0].(map[string]any)["id"].(float64))
	if hnswTopID != 0 {
		t.Fatalf("hnsw top id = %d, want 0", hnswTopID)
	}
	if _, ok := hnswRes["trace"]; ok {
		t.Fatal("trace present without trace=true")
	}

	rec = doJSON(t, h, http.MethodPost, basePath+"/search", map[string]any{
		"vector": query, "k": testSearchK, "mode": "bruteforce",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("bruteforce search status = %d, body = %s", rec.Code, rec.Body.String())
	}
	bfRes := decodeMap(t, rec)
	bfResults := bfRes["results"].([]any)
	if len(bfResults) != testSearchK {
		t.Fatalf("bruteforce found = %d, want %d", len(bfResults), testSearchK)
	}
	bfTopID := int(bfResults[0].(map[string]any)["id"].(float64))
	if bfTopID != hnswTopID {
		t.Fatalf("top ids differ: hnsw=%d bruteforce=%d", hnswTopID, bfTopID)
	}

	rec = doJSON(t, h, http.MethodPost, basePath+"/search", map[string]any{
		"vector": query, "k": testSearchK, "efSearch": 20, "mode": "hnsw", "trace": true,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("trace search status = %d, body = %s", rec.Code, rec.Body.String())
	}
	traced := decodeMap(t, rec)
	trace, ok := traced["trace"].(map[string]any)
	if !ok {
		t.Fatalf("trace missing in %s", rec.Body.String())
	}
	layers := trace["layers"].([]any)
	if len(layers) == 0 {
		t.Fatal("trace has no layers")
	}
	layer0Found := false
	for _, item := range layers {
		layer := item.(map[string]any)
		if int(layer["level"].(float64)) == 0 {
			layer0Found = true
			visited := layer["visited"].([]any)
			if len(visited) == 0 {
				t.Fatal("layer 0 trace has no visited nodes")
			}
			closest := int(layer["closest"].(float64))
			if closest < 0 || closest >= testDatasetN {
				t.Fatalf("trace closest = %d out of range", closest)
			}
		}
	}
	if !layer0Found {
		t.Fatal("trace missing layer 0")
	}

	rec = doJSON(t, h, http.MethodPost, basePath+"/search", map[string]any{
		"vector": query, "k": 1, "mode": "bruteforce", "trace": true,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("bruteforce trace status = %d", rec.Code)
	}
	if _, ok := decodeMap(t, rec)["trace"]; ok {
		t.Fatal("bruteforce returned a trace")
	}

	rec = doJSON(t, h, http.MethodPost, basePath+"/search", map[string]any{
		"vector": query, "k": testDatasetN + 1, "mode": "hnsw",
	})
	expectError(t, rec, http.StatusBadRequest, codeKExceedsN)

	rec = doJSON(t, h, http.MethodPost, basePath+"/search", map[string]any{
		"vector": query, "k": 1, "mode": "quadtree",
	})
	expectError(t, rec, http.StatusBadRequest, codeInvalidRequest)

	rec = doJSON(t, h, http.MethodPost, basePath+"/search", map[string]any{
		"vector": query, "vectorId": 0, "k": 1, "mode": "hnsw",
	})
	expectError(t, rec, http.StatusBadRequest, codeInvalidRequest)

	rec = doJSON(t, h, http.MethodPost, basePath+"/search", map[string]any{
		"k": 1, "mode": "hnsw",
	})
	expectError(t, rec, http.StatusBadRequest, codeInvalidRequest)

	rec = doJSON(t, h, http.MethodPost, basePath+"/search", map[string]any{
		"vector": []float32{1, 2}, "k": 1, "mode": "hnsw",
	})
	expectError(t, rec, http.StatusBadRequest, codeInvalidDimension)

	rec = doJSON(t, h, http.MethodPost, basePath+"/search", map[string]any{
		"vector": query, "k": 0, "mode": "hnsw",
	})
	expectError(t, rec, http.StatusBadRequest, codeInvalidRequest)
}

func TestSearchNotReady(t *testing.T) {
	srv, _ := newTestAPI(t)
	h := srv.Handler()

	rec := doJSON(t, h, http.MethodPost, basePath+"/search", map[string]any{
		"vector": []float32{1, 2}, "k": 1, "mode": "hnsw",
	})
	expectError(t, rec, http.StatusNotFound, codeIndexNotFound)
}

func TestIndexEventsSSE(t *testing.T) {
	srv, mgr := newTestAPI(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	datasetID := createGeneratedDatasetHTTP(t, ts.URL, 400, 8)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+basePath+"/index/events", nil)
	if err != nil {
		t.Fatalf("new SSE request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("open SSE: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("SSE content type = %q", got)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("SSE cache control = %q", got)
	}

	reader := bufio.NewReader(resp.Body)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read initial SSE line: %v", err)
	}
	if strings.TrimSpace(line) != ": idle" {
		t.Fatalf("initial SSE line = %q, want comment : idle", line)
	}

	rec := doJSONHTTP(t, ts.URL, http.MethodPost, basePath+"/index", map[string]any{
		"datasetId":      datasetID,
		"M":              buildM,
		"efConstruction": buildEfC,
		"metric":         "l2",
		"seed":           42,
	})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("build via HTTP status = %d, body = %s", rec.Code, rec.Body.String())
	}

	sawProgress := false
	sawDone := false
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read SSE event (progress=%t done=%t): %v", sawProgress, sawDone, err)
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "event: progress" {
			sawProgress = true
		}
		if trimmed == "event: done" {
			sawDone = true
			dataLine, err := reader.ReadString('\n')
			if err != nil {
				t.Fatalf("read done payload: %v", err)
			}
			if !strings.HasPrefix(dataLine, "data: ") {
				t.Fatalf("done data line = %q", dataLine)
			}
			var payload struct {
				IndexID string `json:"indexId"`
			}
			if err := json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(dataLine), "data: ")), &payload); err != nil {
				t.Fatalf("decode done payload: %v", err)
			}
			if payload.IndexID == "" {
				t.Fatal("done payload missing indexId")
			}
			break
		}
	}
	if !sawProgress {
		t.Fatal("no progress event before done")
	}
	waitReady(t, mgr)

	// A second connection after the build must receive done immediately.
	resp2, err := http.Get(ts.URL + basePath + "/index/events")
	if err != nil {
		t.Fatalf("open ready SSE: %v", err)
	}
	defer resp2.Body.Close()
	body, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatalf("read ready SSE: %v", err)
	}
	if !strings.Contains(string(body), "event: done") {
		t.Fatalf("ready SSE body = %q, want done event", body)
	}
}

func TestSSEClientDisconnect(t *testing.T) {
	srv, _ := newTestAPI(t)
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, basePath+"/index/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		srv.Handler().ServeHTTP(rec, req)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SSE handler did not stop after client disconnect")
	}
}

func TestSSEManagerCloseStopsStream(t *testing.T) {
	srv, mgr := newTestAPI(t)
	req := httptest.NewRequest(http.MethodGet, basePath+"/index/events", nil)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		srv.Handler().ServeHTTP(rec, req)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	mgr.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SSE handler did not stop after manager shutdown")
	}
}

func createGeneratedDatasetHTTP(t *testing.T, baseURL string, n, dim int) string {
	t.Helper()
	rec := doJSONHTTP(t, baseURL, http.MethodPost, basePath+"/datasets", map[string]any{
		"action":       "generate",
		"n":            n,
		"dim":          dim,
		"distribution": "uniform",
		"seed":         1,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create dataset via HTTP: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	m := decodeMap(t, rec)
	id, _ := m["id"].(string)
	if id == "" {
		t.Fatalf("create dataset via HTTP: missing id in %s", rec.Body.String())
	}
	return id
}

func doJSONHTTP(t *testing.T, baseURL, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req, err := http.NewRequest(method, baseURL+path, bytes.NewReader(data))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	rec := httptest.NewRecorder()
	rec.Code = resp.StatusCode
	for k, vs := range resp.Header {
		for _, v := range vs {
			rec.Header().Add(k, v)
		}
	}
	if _, err := io.Copy(rec.Body, resp.Body); err != nil {
		t.Fatalf("read response: %v", err)
	}
	return rec
}
