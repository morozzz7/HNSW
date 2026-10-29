// Package indexmanager owns the lifecycle of the HNSW index exposed through
// the HTTP API. It keeps the current index, its dataset vectors, the
// external-id <-> internal-id mapping, a brute-force counterpart for exact
// search, and the asynchronous build state together with SSE subscribers.
//
// The manager serialises mutating operations (build, insert) against searches
// with a single RWMutex. The lock order is manager -> hnsw.Index; the build
// goroutine touches the fresh hnsw.Index before it is published, so no other
// goroutine can observe a partially built index.
package indexmanager

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"

	"hnsw-coursework/internal/bruteforce"
	"hnsw-coursework/internal/dataset"
	"hnsw-coursework/internal/hnsw"
)

// Status is the lifecycle state of the current index.
type Status string

const (
	StatusIdle     Status = "idle"
	StatusBuilding Status = "building"
	StatusReady    Status = "ready"
	StatusError    Status = "error"
)

// Event types published over SSE.
const (
	EventProgress = "progress"
	EventDone     = "done"
	EventError    = "error"
)

// Sentinel errors returned by the manager. HTTP handlers map them to the
// documented API error codes.
var (
	// ErrNoIndex means no index has been built yet (status idle).
	ErrNoIndex = errors.New("index: no index built")
	// ErrNotReady means the index is building or in an error state.
	ErrNotReady = errors.New("index: not ready")
	// ErrBuildInProgress is returned when a second build is requested while
	// the first one is still running. The API rejects it with 409.
	ErrBuildInProgress = errors.New("index: build already in progress")
	// ErrDatasetNotFound means the requested dataset id is unknown.
	ErrDatasetNotFound = errors.New("index: dataset not found")
	// ErrDuplicateID means the external vector id is already present.
	ErrDuplicateID = errors.New("index: duplicate vector id")
	// ErrDimension means a vector does not match the index dimension.
	ErrDimension = errors.New("index: vector dimension mismatch")
	// ErrInvalid wraps invalid build parameters.
	ErrInvalid = errors.New("index: invalid build config")
)

// BuildConfig describes an asynchronous index build request.
type BuildConfig struct {
	DatasetID      string
	M              int
	Mmax0          int
	EfConstruction int
	Metric         hnsw.Metric
	UseHeuristic   bool
	Seed           int64
}

// State is a snapshot of the manager state for GET /index/status.
type State struct {
	Status    Status
	Progress  float64
	Processed int
	Total     int
	ElapsedMs int64
	IndexID   string
	Error     string
}

// Event is one SSE message published to subscribers. Data is a struct with
// JSON tags (see ProgressData, DoneData, ErrorData).
type Event struct {
	Type string
	Data any
}

// ProgressData is the payload of a "progress" SSE event.
type ProgressData struct {
	Progress  float64 `json:"progress"`
	Processed int     `json:"processed"`
	Total     int     `json:"total"`
}

// DoneData is the payload of a "done" SSE event.
type DoneData struct {
	IndexID string `json:"indexId"`
}

// ErrorData is the payload of an "error" SSE event.
type ErrorData struct {
	Message string `json:"message"`
}

// Subscription is a live SSE subscriber. C delivers events; Done is closed by
// Manager.Close so handlers exit promptly on shutdown. The handler also stops
// on request cancellation or after a terminal (done/error) event.
type Subscription struct {
	C    <-chan Event
	Done <-chan struct{}
	ch   chan Event
	done chan struct{}
}

// SearchResult is one search hit with both id spaces and the stored vector.
type SearchResult struct {
	Internal uint32
	VectorID string
	Distance float32
	Vector   []float32
}

// DegreeDistribution summarizes the graph degree histogram.
type DegreeDistribution struct {
	Min  int
	Max  int
	Mean float64
	P50  int
}

// Stats is a snapshot of the ready index for GET /index/stats.
type Stats struct {
	Nodes              int
	MaxLevel           int
	EntryPoint         uint32
	NodesPerLevel      []int
	DegreeDistribution DegreeDistribution
	Metric             hnsw.Metric
	M                  int
	EfConstruction     int
	IndexID            string
	DatasetID          string
}

