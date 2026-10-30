package dataset

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCSVNoHeader(t *testing.T) {
	data := "0.1,0.2,0.3\n0.4,0.5,0.6\n"
	vecs, err := LoadCSV(strings.NewReader(data))
	if err != nil {
		t.Fatalf("LoadCSV = %v, want nil", err)
	}
	if len(vecs) != 2 || len(vecs[0]) != 3 {
		t.Fatalf("got %d vectors of dim %d, want 2 x 3", len(vecs), len(vecs[0]))
	}
	if vecs[1][2] != 0.6 {
		t.Fatalf("vecs[1][2] = %v, want 0.6", vecs[1][2])
	}
}

func TestLoadCSVWithHeader(t *testing.T) {
	data := "x,y,z\n0.1,0.2,0.3\n0.4,0.5,0.6\n"
	vecs, err := LoadCSV(strings.NewReader(data))
	if err != nil {
		t.Fatalf("LoadCSV(header) = %v, want nil", err)
	}
	if len(vecs) != 2 {
		t.Fatalf("len = %d, want 2 (header skipped)", len(vecs))
	}
}

func TestLoadCSVRowLengthMismatch(t *testing.T) {
	data := "0.1,0.2\n0.4,0.5,0.6\n"
	if _, err := LoadCSV(strings.NewReader(data)); err == nil {
		t.Fatal("LoadCSV(mismatch) = nil, want error")
	}
}

func TestLoadCSVEmpty(t *testing.T) {
	if _, err := LoadCSV(strings.NewReader("")); err == nil {
		t.Fatal("LoadCSV(empty) = nil, want error")
	}
}

func TestLoadJSONBareArray(t *testing.T) {
	data := `[[1,2,3],[4,5,6]]`
	vecs, err := LoadJSON(strings.NewReader(data))
	if err != nil {
		t.Fatalf("LoadJSON(bare) = %v, want nil", err)
	}
	if len(vecs) != 2 || len(vecs[0]) != 3 {
		t.Fatalf("got %d x %d, want 2 x 3", len(vecs), len(vecs[0]))
	}
	if vecs[0][0] != 1 || vecs[1][2] != 6 {
		t.Fatalf("unexpected values: %v", vecs)
	}
}

func TestLoadJSONWrapper(t *testing.T) {
	data := `{"vectors":[[1,2],[3,4]]}`
	vecs, err := LoadJSON(strings.NewReader(data))
	if err != nil {
		t.Fatalf("LoadJSON(wrapper) = %v, want nil", err)
	}
	if len(vecs) != 2 || len(vecs[0]) != 2 {
		t.Fatalf("got %d x %d, want 2 x 2", len(vecs), len(vecs[0]))
	}
}

func TestLoadJSONEmpty(t *testing.T) {
	if _, err := LoadJSON(strings.NewReader("")); err == nil {
		t.Fatal("LoadJSON(empty) = nil, want error")
	}
}

func writeFVecsFile(t *testing.T, vecs [][]float32) string {
	t.Helper()
	var buf bytes.Buffer
	for _, v := range vecs {
		if err := binary.Write(&buf, binary.LittleEndian, int32(len(v))); err != nil {
			t.Fatal(err)
		}
		for _, x := range v {
			if err := binary.Write(&buf, binary.LittleEndian, x); err != nil {
				t.Fatal(err)
			}
		}
	}
	path := filepath.Join(t.TempDir(), "data.fvecs")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadFVecs(t *testing.T) {
	vecs := [][]float32{{1, 2, 3}, {4, 5, 6}}
	path := writeFVecsFile(t, vecs)
	got, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile(fvecs) = %v, want nil", err)
	}
	if len(got) != 2 || len(got[0]) != 3 {
		t.Fatalf("got %d x %d, want 2 x 3", len(got), len(got[0]))
	}
	if got[0][0] != 1 || got[1][2] != 6 {
		t.Fatalf("unexpected: %v", got)
	}
}

func TestLoadIVecs(t *testing.T) {
	ivecs := [][]int32{{1, 2}, {3, 4}}
	var buf bytes.Buffer
	for _, v := range ivecs {
		if err := binary.Write(&buf, binary.LittleEndian, int32(len(v))); err != nil {
			t.Fatal(err)
		}
		for _, x := range v {
			if err := binary.Write(&buf, binary.LittleEndian, x); err != nil {
				t.Fatal(err)
			}
		}
	}
	path := filepath.Join(t.TempDir(), "data.ivecs")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile(ivecs) = %v, want nil", err)
	}
	if len(got) != 2 || got[0][0] != 1 || got[1][1] != 4 {
		t.Fatalf("unexpected: %v", got)
	}
}

func TestLoadFileUnsupported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(path); err == nil {
		t.Fatal("LoadFile(unsupported) = nil, want error")
	}
}
