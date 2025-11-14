package ui

import (
	"github.com/schollz/progressbar/v3"
)

// NewProgressBar creates and returns a new progress bar with a standard
// set of options suitable for the application.
//
// Example:
//
//	bar := ui.NewProgressBar(100, "Downloading files")
//	for i := 0; i < 100; i++ {
//		bar.Add(1)
//		time.Sleep(10 * time.Millisecond)
//	}
func NewProgressBar(total int, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions(total,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWidth(15),
		progressbar.OptionShowCount(),
		progressbar.OptionClearOnFinish(),
	)
}
