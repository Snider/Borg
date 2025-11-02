
package ui

import "github.com/schollz/progressbar/v3"

type progressWriter struct {
	bar *progressbar.ProgressBar
}

func NewProgressWriter(bar *progressbar.ProgressBar) *progressWriter {
	return &progressWriter{bar: bar}
}

func (pw *progressWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	pw.bar.Describe(s)
	return len(p), nil
}
