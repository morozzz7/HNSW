package indexmanager

import (
	"errors"
	"testing"
	"time"

	"hnsw-coursework/internal/dataset"
	"hnsw-coursework/internal/hnsw"
)

func newTestManager(t *testing.T, n, dim int) (*Manager, *dataset.Dataset) {
	t.Helper()
	vectors, err := dataset.Generate(n, dim, "uniform", 1)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	ds, err := dataset.NewDataset("generate", "uniform", vectors)
	if err != nil {
		t.Fatalf("new dataset: %v", err)
	}
	store := dataset.NewStore()
	store.Add(ds)
	mgr := New(store)
	t.Cleanup(mgr.Close)
	return mgr, ds
}

func waitStatus(t *testing.T, mgr *Manager, want Status) State {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		st := mgr.Status()
		if st.Status == want {
			return st
		}
		time.Sleep(2 * time.Millisecond)
	}
	st := mgr.Status()
	t.Fatalf("status = %s (%s), want %s", st.Status, st.Error, want)
	return State{}
}

func TestBuildSearchInsertAndStats(t *testing.T) {
	mgr, ds := newTestManager(t, 128, 8)

	indexID, err := mgr.StartBuild(BuildConfig{
		DatasetID:      ds.ID,
		M:              8,
		EfConstruction: 50,
		Metric:         hnsw.MetricL2,
		UseHeuristic:   true,
		Seed:           1,
	})
	if err != nil {
		t.Fatalf("start build: %v", err)
	}
	st := waitStatus(t, mgr, StatusReady)
	if st.IndexID != indexID {
		t.Fatalf("indexID = %q, want %q", st.IndexID, indexID)
	}
	if st.Progress != 1 {
		t.Fatalf("progress = %v, want 1", st.Progress)
	}

	stats, err := mgr.Stats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.Nodes != 128 {
		t.Fatalf("nodes = %d, want 128", stats.Nodes)
	}
	if stats.Metric != hnsw.MetricL2 {
		t.Fatalf("metric = %s, want l2", stats.Metric)
	}
	if stats.DatasetID != ds.ID {
		t.Fatalf("datasetID = %q, want %q", stats.DatasetID, ds.ID)
	}

	q := ds.Vectors[0]
	res, trace, err := mgr.HNSWSearch(q, 5, 10, true)
	if err != nil {
		t.Fatalf("hnsw search: %v", err)
	}
	if len(res) != 5 {
		t.Fatalf("hnsw results = %d, want 5", len(res))
	}
	if res[0].Internal != 0 || res[0].VectorID != "0" {
		t.Fatalf("hnsw top = %d/%q, want 0/0", res[0].Internal, res[0].VectorID)
	}
	if res[0].Distance != 0 {
		t.Fatalf("hnsw top distance = %v, want 0", res[0].Distance)
	}
	if trace == nil || len(trace.Layers) == 0 {
		t.Fatal("hnsw trace is empty")
	}

	exact, err := mgr.BruteForceSearch(q, 5)
	if err != nil {
		t.Fatalf("bruteforce search: %v", err)
	}
	if exact[0].Internal != 0 {
		t.Fatalf("bruteforce top = %d, want 0", exact[0].Internal)
	}

	newVec := make([]float32, len(q))
	copy(newVec, q)
	newVec[0] += 0.5
	internal, assigned, err := mgr.InsertVector(newVec, "")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if internal != 128 || assigned != "ext_0" {
		t.Fatalf("insert = %d/%q, want 128/ext_0", internal, assigned)
	}
	if mgr.Len() != 129 {
		t.Fatalf("len = %d, want 129", mgr.Len())
	}

	got, ok := mgr.VectorByID("ext_0")
	if !ok || len(got) != len(newVec) || got[0] != newVec[0] {
		t.Fatalf("VectorByID(ext_0) = %v, %v", got, ok)
	}
	res, _, err = mgr.HNSWSearch(newVec, 1, 10, false)
	if err != nil {
		t.Fatalf("search inserted: %v", err)
	}
	if res[0].VectorID != "ext_0" || res[0].Distance != 0 {
		t.Fatalf("inserted top = %q/%v, want ext_0/0", res[0].VectorID, res[0].Distance)
	}
	exact, err = mgr.BruteForceSearch(newVec, 1)
	if err != nil {
		t.Fatalf("bruteforce inserted: %v", err)
	}
	if exact[0].VectorID != "ext_0" {
		t.Fatalf("bruteforce inserted top = %q, want ext_0", exact[0].VectorID)
	}

	if _, _, err := mgr.InsertVector(newVec, "ext_0"); !errors.Is(err, ErrDuplicateID) {
		t.Fatalf("duplicate insert error = %v, want ErrDuplicateID", err)
	}
	if _, _, err := mgr.InsertVector([]float32{1, 2}, "bad"); !errors.Is(err, ErrDimension) {
		t.Fatalf("wrong dimension error = %v, want ErrDimension", err)
	}

	stats, err = mgr.Stats()
	if err != nil {
		t.Fatalf("stats after insert: %v", err)
	}
	if stats.Nodes != 129 {
		t.Fatalf("nodes after insert = %d, want 129", stats.Nodes)
	}
}

func TestSecondBuildRejectedAndCloseCancels(t *testing.T) {
	mgr, ds := newTestManager(t, 200_000, 4)
	cfg := BuildConfig{
		DatasetID:      ds.ID,
		M:              8,
		EfConstruction: 50,
		Metric:         hnsw.MetricL2,
		UseHeuristic:   true,
	}
	if _, err := mgr.StartBuild(cfg); err != nil {
		t.Fatalf("first build: %v", err)
	}
	if _, err := mgr.StartBuild(cfg); !errors.Is(err, ErrBuildInProgress) {
		t.Fatalf("second build error = %v, want ErrBuildInProgress", err)
	}

	mgr.Close()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		st := mgr.Status()
		if st.Status == StatusError {
			if st.Error == "" {
				t.Fatal("cancelled build has empty error message")
			}
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("build was not cancelled, status = %s", mgr.Status().Status)
}

func TestNotReadyErrors(t *testing.T) {
	mgr, _ := newTestManager(t, 4, 2)
	if _, _, err := mgr.HNSWSearch([]float32{1, 2}, 1, 10, false); !errors.Is(err, ErrNoIndex) {
		t.Fatalf("search idle error = %v, want ErrNoIndex", err)
	}
	if _, err := mgr.BruteForceSearch([]float32{1, 2}, 1); !errors.Is(err, ErrNoIndex) {
		t.Fatalf("bruteforce idle error = %v, want ErrNoIndex", err)
	}
	if _, err := mgr.Stats(); !errors.Is(err, ErrNoIndex) {
		t.Fatalf("stats idle error = %v, want ErrNoIndex", err)
	}
	if _, _, err := mgr.InsertVector([]float32{1, 2}, ""); !errors.Is(err, ErrNoIndex) {
		t.Fatalf("insert idle error = %v, want ErrNoIndex", err)
	}
	if _, err := mgr.StartBuild(BuildConfig{DatasetID: "missing", M: 8, EfConstruction: 50, Metric: hnsw.MetricL2}); !errors.Is(err, ErrDatasetNotFound) {
		t.Fatalf("missing dataset error = %v, want ErrDatasetNotFound", err)
	}
	if _, err := mgr.StartBuild(BuildConfig{DatasetID: "x", M: -1, EfConstruction: 50, Metric: hnsw.MetricL2}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid M error = %v, want ErrInvalid", err)
	}
}
