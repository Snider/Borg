package ui

import (
	"fmt"
	"io"
	"os"

	"github.com/mattn/go-isatty"
)

// Progress abstracts output for both interactive and scripted use.
type Progress interface {
	Start(label string)
	Update(current, total int64)
	Finish(label string)
	Log(level, msg string, args ...any)
}

// QuietProgress writes structured log lines. For cron, pipes, --quiet.
type QuietProgress struct {
	w io.Writer
}

func NewQuietProgress(w io.Writer) *QuietProgress {
	return &QuietProgress{w: w}
}

func (q *QuietProgress) Start(label string) {
	fmt.Fprintf(q.w, "[START] %s\n", label)
}

func (q *QuietProgress) Update(current, total int64) {
	if total > 0 {
		fmt.Fprintf(q.w, "[PROGRESS] %d/%d\n", current, total)
	}
}

func (q *QuietProgress) Finish(label string) {
	fmt.Fprintf(q.w, "[DONE] %s\n", label)
}

func (q *QuietProgress) Log(level, msg string, args ...any) {
	fmt.Fprintf(q.w, "[%s] %s", level, msg)
	for i := 0; i+1 < len(args); i += 2 {
		fmt.Fprintf(q.w, " %v=%v", args[i], args[i+1])
	}
	fmt.Fprintln(q.w)
}

// InteractiveProgress uses simple terminal output for TTY sessions.
type InteractiveProgress struct {
	w io.Writer
}

func NewInteractiveProgress(w io.Writer) *InteractiveProgress {
	return &InteractiveProgress{w: w}
}

func (p *InteractiveProgress) Start(label string) {
	fmt.Fprintf(p.w, "→ %s\n", label)
}

func (p *InteractiveProgress) Update(current, total int64) {
	if total > 0 {
		pct := current * 100 / total
		fmt.Fprintf(p.w, "\r  %d%%", pct)
	}
}

func (p *InteractiveProgress) Finish(label string) {
	fmt.Fprintf(p.w, "\r✓ %s\n", label)
}

func (p *InteractiveProgress) Log(level, msg string, args ...any) {
	fmt.Fprintf(p.w, "  %s", msg)
	for i := 0; i+1 < len(args); i += 2 {
		fmt.Fprintf(p.w, " %v=%v", args[i], args[i+1])
	}
	fmt.Fprintln(p.w)
}

// IsTTY returns true if the given file descriptor is a terminal.
func IsTTY(fd uintptr) bool {
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

// DefaultProgress returns InteractiveProgress for TTYs, QuietProgress otherwise.
func DefaultProgress() Progress {
	if IsTTY(os.Stdout.Fd()) {
		return NewInteractiveProgress(os.Stdout)
	}
	return NewQuietProgress(os.Stdout)
}
