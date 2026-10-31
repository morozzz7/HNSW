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

type loadLimits struct {
	maxVectors int
	maxDim     int
	maxValues  int64
}

var defaultLoadLimits = loadLimits{maxVectors: MaxVectors, maxDim: MaxDim, maxValues: MaxValues}

func LoadCSV(r io.Reader) ([][]float32, error) {
	return loadCSV(r, defaultLoadLimits)
}

func loadCSV(r io.Reader, lim loadLimits) ([][]float32, error) {
	reader := csv.NewReader(r)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = true

	first, err := reader.Read()
	if err == io.EOF {
		return nil, errors.New("csv: no records")
	}
	if err != nil {
		return nil, fmt.Errorf("csv: %w", err)
	}

	vectors := make([][]float32, 0, 64)
	dim := 0
	line := 0
	if isFloatRow(first) {
		vec, err := parseCSVRow(first, line, &dim, len(vectors), lim)
		if err != nil {
			return nil, err
		}
		vectors = append(vectors, vec)
		line++
	} else {
		line = 1
	}

	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("csv: %w", err)
		}
		vec, err := parseCSVRow(rec, line, &dim, len(vectors), lim)
		if err != nil {
			return nil, err
		}
		vectors = append(vectors, vec)
		line++
	}

	if len(vectors) == 0 {
		return nil, errors.New("csv: header row present but no data rows")
	}
	return vectors, nil
}

func parseCSVRow(rec []string, line int, dim *int, count int, lim loadLimits) ([]float32, error) {
	if len(rec) == 0 {
		return nil, fmt.Errorf("csv: row %d is empty", line)
	}
	if *dim == 0 {
		if len(rec) > lim.maxDim {
			return nil, fmt.Errorf("csv: dimension %d exceeds limit %d", len(rec), lim.maxDim)
		}
		*dim = len(rec)
	} else if len(rec) != *dim {
		return nil, fmt.Errorf("csv: row %d has %d fields, want %d", line, len(rec), *dim)
	}
	if count >= lim.maxVectors {
		return nil, fmt.Errorf("csv: %d vectors exceeds limit %d", count+1, lim.maxVectors)
	}
	if int64(count+1)*int64(*dim) > lim.maxValues {
		return nil, fmt.Errorf("csv: %d values exceeds limit %d", int64(count+1)*int64(*dim), lim.maxValues)
	}

	vec := make([]float32, *dim)
	for j, f := range rec {
		v, err := strconv.ParseFloat(strings.TrimSpace(f), 32)
		if err != nil {
			return nil, fmt.Errorf("csv: row %d field %d is not a float: %w", line, j, err)
		}
		vec[j] = float32(v)
	}
	return vec, nil
}

func LoadJSON(r io.Reader) ([][]float32, error) {
	return loadJSON(r, defaultLoadLimits)
}

func loadJSON(r io.Reader, lim loadLimits) ([][]float32, error) {
	dec := json.NewDecoder(r)
	tok, err := dec.Token()
	if err == io.EOF {
		return nil, errors.New("json: empty document")
	}
	if err != nil {
		return nil, fmt.Errorf("json: parse: %w", err)
	}

	delim, ok := tok.(json.Delim)
	if !ok {
		return nil, errors.New("json: expected an array or an object")
	}

	var vectors [][]float32
	switch delim {
	case '[':
		vectors, err = readJSONVectors(dec, lim)
	case '{':
		vectors, err = readJSONWrapper(dec, lim)
	default:
		return nil, errors.New("json: expected an array or an object")
	}
	if err != nil {
		return nil, err
	}

	var trailing json.RawMessage
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, errors.New("json: unexpected trailing data")
	}
	return vectors, nil
}

func readJSONVectors(dec *json.Decoder, lim loadLimits) ([][]float32, error) {
	vectors := make([][]float32, 0, 64)
	dim := 0
	for dec.More() {
		var row []float32
		if err := dec.Decode(&row); err != nil {
			return nil, fmt.Errorf("json: parse: %w", err)
		}
		if len(row) == 0 {
			return nil, errors.New("json: zero-dimensional vector")
		}
		if dim == 0 {
			if len(row) > lim.maxDim {
				return nil, fmt.Errorf("json: dimension %d exceeds limit %d", len(row), lim.maxDim)
			}
			dim = len(row)
		} else if len(row) != dim {
			return nil, fmt.Errorf("json: vector %d has dimension %d, want %d", len(vectors), len(row), dim)
		}
		if len(vectors) >= lim.maxVectors {
			return nil, fmt.Errorf("json: %d vectors exceeds limit %d", len(vectors)+1, lim.maxVectors)
		}
		if int64(len(vectors)+1)*int64(dim) > lim.maxValues {
			return nil, fmt.Errorf("json: %d values exceeds limit %d", int64(len(vectors)+1)*int64(dim), lim.maxValues)
		}
		vectors = append(vectors, row)
	}
	if _, err := dec.Token(); err != nil {
		return nil, fmt.Errorf("json: parse: %w", err)
	}
	if len(vectors) == 0 {
		return nil, errors.New("json: no vectors found")
	}
	return vectors, nil
}

func readJSONWrapper(dec *json.Decoder, lim loadLimits) ([][]float32, error) {
	var vectors [][]float32
	found := false
	for dec.More() {
		keyToken, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("json: parse: %w", err)
		}
		key, ok := keyToken.(string)
		if !ok {
			return nil, errors.New("json: object keys must be strings")
		}
		if key != "vectors" {
			var skip json.RawMessage
			if err := dec.Decode(&skip); err != nil {
				return nil, fmt.Errorf("json: parse: %w", err)
			}
			continue
		}
		if found {
			return nil, errors.New("json: duplicate vectors field")
		}
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("json: parse: %w", err)
		}
		if tok != json.Delim('[') {
			return nil, errors.New("json: vectors must be an array")
		}
		vectors, err = readJSONVectors(dec, lim)
		if err != nil {
			return nil, err
		}
		found = true
	}
	if _, err := dec.Token(); err != nil {
		return nil, fmt.Errorf("json: parse: %w", err)
	}
	if !found {
		return nil, errors.New("json: no vectors found")
	}
	return vectors, nil
}

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
		if len(vectors) >= MaxVectors {
			return nil, fmt.Errorf("%s: %d vectors exceeds limit %d", extLabel(isIVecs), len(vectors)+1, MaxVectors)
		}
		if dim == -1 {
			dim = int(d)
		} else if int(d) != dim {
			return nil, fmt.Errorf("%s: inconsistent dimension %d, want %d", extLabel(isIVecs), d, dim)
		}
		if int64(len(vectors)+1)*int64(dim) > MaxValues {
			return nil, fmt.Errorf("%s: %d values exceeds limit %d", extLabel(isIVecs), int64(len(vectors)+1)*int64(dim), int64(MaxValues))
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