// Manager holds the current index and its build state.
type Manager struct {
	mu    sync.RWMutex
	store *dataset.Store

	status      Status
	indexID     string
	datasetID   string
	idx         *hnsw.Index
	bf          *bruteforce.Index
	vectors     [][]float32 // internal id -> vector
	extIDs      []string    // internal id -> external id
	extToInt    map[string]uint32
	nextExt     int
	metric      hnsw.Metric
	progress    float64
	processed   int
	total       int
	startedAt   time.Time
	elapsedMs   int64
	lastErr     string
	buildCancel context.CancelFunc // non-nil while building
	subs        map[*Subscription]struct{}
	closed      bool
}

// New creates an idle manager attached to the dataset store.
func New(store *dataset.Store) *Manager {
	return &Manager{
		store:    store,
		status:   StatusIdle,
		extToInt: make(map[string]uint32),
		subs:     make(map[*Subscription]struct{}),
	}
}

// StartBuild validates the request, resets the previous index and starts an
// asynchronous build. A second concurrent build is rejected with
// ErrBuildInProgress; rebuilding a ready index is allowed and replaces it.
func (m *Manager) StartBuild(cfg BuildConfig) (string, error) {
	if cfg.DatasetID == "" {
		return "", fmt.Errorf("%w: datasetId is required", ErrInvalid)
	}
	if cfg.M <= 0 {
		return "", fmt.Errorf("%w: M must be positive", ErrInvalid)
	}
	if cfg.Mmax0 < 0 {
		return "", fmt.Errorf("%w: Mmax0 must not be negative", ErrInvalid)
	}
	if cfg.EfConstruction <= 0 {
		return "", fmt.Errorf("%w: efConstruction must be positive", ErrInvalid)
	}
	if !validMetric(cfg.Metric) {
		return "", fmt.Errorf("%w: unknown metric %q", ErrInvalid, cfg.Metric)
	}

	m.mu.Lock()

	if m.status == StatusBuilding {
		m.mu.Unlock()
		return "", ErrBuildInProgress
	}
	if m.closed {
		m.mu.Unlock()
		return "", fmt.Errorf("%w: manager is closed", ErrInvalid)
	}
	ds := m.store.Get(cfg.DatasetID)
	if ds == nil {
		m.mu.Unlock()
		return "", ErrDatasetNotFound
	}
	if ds.N == 0 || len(ds.Vectors) == 0 {
		m.mu.Unlock()
		return "", fmt.Errorf("%w: dataset is empty", ErrInvalid)
	}

	hcfg := hnsw.Config{
		M:              cfg.M,
		Mmax0:          cfg.Mmax0,
		EfConstruction: cfg.EfConstruction,
		Metric:         cfg.Metric,
		Seed:           cfg.Seed,
		UseHeuristic:   cfg.UseHeuristic,
	}
	idx := hnsw.New(hcfg)

	// Own copies of the outer slice so later inserts never write into the
	// dataset's backing array. Inner vectors are immutable and shared.
	vectors := make([][]float32, len(ds.Vectors))
	copy(vectors, ds.Vectors)
	extIDs := make([]string, len(vectors))
	extToInt := make(map[string]uint32, len(vectors))
	for i := range vectors {
		id := strconv.Itoa(i)
		extIDs[i] = id
		extToInt[id] = uint32(i)
	}

	ctx, cancel := context.WithCancel(context.Background())
	indexID := newIndexID()

	m.status = StatusBuilding
	m.indexID = indexID
	m.datasetID = cfg.DatasetID
	m.idx = nil // previous index is reset: searches return INDEX_NOT_READY
	m.bf = nil
	m.vectors = vectors
	m.extIDs = extIDs
	m.extToInt = extToInt
	m.nextExt = 0
	m.metric = hcfg.Metric
	m.progress = 0
	m.processed = 0
	m.total = len(vectors)
	m.startedAt = time.Now()
	m.elapsedMs = 0
	m.lastErr = ""
	m.buildCancel = cancel
	m.publishLocked(Event{
		Type: EventProgress,
		Data: ProgressData{Progress: 0, Processed: 0, Total: len(vectors)},
	})
	m.mu.Unlock()

	go m.build(ctx, idx, vectors, indexID)
	return indexID, nil
}

