package logger

import (
	"log/slog"
	"os"
)

// New returns a configured slog.Logger.
// When verbose is true, the logger emits debug-level logs; otherwise info-level.
func New(verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
}
