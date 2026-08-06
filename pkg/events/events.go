package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// EventType is the type of an event.
type EventType string

const (
	// CollectionStarted is emitted when a collection starts.
	CollectionStarted EventType = "collection_started"
	// ItemStarted is emitted when an item starts being processed.
	ItemStarted EventType = "item_started"
	// ItemCompleted is emitted when an item is successfully processed.
	ItemCompleted EventType = "item_completed"
	// ItemFailed is emitted when an item fails to be processed.
	ItemFailed EventType = "item_failed"
	// CollectionCompleted is emitted when a collection completes.
	CollectionCompleted EventType = "collection_completed"
)

// Event is the structure of an event.
type Event struct {
	Event     EventType   `json:"event"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// CollectionStartedData is the data for a collection_started event.
type CollectionStartedData struct {
	Source         string `json:"source"`
	EstimatedItems int    `json:"estimated_items"`
}

// ItemStartedData is the data for an item_started event.
type ItemStartedData struct {
	URL   string `json:"url"`
	Index int    `json:"index"`
	Total int    `json:"total"`
}

// ItemCompletedData is the data for an item_completed event.
type ItemCompletedData struct {
	URL        string `json:"url"`
	Size       int64  `json:"size"`
	DurationMs int64  `json:"duration_ms"`
	Index      int    `json:"index"`
	Total      int    `json:"total"`
}

// ItemFailedData is the data for an item_failed event.
type ItemFailedData struct {
	URL   string `json:"url"`
	Error string `json:"error"`
}

// CollectionCompletedData is the data for a collection_completed event.
type CollectionCompletedData struct {
	Stats      map[string]interface{} `json:"stats"`
	DurationMs int64                  `json:"duration_ms"`
}

// EventEmitter is responsible for emitting events.
type EventEmitter struct {
	stdout   io.Writer
	webhook  string
	logFile  *os.File
	noEvents bool
}

// NewEventEmitter creates a new EventEmitter.
func NewEventEmitter(stdout io.Writer, webhook, logFilePath string) (*EventEmitter, error) {
	if stdout == nil && webhook == "" && logFilePath == "" {
		return &EventEmitter{noEvents: true}, nil
	}

	var logFile *os.File
	if logFilePath != "" {
		var err error
		logFile, err = os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open event log file: %w", err)
		}
	}

	return &EventEmitter{
		stdout:  stdout,
		webhook: webhook,
		logFile: logFile,
	}, nil
}

// Emit emits an event.
func (e *EventEmitter) Emit(eventType EventType, data interface{}) error {
	if e.noEvents {
		return nil
	}
	event := Event{
		Event:     eventType,
		Timestamp: time.Now().UTC(),
		Data:      data,
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if e.stdout != nil {
		fmt.Fprintln(e.stdout, string(jsonData))
	}

	if e.webhook != "" {
		go e.sendWebhook(jsonData)
	}

	if e.logFile != nil {
		if _, err := e.logFile.Write(append(jsonData, '\n')); err != nil {
			// Cannot write to log file, maybe log to stderr?
			fmt.Fprintf(os.Stderr, "failed to write event to log file: %v\n", err)
		}
	}

	return nil
}

// Close closes the event emitter.
func (e *EventEmitter) Close() error {
	if e.logFile != nil {
		return e.logFile.Close()
	}
	return nil
}

func (e *EventEmitter) sendWebhook(jsonData []byte) {
	resp, err := http.Post(e.webhook, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to send webhook: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "webhook returned non-2xx status code: %d %s\n", resp.StatusCode, string(body))
	}
}
