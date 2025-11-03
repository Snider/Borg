package ui

import "github.com/schollz/progressbar/v3"

// progressWriter implements io.Writer to update a progress bar with textual status.
type progressWriter struct {
	bar *progressbar.ProgressBar
}

// NewProgressWriter returns a writer that updates the provided progress bar's description.
func NewProgressWriter(bar *progressbar.ProgressBar) *progressWriter {
	return &progressWriter{bar: bar}
}

// Write updates the progress bar description with the provided bytes as a string.
// It returns the length of p to satisfy io.Writer semantics.
func (pw *progressWriter) Write(p []byte) (n int, err error) {
	if pw == nil || pw.bar == nil {
		return len(p), nil
	}
	s := string(p)
	pw.bar.Describe(s)
	return len(p), nil
}
