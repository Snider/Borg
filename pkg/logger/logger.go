// Package logger provides a simple configurable logger for the application.
package logger

import (
	"log/slog"
	"os"
)

// New creates a new slog.Logger. If verbose is true, the logger will be
// configured to show debug messages. Otherwise, it will only show info
// level and above.
//
// Example:
//
//	// Create a standard logger
//	log := logger.New(false)
//	log.Info("This is an info message")
//
//	// Create a verbose logger
//	verboseLog := logger.New(true)
//	verboseLog.Debug("This is a debug message")
func New(verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
}
