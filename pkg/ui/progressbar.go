package ui

import (
	"github.com/schollz/progressbar/v3"
)

// NewProgressBar creates a new progress bar with the specified total and description.
func NewProgressBar(total int, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions(total,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWidth(15),
		progressbar.OptionShowCount(),
		progressbar.OptionClearOnFinish(),
	)
}
