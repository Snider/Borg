
package ui

import "github.com/schollz/progressbar/v3"

// progressWriter is an io.Writer that updates a progress bar's description
// with the written data. This is useful for showing the progress of operations
// that produce streaming text output, like git cloning.
type progressWriter struct {
	bar *progressbar.ProgressBar
}

// NewProgressWriter creates a new progressWriter that wraps the given
// progress bar.
func NewProgressWriter(bar *progressbar.ProgressBar) *progressWriter {
	return &progressWriter{bar: bar}
}

// Write implements the io.Writer interface. It updates the progress bar's
// description with the contents of the byte slice.
func (pw *progressWriter) Write(p []byte) (n int, err error) {
	if pw == nil || pw.bar == nil {
		return len(p), nil
	}
	s := string(p)
	pw.bar.Describe(s)
	return len(p), nil
}