// build inserts vectors one by one, updates progress and finally publishes a
// ready index. The index is not visible to searches until finishBuild.
func (m *Manager) build(ctx context.Context, idx *hnsw.Index, vectors [][]float32, indexID string) {
	total := len(vectors)
	step := total / 100
	if step < 1 {
		step = 1
	}
	lastPublished := 0

	for i, v := range vectors {
		select {
		case <-ctx.Done():
			m.failBuild(indexID, "build cancelled")
			return
		default:
		}
		if _, err := idx.Insert(v, uint32(i)); err != nil {
			m.failBuild(indexID, fmt.Sprintf("insert vector %d: %v", i, err))
			return
		}
		processed := i + 1
		publish := processed == total || processed-lastPublished >= step
		m.updateProgress(indexID, processed, total, publish)
		if publish {
			lastPublished = processed
		}
	}

	bf, err := bruteforce.NewWithWorkers(vectors, hnsw.Distance(idx.Config().Metric), runtime.GOMAXPROCS(0))
	if err != nil {
		m.failBuild(indexID, fmt.Sprintf("build brute-force index: %v", err))
		return
	}
	m.finishBuild(indexID, idx, bf)
}

func (m *Manager) updateProgress(indexID string, processed, total int, publish bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.indexID != indexID || m.status != StatusBuilding {
		return
	}
	m.processed = processed
	if total > 0 {
		m.progress = float64(processed) / float64(total)
	}
	if publish {
		m.publishLocked(Event{
			Type: EventProgress,
			Data: ProgressData{Progress: m.progress, Processed: processed, Total: total},
		})
	}
}

func (m *Manager) finishBuild(indexID string, idx *hnsw.Index, bf *bruteforce.Index) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.indexID != indexID || m.status != StatusBuilding {
		return
	}
	m.idx = idx
	m.bf = bf
	m.status = StatusReady
	m.progress = 1
	m.processed = m.total
	m.elapsedMs = time.Since(m.startedAt).Milliseconds()
	m.lastErr = ""
	m.buildCancel = nil
	m.publishLocked(Event{Type: EventDone, Data: DoneData{IndexID: indexID}})
}

func (m *Manager) failBuild(indexID, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.indexID != indexID || m.status != StatusBuilding {
		return
	}
	m.idx = nil
	m.bf = nil
	m.status = StatusError
	m.lastErr = message
	m.elapsedMs = time.Since(m.startedAt).Milliseconds()
	m.buildCancel = nil
	m.publishLocked(Event{Type: EventError, Data: ErrorData{Message: message}})
}

// Close cancels an in-flight build. It is called on server shutdown.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return
	}
	m.closed = true
	if m.buildCancel != nil {
		m.buildCancel()
		m.buildCancel = nil
	}
	for sub := range m.subs {
		close(sub.done)
	}
}

// Status returns the current build state snapshot.
func (m *Manager) Status() State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stateLocked()
}

func (m *Manager) stateLocked() State {
	elapsed := m.elapsedMs
	if m.status == StatusBuilding && !m.startedAt.IsZero() {
		elapsed = time.Since(m.startedAt).Milliseconds()
	}
	return State{
		Status:    m.status,
		Progress:  m.progress,
		Processed: m.processed,
		Total:     m.total,
		ElapsedMs: elapsed,
		IndexID:   m.indexID,
		Error:     m.lastErr,
	}
}

// Len returns the number of vectors in the current index (0 while idle).
func (m *Manager) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.vectors)
}

// Dim returns the vector dimension of the current index.
func (m *Manager) Dim() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.idx != nil {
		return m.idx.Dim()
	}
	if len(m.vectors) > 0 {
		return len(m.vectors[0])
	}
	return 0
}

// Metric returns the distance metric of the current index.
func (m *Manager) Metric() hnsw.Metric {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metric
}

