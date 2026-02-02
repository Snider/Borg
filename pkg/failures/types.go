package failures

import "time"

// Failure represents a single failure event.
type Failure struct {
	URL       string `json:"url"`
	Error     string `json:"error"`
	Attempts  int    `json:"attempts"`
	Retryable bool   `json:"retryable"`
}

// FailureReport represents a collection of failures for a specific run.
type FailureReport struct {
	Collection string    `json:"collection"`
	Started    time.Time `json:"started"`
	Completed  time.Time `json:"completed"`
	Stats      struct {
		Total   int `json:"total"`
		Success int `json:"success"`
		Failed  int `json:"failed"`
	} `json:"stats"`
	Failures []*Failure `json:"failures"`
}
