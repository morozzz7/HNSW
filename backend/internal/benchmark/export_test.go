package benchmark

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"hnsw-coursework/internal/hnsw"
)

func sampleResults() []Result {
	return []Result{
		{
			M: 16, EfConstruction: 200, EfSearch: 100,
			Metric: hnsw.MetricL2, UseHeuristic: true, K: 10,
			Recall: 0.968, QPS: 3420.5,
			LatencyP50Ms: 0.12, LatencyP95Ms: 0.31, LatencyP99Ms: 0.58,
			BuildTimeMs: 412, BruteforceQPS: 12.4, Speedup: 275.8,
		},
		{
			M: 8, EfConstruction: 100, EfSearch: 50,
			Metric: hnsw.MetricCosine, UseHeuristic: false, K: 10,
			Recall: 0.5, QPS: 1000,
			LatencyP50Ms: 1, LatencyP95Ms: 2, LatencyP99Ms: 3,
			BuildTimeMs: 100, BruteforceQPS: 10, Speedup: 100,
		},
	}
}

func TestWriteCSV(t *testing.T) {
	results := sampleResults()
	var buf bytes.Buffer
	if err := WriteCSV(&buf, results); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}

	records, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(records) != len(results)+1 {
		t.Fatalf("csv lines = %d, want %d", len(records), len(results)+1)
	}
	wantHeader := ResultCSVHeader()
	if !reflect.DeepEqual(records[0], wantHeader) {
		t.Fatalf("header = %v, want %v", records[0], wantHeader)
	}

	// Stable header: the same columns are produced regardless of k.
	var other bytes.Buffer
	otherResults := []Result{{M: 1, EfConstruction: 1, EfSearch: 1, K: 100}}
	if err := WriteCSV(&other, otherResults); err != nil {
		t.Fatalf("WriteCSV (k=100): %v", err)
	}
	otherRecords, err := csv.NewReader(&other).ReadAll()
	if err != nil {
		t.Fatalf("parse csv (k=100): %v", err)
	}
	if !reflect.DeepEqual(otherRecords[0], wantHeader) {
		t.Fatalf("header changed with k: %v vs %v", otherRecords[0], wantHeader)
	}

	first := records[1]
	if first[0] != "16" || first[1] != "200" || first[2] != "100" {
		t.Fatalf("first row M/efC/efSearch = %v/%v/%v, want 16/200/100", first[0], first[1], first[2])
	}
	if first[3] != "l2" || first[4] != "true" || first[5] != "10" {
		t.Fatalf("first row metric/heuristic/k = %v/%v/%v, want l2/true/10", first[3], first[4], first[5])
	}
	if first[6] != "0.968000" {
		t.Fatalf("first row recall = %q, want 0.968000", first[6])
	}
	if first[13] != "275.800000" {
		t.Fatalf("first row speedup = %q, want 275.800000", first[13])
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, sampleResults()); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	var rows []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rows); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	for i, row := range rows {
		if _, ok := row["recall"]; ok {
			t.Errorf("row %d: unexpected plain \"recall\" key", i)
		}
		recall, ok := row["recall@10"].(float64)
		if !ok {
			t.Fatalf("row %d: missing numeric \"recall@10\" key: %v", i, row)
		}
		if i == 0 && math.Abs(recall-0.968) > 1e-9 {
			t.Errorf("row 0: recall@10 = %v, want 0.968", recall)
		}
		if got := row["M"].(float64); got != 16 && i == 0 {
			t.Errorf("row 0: M = %v, want 16", got)
		}
		if got := row["heuristic"].(bool); got != (i == 0) {
			t.Errorf("row %d: heuristic = %v", i, got)
		}
		for _, key := range []string{
			"qps", "latencyP50Ms", "latencyP95Ms", "latencyP99Ms",
			"buildTimeMs", "bruteforceQps", "speedup",
		} {
			if _, ok := row[key]; !ok {
				t.Errorf("row %d: missing key %q", i, key)
			}
		}
	}
}

func TestExportFiles(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "results.csv")
	jsonPath := filepath.Join(dir, "results.json")

	results := sampleResults()
	if err := WriteCSVFile(csvPath, results); err != nil {
		t.Fatalf("WriteCSVFile: %v", err)
	}
	if err := WriteJSONFile(jsonPath, results); err != nil {
		t.Fatalf("WriteJSONFile: %v", err)
	}

	csvData, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	if lines := strings.Count(string(csvData), "\n"); lines != len(results)+1 {
		t.Fatalf("csv lines = %d, want %d", lines, len(results)+1)
	}

	jsonData, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read json: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(jsonData, &rows); err != nil {
		t.Fatalf("unmarshal json file: %v", err)
	}
	if len(rows) != len(results) {
		t.Fatalf("json rows = %d, want %d", len(rows), len(results))
	}
}

func TestWriteScalingCSVAndJSON(t *testing.T) {
	results := []ScalingResult{
		{
			N: 1000, Dim: 16, M: 16, EfConstruction: 200, EfSearch: 100,
			Metric: hnsw.MetricL2, UseHeuristic: true, K: 5, Queries: 10,
			Recall: 0.9, QPS: 5000, LatencyP50Ms: 0.2, LatencyP95Ms: 0.4, LatencyP99Ms: 0.6,
			BuildTimeMs: 120, BruteforceQPS: 20, Speedup: 250,
		},
		{
			N: 10000, Dim: 16, M: 16, EfConstruction: 200, EfSearch: 100,
			Metric: hnsw.MetricL2, UseHeuristic: true, K: 5, Queries: 10,
			Recall: 0.85, QPS: 4000, LatencyP50Ms: 0.25, LatencyP95Ms: 0.5, LatencyP99Ms: 0.7,
			BuildTimeMs: 1400, BruteforceQPS: 2, Speedup: 2000,
		},
	}

	var csvBuf bytes.Buffer
	if err := WriteScalingCSV(&csvBuf, results); err != nil {
		t.Fatalf("WriteScalingCSV: %v", err)
	}
	records, err := csv.NewReader(&csvBuf).ReadAll()
	if err != nil {
		t.Fatalf("parse scaling csv: %v", err)
	}
	if len(records) != len(results)+1 {
		t.Fatalf("scaling csv lines = %d, want %d", len(records), len(results)+1)
	}
	if !reflect.DeepEqual(records[0], ScalingCSVHeader()) {
		t.Fatalf("scaling header = %v, want %v", records[0], ScalingCSVHeader())
	}
	if records[1][0] != "1000" || records[2][0] != "10000" {
		t.Fatalf("scaling N column = %q, %q; want 1000, 10000", records[1][0], records[2][0])
	}

	var jsonBuf bytes.Buffer
	if err := WriteJSON(&jsonBuf, results); err != nil {
		t.Fatalf("WriteJSON scaling: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(jsonBuf.Bytes(), &rows); err != nil {
		t.Fatalf("unmarshal scaling json: %v", err)
	}
	if _, ok := rows[0]["recall@5"]; !ok {
		t.Fatalf("scaling json missing recall@5 key: %v", rows[0])
	}
	if got := rows[1]["n"].(float64); got != 10000 {
		t.Fatalf("scaling json n = %v, want 10000", got)
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestWriteCSVPropagatesWriterError(t *testing.T) {
	want := errors.New("disk full")
	err := WriteCSV(failingWriter{err: want}, sampleResults())
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}
