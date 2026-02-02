package progress

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	progressFile := filepath.Join(tmpDir, ".borg-progress")

	now := time.Now()
	p := &Progress{
		Source:    "test:source",
		StartedAt: now,
		Completed: []string{"item1", "item2"},
		Pending:   []string{"item3", "item4"},
		Failed:    []string{"item5"},
	}

	if err := p.Save(progressFile); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	loaded, err := Load(progressFile)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Truncate for comparison, as JSON marshaling can lose precision.
	p.StartedAt = p.StartedAt.Truncate(time.Second)
	loaded.StartedAt = loaded.StartedAt.Truncate(time.Second)

	if diff := cmp.Diff(p, loaded); diff != "" {
		t.Errorf("Loaded progress does not match saved progress (-want +got):\n%s", diff)
	}
}

func TestLoad_FileNotExists(t *testing.T) {
	_, err := Load("non-existent-file")
	if !os.IsNotExist(err) {
		t.Errorf("Expected a not-exist error, but got %v", err)
	}
}
