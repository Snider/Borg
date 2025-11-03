package logger

import (
	"bytes"
	"context"
	"io"
	"os"
	"log/slog"
	"testing"
)

func TestNew(t *testing.T) {
	// Test non-verbose logger
	var buf bytes.Buffer
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	log := New(false)
	log.Info("info message")
	log.Debug("debug message")

	w.Close()
	os.Stderr = oldStderr
	io.Copy(&buf, r)

	if !bytes.Contains(buf.Bytes(), []byte("info message")) {
		t.Errorf("expected info message to be logged")
	}
	if bytes.Contains(buf.Bytes(), []byte("debug message")) {
		t.Errorf("expected debug message not to be logged")
	}

	// Test verbose logger
	buf.Reset()
	r, w, _ = os.Pipe()
	os.Stderr = w

	log = New(true)
	log.Info("info message")
	log.Debug("debug message")

	w.Close()
	os.Stderr = oldStderr
	io.Copy(&buf, r)

	if !bytes.Contains(buf.Bytes(), []byte("info message")) {
		t.Errorf("expected info message to be logged")
	}
	if !bytes.Contains(buf.Bytes(), []byte("debug message")) {
		t.Errorf("expected debug message to be logged")
	}
}
func TestNew_Level(t *testing.T) {
	log := New(true)
	if !log.Enabled(context.Background(), slog.LevelDebug) {
		t.Errorf("expected debug level to be enabled")
	}

	log = New(false)
	if log.Enabled(context.Background(), slog.LevelDebug) {
		t.Errorf("expected debug level to be disabled")
	}
}
