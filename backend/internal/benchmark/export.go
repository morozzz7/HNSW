package benchmark

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
)

// CSV headers are stable: every run of a given exporter produces the same
// columns, independently of k. Recall is exported as a plain "recall" column;
// the JSON encoder uses "recall@<k>" to match the API contract.
var (
	resultCSVHeader = []string{
		"M", "efConstruction", "efSearch", "metric", "heuristic", "k",
		"recall", "qps", "latencyP50Ms", "latencyP95Ms", "latencyP99Ms",
		"buildTimeMs", "bruteforceQps", "speedup",
	}
	scalingCSVHeader = []string{
		"n", "dim", "M", "efConstruction", "efSearch", "metric", "heuristic",
		"k", "queries", "recall", "qps", "latencyP50Ms", "latencyP95Ms",
		"latencyP99Ms", "buildTimeMs", "bruteforceQps", "speedup",
	}
)

// ResultCSVHeader returns a copy of the stable CSV header used by WriteCSV.
func ResultCSVHeader() []string {
	return append([]string(nil), resultCSVHeader...)
}

// ScalingCSVHeader returns a copy of the stable CSV header used by
// WriteScalingCSV.
func ScalingCSVHeader() []string {
	return append([]string(nil), scalingCSVHeader...)
}

// WriteJSON writes v as indented JSON to w. Grid runs pass []Result and scaling
// runs pass []ScalingResult; both use their MarshalJSON methods, so recall is
// encoded as "recall@<k>".
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("benchmark: encode json: %w", err)
	}
	return nil
}

// WriteJSONFile writes v as indented JSON to path, creating or truncating it.
func WriteJSONFile(path string, v any) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("benchmark: create %s: %w", path, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("benchmark: close %s: %w", path, cerr)
		}
	}()
	if err := WriteJSON(f, v); err != nil {
		return err
	}
	return nil
}

// WriteCSV writes grid results to w as CSV with a stable header and one row per
// result. The writer is flushed before returning.
func WriteCSV(w io.Writer, results []Result) error {
	rows := make([][]string, len(results))
	for i, r := range results {
		rows[i] = []string{
			strconv.Itoa(r.M),
			strconv.Itoa(r.EfConstruction),
			strconv.Itoa(r.EfSearch),
			string(r.Metric),
			strconv.FormatBool(r.UseHeuristic),
			strconv.Itoa(r.K),
			formatFloat(r.Recall),
			formatFloat(r.QPS),
			formatFloat(r.LatencyP50Ms),
			formatFloat(r.LatencyP95Ms),
			formatFloat(r.LatencyP99Ms),
			formatFloat(r.BuildTimeMs),
			formatFloat(r.BruteforceQPS),
			formatFloat(r.Speedup),
		}
	}
	return writeCSV(w, resultCSVHeader, rows)
}

// WriteCSVFile writes grid results to path as CSV.
func WriteCSVFile(path string, results []Result) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("benchmark: create %s: %w", path, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("benchmark: close %s: %w", path, cerr)
		}
	}()
	return WriteCSV(f, results)
}

// WriteScalingCSV writes scaling results to w as CSV with a stable header and
// one row per N.
func WriteScalingCSV(w io.Writer, results []ScalingResult) error {
	rows := make([][]string, len(results))
	for i, r := range results {
		rows[i] = []string{
			strconv.Itoa(r.N),
			strconv.Itoa(r.Dim),
			strconv.Itoa(r.M),
			strconv.Itoa(r.EfConstruction),
			strconv.Itoa(r.EfSearch),
			string(r.Metric),
			strconv.FormatBool(r.UseHeuristic),
			strconv.Itoa(r.K),
			strconv.Itoa(r.Queries),
			formatFloat(r.Recall),
			formatFloat(r.QPS),
			formatFloat(r.LatencyP50Ms),
			formatFloat(r.LatencyP95Ms),
			formatFloat(r.LatencyP99Ms),
			formatFloat(r.BuildTimeMs),
			formatFloat(r.BruteforceQPS),
			formatFloat(r.Speedup),
		}
	}
	return writeCSV(w, scalingCSVHeader, rows)
}

// WriteScalingCSVFile writes scaling results to path as CSV.
func WriteScalingCSVFile(path string, results []ScalingResult) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("benchmark: create %s: %w", path, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("benchmark: close %s: %w", path, cerr)
		}
	}()
	return WriteScalingCSV(f, results)
}

// writeCSV writes a header and rows, flushing the underlying writer.
func writeCSV(w io.Writer, header []string, rows [][]string) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(header); err != nil {
		return fmt.Errorf("benchmark: write csv header: %w", err)
	}
	for _, row := range rows {
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("benchmark: write csv row: %w", err)
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return fmt.Errorf("benchmark: flush csv: %w", err)
	}
	return nil
}

// formatFloat formats a metric with six decimal places, which keeps report
// tables readable while retaining enough precision for QPS and latency.
func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 6, 64)
}
