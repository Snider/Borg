package logger

import (
	"log/slog"
	"os"
)

// New constructs a slog.Logger configured for stderr with the given verbosity.
// When verbose is true, the logger emits debug-level messages; otherwise info-level.
func New(verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
}
