// Package depositidtest loads the shared deposit-ID golden vectors. Test use
// only.
package depositidtest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Vectors returns the vectors of section chain in
// testdata/deposit_id_vectors.json at the repository root, decoded into V. It
// fails t if the file or the section is missing or has no vectors.
func Vectors[V any](t testing.TB, chain string) []V {
	t.Helper()
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate golden vectors: no caller information")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(self), "../../../../testdata/deposit_id_vectors.json"))
	if err != nil {
		t.Fatalf("read golden vectors: %v", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse golden vectors: %v", err)
	}
	body, ok := doc[chain]
	if !ok {
		t.Fatalf("golden vectors file has no %q section", chain)
	}
	var section struct {
		Vectors []V `json:"vectors"`
	}
	if err := json.Unmarshal(body, &section); err != nil {
		t.Fatalf("parse golden vectors: %v", err)
	}
	if len(section.Vectors) == 0 {
		t.Fatal("golden vectors file has no vectors")
	}
	return section.Vectors
}
