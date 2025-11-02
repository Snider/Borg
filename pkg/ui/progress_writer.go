package ui

import "github.com/schollz/progressbar/v3"

// ProgressWriter updates a progress bar’s description on writes.
type ProgressWriter struct {
	bar *progressbar.ProgressBar
}

// NewProgressWriter creates a writer that sets the progress bar description to the last written line.
func NewProgressWriter(bar *progressbar.ProgressBar) *ProgressWriter {
	return &ProgressWriter{bar: bar}
}

// Write implements io.Writer by describing the progress with the provided bytes.
func (pw *ProgressWriter) Write(p []byte) (n int, err error) {
	if pw == nil || pw.bar == nil {
		return len(p), nil
	}
	s := string(p)
	pw.bar.Describe(s)
	return len(p), nil
}