// VectorByID resolves an external id to its stored vector.
func (m *Manager) VectorByID(id string) ([]float32, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.status != StatusReady {
		return nil, false
	}
	internal, ok := m.extToInt[id]
	if !ok || int(internal) >= len(m.vectors) {
		return nil, false
	}
	return m.vectors[internal], true
}

// HNSWSearch runs an approximate search and maps internal ids to external ids.
// The caller must have verified that the index is ready; ErrNoIndex/ErrNotReady
// are still returned for races with a rebuild.
func (m *Manager) HNSWSearch(q []float32, k, ef int, trace bool) ([]SearchResult, *hnsw.Trace, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.status != StatusReady || m.idx == nil {
		return nil, nil, m.notReadyErrLocked()
	}
	if len(q) != m.idx.Dim() {
		return nil, nil, ErrDimension
	}

	var (
		raw []hnsw.Result
		tr  *hnsw.Trace
		err error
	)
	if trace {
		raw, tr, err = m.idx.SearchWithTrace(q, k, ef)
	} else {
		raw, err = m.idx.Search(q, k, ef)
	}
	if err != nil {
		return nil, nil, err
	}
	return m.mapResultsLocked(raw), tr, nil
}

// BruteForceSearch runs an exact search over the same vector set.
func (m *Manager) BruteForceSearch(q []float32, k int) ([]SearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.status != StatusReady || m.bf == nil {
		return nil, m.notReadyErrLocked()
	}
	if len(q) != m.bf.Dim() {
		return nil, ErrDimension
	}
	raw, err := m.bf.Search(q, k)
	if err != nil {
		return nil, err
	}
	out := make([]SearchResult, len(raw))
	for i, r := range raw {
		out[i] = SearchResult{
			Internal: r.ID,
			VectorID: m.externalIDLocked(r.ID),
			Distance: r.Dist,
			Vector:   m.vectorLocked(r.ID),
		}
	}
	return out, nil
}

// InsertVector inserts one vector into a ready index. When vectorID is empty,
// a unique "ext_<n>" id is assigned. The brute-force counterpart and the id
// maps are updated atomically with the HNSW insert.
func (m *Manager) InsertVector(vector []float32, vectorID string) (uint32, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status != StatusReady || m.idx == nil {
		return 0, "", m.notReadyErrLocked()
	}
	if len(vector) == 0 || len(vector) != m.idx.Dim() {
		return 0, "", ErrDimension
	}
	for _, x := range vector {
		if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
			return 0, "", fmt.Errorf("%w: vector contains NaN or Inf", ErrInvalid)
		}
	}
	if vectorID == "" {
		vectorID = m.nextAssignedIDLocked()
	}
	if _, exists := m.extToInt[vectorID]; exists {
		return 0, "", fmt.Errorf("%w: %q", ErrDuplicateID, vectorID)
	}

	// Build the brute-force snapshot before mutating the HNSW index so that a
	// failure leaves the manager in a consistent state.
	snapshot := make([][]float32, len(m.vectors)+1)
	copy(snapshot, m.vectors)
	snapshot[len(m.vectors)] = vector
	bf, err := bruteforce.NewWithWorkers(snapshot, hnsw.Distance(m.metric), runtime.GOMAXPROCS(0))
	if err != nil {
		return 0, "", fmt.Errorf("rebuild brute-force index: %w", err)
	}

	internal, err := m.idx.Insert(vector, uint32(len(m.vectors)))
	if err != nil {
		return 0, "", err
	}

	m.vectors = snapshot
	m.bf = bf
	m.extIDs = append(m.extIDs, vectorID)
	m.extToInt[vectorID] = internal
	return internal, vectorID, nil
}

// Stats returns a structural snapshot of the ready index.
func (m *Manager) Stats() (Stats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.status != StatusReady || m.idx == nil {
		return Stats{}, m.notReadyErrLocked()
	}
	st := m.idx.Stats()
	return Stats{
		Nodes:              st.Nodes,
		MaxLevel:           st.MaxLevel,
		EntryPoint:         st.EntryPoint,
		NodesPerLevel:      st.NodesPerLevel,
		DegreeDistribution: degreeDistribution(st.DegreeHistogram),
		Metric:             st.Metric,
		M:                  st.M,
		EfConstruction:     st.EfConstruction,
		IndexID:            m.indexID,
		DatasetID:          m.datasetID,
	}, nil
}

