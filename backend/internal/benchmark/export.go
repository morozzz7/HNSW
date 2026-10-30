package benchmark

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
)

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

func ResultCSVHeader() []string {
	return append([]string(nil), resultCSVHeader...)
}

func ScalingCSVHeader() []string {
	return append([]string(nil), scalingCSVHeader...)
}

func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("benchmark: encode json: %w", err)
	}
	return nil
}

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

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 6, 64)
}
