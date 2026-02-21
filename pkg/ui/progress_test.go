package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestQuietProgress_Log_Good(t *testing.T) {
	var buf bytes.Buffer
	p := NewQuietProgress(&buf)
	p.Log("info", "test message", "key", "val")
	out := buf.String()
	if !strings.Contains(out, "test message") {
		t.Fatalf("expected log output to contain 'test message', got: %s", out)
	}
}

func TestQuietProgress_StartFinish_Good(t *testing.T) {
	var buf bytes.Buffer
	p := NewQuietProgress(&buf)
	p.Start("collecting")
	p.Update(50, 100)
	p.Finish("done")
	out := buf.String()
	if !strings.Contains(out, "collecting") {
		t.Fatalf("expected 'collecting' in output, got: %s", out)
	}
	if !strings.Contains(out, "done") {
		t.Fatalf("expected 'done' in output, got: %s", out)
	}
}

func TestQuietProgress_Update_Ugly(t *testing.T) {
	var buf bytes.Buffer
	p := NewQuietProgress(&buf)
	// Should not panic with zero total
	p.Update(0, 0)
	p.Update(5, 0)
}