// Subscribe registers an SSE subscriber and returns the current state snapshot
// atomically (no event can be missed between subscribing and reading state).
func (m *Manager) Subscribe() (*Subscription, State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch := make(chan Event, 256)
	done := make(chan struct{})
	sub := &Subscription{C: ch, Done: done, ch: ch, done: done}
	m.subs[sub] = struct{}{}
	return sub, m.stateLocked()
}

// Unsubscribe removes a subscriber.
func (m *Manager) Unsubscribe(sub *Subscription) {
	if sub == nil {
		return
	}
	m.mu.Lock()
	delete(m.subs, sub)
	m.mu.Unlock()
}

// publishLocked sends ev to every subscriber without blocking the build. A
// full buffer drops progress events; terminal events make room by discarding
// the oldest buffered event.
func (m *Manager) publishLocked(ev Event) {
	for sub := range m.subs {
		select {
		case sub.ch <- ev:
		default:
			if ev.Type == EventProgress {
				continue
			}
			// Make room for a terminal event.
			select {
			case <-sub.ch:
			default:
			}
			select {
			case sub.ch <- ev:
			default:
			}
		}
	}
}

func (m *Manager) notReadyErrLocked() error {
	if m.status == StatusIdle {
		return ErrNoIndex
	}
	return ErrNotReady
}

func (m *Manager) mapResultsLocked(raw []hnsw.Result) []SearchResult {
	out := make([]SearchResult, len(raw))
	for i, r := range raw {
		out[i] = SearchResult{
			Internal: r.ID,
			VectorID: m.externalIDLocked(r.ID),
			Distance: r.Dist,
			Vector:   m.vectorLocked(r.ID),
		}
	}
	return out
}

func (m *Manager) externalIDLocked(internal uint32) string {
	if int(internal) < len(m.extIDs) {
		return m.extIDs[internal]
	}
	return ""
}

func (m *Manager) vectorLocked(internal uint32) []float32 {
	if int(internal) < len(m.vectors) {
		return m.vectors[internal]
	}
	return nil
}

func (m *Manager) nextAssignedIDLocked() string {
	for {
		id := fmt.Sprintf("ext_%d", m.nextExt)
		m.nextExt++
		if _, exists := m.extToInt[id]; !exists {
			return id
		}
	}
}

func validMetric(m hnsw.Metric) bool {
	switch m {
	case hnsw.MetricL2, hnsw.MetricCosine, hnsw.MetricInnerProduct:
		return true
	default:
		return false
	}
}

// degreeDistribution summarizes a degree histogram: min/max/mean degree and
// the nearest-rank median (p50). The histogram includes every (node, level)
// pair present in the graph.
func degreeDistribution(hist map[int]int) DegreeDistribution {
	if len(hist) == 0 {
		return DegreeDistribution{}
	}
	degrees := make([]int, 0, len(hist))
	total := 0
	sum := 0
	minDegree := math.MaxInt
	maxDegree := 0
	for degree, count := range hist {
		if count <= 0 {
			continue
		}
		degrees = append(degrees, degree)
		total += count
		sum += degree * count
		if degree < minDegree {
			minDegree = degree
		}
		if degree > maxDegree {
			maxDegree = degree
		}
	}
	if total == 0 {
		return DegreeDistribution{}
	}
	sort.Ints(degrees)

	p50 := degrees[len(degrees)-1]
	cumulative := 0
	half := (total + 1) / 2
	for _, degree := range degrees {
		cumulative += hist[degree]
		if cumulative >= half {
			p50 = degree
			break
		}
	}
	return DegreeDistribution{
		Min:  minDegree,
		Max:  maxDegree,
		Mean: float64(sum) / float64(total),
		P50:  p50,
	}
}

// newIndexID returns a short random id such as "idx_1a2b3c4d".
func newIndexID() string {
	var b [4]byte
	if _, err := crand.Read(b[:]); err == nil {
		return "idx_" + hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("idx_%08x", time.Now().UnixNano())
}
