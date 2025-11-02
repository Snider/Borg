package cmd

import (
	"testing"
)

func TestExecute(t *testing.T) {
	// This test simply checks that the Execute function can be called without error.
	// It doesn't actually test any of the application's functionality.
	if err := Execute(); err != nil {
		t.Errorf("Execute() failed: %v", err)
	}
}
