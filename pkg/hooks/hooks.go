package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// HookEventType represents the type of a hook event.
type HookEventType string

const (
	// OnFileCollected is triggered after each file is collected.
	OnFileCollected HookEventType = "on_file_collected"
	// OnURLFound is triggered when a new URL is discovered.
	OnURLFound HookEventType = "on_url_found"
	// OnCollectionComplete is triggered after the entire collection is done.
	OnCollectionComplete HookEventType = "on_collection_complete"
	// OnError is triggered when a failure occurs.
	OnError HookEventType = "on_error"
)

// Hook represents a single hook to be executed.
type Hook struct {
	Pattern string `yaml:"pattern"`
	Run     string `yaml:"run"`
}

// HookConfig represents the configuration for all hooks.
type HookConfig struct {
	Hooks map[HookEventType][]Hook `yaml:"hooks"`
}

// HookRunner is responsible for running hooks.
type HookRunner struct {
	config *HookConfig
}

// NewHookRunner creates a new HookRunner.
func NewHookRunner(configFile string) (*HookRunner, error) {
	if configFile == "" {
		return &HookRunner{config: &HookConfig{}}, nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read hook config file: %w", err)
	}

	var config HookConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal hook config: %w", err)
	}

	return &HookRunner{config: &config}, nil
}

// Event represents a hook event.
type Event struct {
	Event HookEventType `json:"event"`
	File  string        `json:"file,omitempty"`
	URL   string        `json:"url,omitempty"`
	Type  string        `json:"type,omitempty"`
	Error string        `json:"error,omitempty"`
}

// Trigger triggers the hooks for a given event.
func (r *HookRunner) Trigger(event Event) error {
	if r.config == nil {
		return nil
	}

	hooks, ok := r.config.Hooks[event.Event]
	if !ok {
		return nil
	}

	for _, hook := range hooks {
		if hook.Pattern != "" && event.File != "" {
			matched, err := filepath.Match(hook.Pattern, filepath.Base(event.File))
			if err != nil {
				return fmt.Errorf("failed to match pattern '%s' with file '%s': %w", hook.Pattern, event.File, err)
			}
			if !matched {
				continue
			}
		}

		err := r.runHook(hook, event)
		if err != nil {
			return fmt.Errorf("failed to run hook '%s': %w", hook.Run, err)
		}
	}

	return nil
}

func (r *HookRunner) runHook(hook Hook, event Event) error {
	cmd := exec.Command("sh", "-c", hook.Run)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	go func() {
		defer stdin.Close()
		err := json.NewEncoder(stdin).Encode(event)
		if err != nil {
			// It's hard to propagate this error, so we'll just log it
			fmt.Fprintf(os.Stderr, "failed to write to hook stdin: %v\n", err)
		}
	}()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hook execution failed: %w\n%s", err, string(output))
	}

	return nil
}
