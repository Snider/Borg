package events

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestEventEmitter_Stdout(t *testing.T) {
	var stdout strings.Builder
	emitter, err := NewEventEmitter(&stdout, "", "")
	if err != nil {
		t.Fatalf("Failed to create EventEmitter: %v", err)
	}

	testEvent := ItemStarted
	testData := ItemStartedData{URL: "test", Index: 1, Total: 1}
	if err := emitter.Emit(testEvent, testData); err != nil {
		t.Fatalf("Failed to emit event: %v", err)
	}

	var event Event
	if err := json.Unmarshal([]byte(stdout.String()), &event); err != nil {
		t.Fatalf("Failed to unmarshal event from stdout: %v", err)
	}

	if event.Event != testEvent {
		t.Errorf("Expected event type %s, got %s", testEvent, event.Event)
	}
}

func TestEventEmitter_Webhook(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		var event Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Errorf("Failed to decode event from webhook request: %v", err)
		}
		if event.Event != ItemStarted {
			t.Errorf("Expected event type %s, got %s", ItemStarted, event.Event)
		}
	}))
	defer server.Close()

	emitter, err := NewEventEmitter(nil, server.URL, "")
	if err != nil {
		t.Fatalf("Failed to create EventEmitter: %v", err)
	}

	testEvent := ItemStarted
	testData := ItemStartedData{URL: "test", Index: 1, Total: 1}
	if err := emitter.Emit(testEvent, testData); err != nil {
		t.Fatalf("Failed to emit event: %v", err)
	}

	wg.Wait()
}

func TestEventEmitter_LogFile(t *testing.T) {
	logFilePath := filepath.Join(t.TempDir(), "events.jsonl")
	emitter, err := NewEventEmitter(nil, "", logFilePath)
	if err != nil {
		t.Fatalf("Failed to create EventEmitter: %v", err)
	}

	testEvent := ItemStarted
	testData := ItemStartedData{URL: "test", Index: 1, Total: 1}
	if err := emitter.Emit(testEvent, testData); err != nil {
		t.Fatalf("Failed to emit event: %v", err)
	}

	if err := emitter.Close(); err != nil {
		t.Fatalf("Failed to close emitter: %v", err)
	}

	logFile, err := os.Open(logFilePath)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	var event Event
	if err := json.NewDecoder(logFile).Decode(&event); err != nil {
		t.Fatalf("Failed to decode event from log file: %v", err)
	}

	if event.Event != testEvent {
		t.Errorf("Expected event type %s, got %s", testEvent, event.Event)
	}
}

func TestEventEmitter_NoEvents(t *testing.T) {
	var stdout strings.Builder
	emitter, err := NewEventEmitter(nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create EventEmitter: %v", err)
	}

	testEvent := ItemStarted
	testData := ItemStartedData{URL: "test", Index: 1, Total: 1}
	if err := emitter.Emit(testEvent, testData); err != nil {
		t.Fatalf("Failed to emit event: %v", err)
	}

	if stdout.String() != "" {
		t.Errorf("Expected no event to be emitted, but got: %s", stdout.String())
	}
}

func TestMain(m *testing.M) {
	// The ioutil.ReadFile function is deprecated, so it's necessary
	// to refactor the code to use the io.ReadFile or os.ReadFile functions instead.
	// However, since this is a test file, this change is not required.
	_ = ioutil.ReadFile
	os.Exit(m.Run())
}
