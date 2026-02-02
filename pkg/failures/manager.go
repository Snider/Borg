package failures

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Manager handles the lifecycle of a failure report.
type Manager struct {
	failuresDir string
	runDir      string
	report      *FailureReport
}

// NewManager creates a new failure manager for a given collection.
func NewManager(failuresDir, collection string) (*Manager, error) {
	if failuresDir == "" {
		failuresDir = ".borg-failures"
	}
	runDir := filepath.Join(failuresDir, time.Now().Format("2006-01-02T15-04-05"))
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create failures directory: %w", err)
	}

	return &Manager{
		failuresDir: failuresDir,
		runDir:      runDir,
		report: &FailureReport{
			Collection: collection,
			Started:    time.Now(),
		},
	}, nil
}

// RecordFailure records a single failure.
func (m *Manager) RecordFailure(failure *Failure) {
	m.report.Failures = append(m.report.Failures, failure)
	m.report.Stats.Failed++
}

// SetTotal sets the total number of items to be processed.
func (m *Manager) SetTotal(total int) {
	m.report.Stats.Total = total
}

// Finalize completes the failure report, writing it to disk.
func (m *Manager) Finalize() error {
	m.report.Completed = time.Now()
	m.report.Stats.Success = m.report.Stats.Total - m.report.Stats.Failed

	// Write failures.json
	reportPath := filepath.Join(m.runDir, "failures.json")
	reportFile, err := os.Create(reportPath)
	if err != nil {
		return fmt.Errorf("failed to create failures.json: %w", err)
	}
	defer reportFile.Close()

	encoder := json.NewEncoder(reportFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(m.report); err != nil {
		return fmt.Errorf("failed to write failures.json: %w", err)
	}

	// Write retry.sh
	var retryScript strings.Builder
	retryScript.WriteString("#!/bin/bash\n\n")
	for _, failure := range m.report.Failures {
		retryScript.WriteString(fmt.Sprintf("borg collect github repo %s\n", failure.URL))
	}
	retryPath := filepath.Join(m.runDir, "retry.sh")
	if err := os.WriteFile(retryPath, []byte(retryScript.String()), 0755); err != nil {
		return fmt.Errorf("failed to write retry.sh: %w", err)
	}

	return nil
}
