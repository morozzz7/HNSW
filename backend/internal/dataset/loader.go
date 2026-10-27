// Package dataset provides loaders for CSV, JSON and .fvecs/.ivecs vector files.
package dataset

import (
	"bytes"
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// LoadCSV reads vectors from a CSV reader.
// Each data row must contain the same number of float fields. If the first row
// contains any non-float field it is treated as a header and skipped.
func LoadCSV(r io.Reader) ([][]float32, error) {
	reader := csv.NewReader(r)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1 // allow variable-length records during read
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv: %w", err)
	}
	if len(records) == 0 {
		return nil, errors.New("csv: no records")
	}

	start := 0
	if !isFloatRow(records[0]) {
		start = 1 // header row
	}
	if start >= len(records) {
		return nil, errors.New("csv: header row present but no data rows")
	}

	dim := len(records[start])
	vectors := make([][]float32, 0, len(records)-start)
	for i := start; i < len(records); i++ {
		rec := records[i]
		if len(rec) != dim {
			return nil, fmt.Errorf("csv: row %d has %d fields, want %d", i, len(rec), dim)
		}
		vec := make([]float32, dim)
		for j, f := range rec {
			v, err := strconv.ParseFloat(strings.TrimSpace(f), 32)
			if err != nil {
				return nil, fmt.Errorf("csv: row %d field %d is not a float: %w", i, j, err)
			}
			vec[j] = float32(v)
		}
		vectors = append(vectors, vec)
	}
	return vectors, nil
}

// LoadJSON reads vectors from a JSON reader.
// Accepted top-level shapes:
//   - a bare array of arrays: [[1,2], [3,4]]
//   - an object with a "vectors" field: {"vectors": [[1,2], [3,4]]}
func LoadJSON(r io.Reader) ([][]float32, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("json: read: %w", err)
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, errors.New("json: empty document")
	}

	// Bare array of arrays.
	var bare [][]float64
	if err := json.Unmarshal(data, &bare); err == nil && len(bare) > 0 {
		return toFloat32(bare), nil
	}

	// Object wrapper.
	var wrapper struct {
		Vectors [][]float64 `json:"vectors"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("json: parse: %w", err)
	}
	if len(wrapper.Vectors) == 0 {
		return nil, errors.New("json: no vectors found")
	}
	return toFloat32(wrapper.Vectors), nil
}

// LoadFile dispatches to the loader matching the file extension.
// Supported extensions: .csv, .json, .fvecs, .ivecs.
func LoadFile(path string) ([][]float32, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".csv":
		return loadPathCSV(path)
	case ".json":
		return loadPathJSON(path)
	case ".fvecs":
		return loadBinary(path, false)
	case ".ivecs":
		return loadBinary(path, true)
	default:
		return nil, fmt.Errorf("dataset: unsupported file extension %q", ext)
	}
}

func loadPathCSV(path string) ([][]float32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return LoadCSV(f)
}

func loadPathJSON(path string) ([][]float32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return LoadJSON(f)
}

// isFloatRow reports whether every field in the row parses as a float.
func isFloatRow(row []string) bool {
	if len(row) == 0 {
		return false
	}
	for _, f := range row {
		if _, err := strconv.ParseFloat(strings.TrimSpace(f), 64); err != nil {
			return false
		}
	}
	return true
}

// toFloat32 converts [][]float64 to [][]float32.
func toFloat32(src [][]float64) [][]float32 {
	out := make([][]float32, len(src))
	for i, row := range src {
		out[i] = make([]float32, len(row))
		for j, v := range row {
			out[i][j] = float32(v)
		}
	}
	return out
}

// loadBinary reads an .fvecs or .ivecs file.
// Format: repeated records of int32 dimension followed by `dim` elements
// (float32 for .fvecs, int32 for .ivecs). All records must share a dimension.
func loadBinary(path string, isIVecs bool) ([][]float32, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseBinary(data, isIVecs)
}

func parseBinary(data []byte, isIVecs bool) ([][]float32, error) {
	r := bytes.NewReader(data)
	var vectors [][]float32
	dim := -1
	for r.Len() > 0 {
		var d int32
		if err := binary.Read(r, binary.LittleEndian, &d); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("%s: read dimension: %w", extLabel(isIVecs), err)
		}
		if d <= 0 || int(d) > MaxDim {
			return nil, fmt.Errorf("%s: invalid dimension %d", extLabel(isIVecs), d)
		}
		vec := make([]float32, d)
		if isIVecs {
			tmp := make([]int32, d)
			if err := binary.Read(r, binary.LittleEndian, tmp); err != nil {
				return nil, fmt.Errorf("%s: read values: %w", extLabel(isIVecs), err)
			}
			for j, v := range tmp {
				vec[j] = float32(v)
			}
		} else {
			buf := make([]byte, 4*d)
			if _, err := io.ReadFull(r, buf); err != nil {
				return nil, fmt.Errorf("%s: read values: %w", extLabel(isIVecs), err)
			}
			dimI := int(d)
			for j := 0; j < dimI; j++ {
				vec[j] = math.Float32frombits(binary.LittleEndian.Uint32(buf[4*j:]))
			}
		}
		if dim == -1 {
			dim = int(d)
		} else if int(d) != dim {
			return nil, fmt.Errorf("%s: inconsistent dimension %d, want %d", extLabel(isIVecs), d, dim)
		}
		vectors = append(vectors, vec)
	}
	if len(vectors) == 0 {
		return nil, fmt.Errorf("%s: no vectors found", extLabel(isIVecs))
	}
	return vectors, nil
}

func extLabel(isIVecs bool) string {
	if isIVecs {
		return "ivecs"
	}
	return "fvecs"
}
