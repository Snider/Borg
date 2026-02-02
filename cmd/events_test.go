package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Snider/Borg/pkg/events"
)

type mockGithubClient struct {
	repos []string
}

func (m *mockGithubClient) GetPublicRepos(ctx context.Context, userOrOrg string) ([]string, error) {
	return m.repos, nil
}

func TestCollectGithubReposCmd_Events(t *testing.T) {
	// Mock the GithubClient
	originalClient := GithubClient
	GithubClient = &mockGithubClient{repos: []string{"repo1", "repo2"}}
	defer func() { GithubClient = originalClient }()

	// Capture output
	var output bytes.Buffer
	collectGithubReposCmd.SetOut(&output)
	defer collectGithubReposCmd.SetOut(nil)

	// Execute the command's RunE function
	eventsFlag = true
	if err := collectGithubReposCmd.RunE(collectGithubReposCmd, []string{"test-org"}); err != nil {
		t.Fatalf("Failed to execute command: %v", err)
	}
	eventsFlag = false

	// Verify the events
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 6 {
		t.Fatalf("Expected 6 events, got %d", len(lines))
	}

	expectedEvents := []events.EventType{
		events.CollectionStarted,
		events.ItemStarted,
		events.ItemCompleted,
		events.ItemStarted,
		events.ItemCompleted,
		events.CollectionCompleted,
	}

	for i, line := range lines {
		var event events.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("Failed to unmarshal event from line %d: %v", i, err)
		}

		if event.Event != expectedEvents[i] {
			t.Errorf("Expected event type %s, got %s", expectedEvents[i], event.Event)
		}
	}
}
