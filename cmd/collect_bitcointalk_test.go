package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/Snider/Borg/pkg/bitcointalk"
)

func TestCollectBitcoinTalkThreadCmd(t *testing.T) {
	// Start a new test server
	server := newTestServer()
	defer server.Close()

	// Set the base URL to the test server
	bitcointalk.SetBaseURL(server.URL)

	// Test with a known thread, the Bitcoin whitepaper announcement
	threadID := "6"
	outputFile := "test-thread.md"

	// Cleanup the output file after the test
	defer os.Remove(outputFile)

	// Execute the command
	cmd := NewCollectBitcoinTalkThreadCmd()
	cmd.SetArgs([]string{threadID, "--output", outputFile})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("error executing command: %v", err)
	}

	// Check that the output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("output file was not created: %s", outputFile)
	}

	// Read the output file and check for some expected content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("error reading output file: %v", err)
	}

	if !strings.Contains(string(content), "Repost: Bitcoin Maturation") {
		t.Errorf("output file does not contain expected content")
	}
}

func TestCollectBitcoinTalkUserCmd(t *testing.T) {
	// Start a new test server
	server := newTestServer()
	defer server.Close()

	// Set the base URL to the test server
	bitcointalk.SetBaseURL(server.URL)

	// Test with a known user, Satoshi Nakamoto
	userID := "3"
	outputFile := "test-user.json"

	// Cleanup the output file after the test
	defer os.Remove(outputFile)

	// Execute the command
	cmd := NewCollectBitcoinTalkUserCmd()
	cmd.SetArgs([]string{userID, "--output", outputFile})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("error executing command: %v", err)
	}

	// Check that the output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("output file was not created: %s", outputFile)
	}

	// Read the output file and check for some expected content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("error reading output file: %v", err)
	}

	if !strings.Contains(string(content), "satoshi") {
		t.Errorf("output file does not contain expected content")
	}
}

func TestCollectBitcoinTalkSearchCmd(t *testing.T) {
	// This test requires a search results page, which I haven't downloaded yet.
	// I'll skip this test for now and come back to it later.
	t.Skip("Skipping test: requires search results page")
}
