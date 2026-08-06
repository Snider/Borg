package failures

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "borg-failures-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager, err := NewManager(tempDir, "test-collection")
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	manager.SetTotal(1)
	manager.RecordFailure(&Failure{
		URL:       "http://example.com/failed",
		Error:     "test error",
		Retryable: true,
	})

	if err := manager.Finalize(); err != nil {
		t.Fatalf("failed to finalize manager: %v", err)
	}

	// Verify failures.json
	reportPath := filepath.Join(manager.runDir, "failures.json")
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Fatalf("failures.json was not created")
	}

	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("failed to read failures.json: %v", err)
	}

	var report FailureReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("failed to unmarshal failures.json: %v", err)
	}

	if report.Collection != "test-collection" {
		t.Errorf("expected collection 'test-collection', got '%s'", report.Collection)
	}
	if len(report.Failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(report.Failures))
	}
	if report.Failures[0].URL != "http://example.com/failed" {
		t.Errorf("unexpected failure URL: %s", report.Failures[0].URL)
	}

	// Verify retry.sh
	retryPath := filepath.Join(manager.runDir, "retry.sh")
	if _, err := os.Stat(retryPath); os.IsNotExist(err) {
		t.Fatalf("retry.sh was not created")
	}

	retryScript, err := os.ReadFile(retryPath)
	if err != nil {
		t.Fatalf("failed to read retry.sh: %v", err)
	}

	if !strings.Contains(string(retryScript), "http://example.com/failed") {
		t.Errorf("retry.sh does not contain the failed URL")
	}
}
