package progress

import (
	"encoding/json"
	"os"
	"time"
)

// Progress holds the state of a collection operation.
type Progress struct {
	Source    string    `json:"source"`
	StartedAt time.Time `json:"started_at"`
	Completed []string  `json:"completed"`
	Pending   []string  `json:"pending"`
	Failed    []string  `json:"failed"`
}

// Load reads a progress file from the given path.
func Load(path string) (*Progress, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var p Progress
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// Save writes the progress to the given path.
func (p *Progress) Save(path string) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
