package yellow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadJSONRequiresRegularFile(t *testing.T) {
	var value any
	dir := t.TempDir()
	if err := readJSON(dir, &value); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("directory read error = %v, want regular-file rejection", err)
	}
	data := filepath.Join(dir, "data.json")
	if err := os.WriteFile(data, []byte(`{"version":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "data-link.json")
	if err := os.Symlink(data, link); err == nil {
		if err := readJSON(link, &value); err == nil || !strings.Contains(err.Error(), "regular file") {
			t.Fatalf("symlink read error = %v, want regular-file rejection", err)
		}
	}
	if err := readJSON(data, &value); err != nil {
		t.Fatalf("regular JSON file rejected: %v", err)
	}
}
