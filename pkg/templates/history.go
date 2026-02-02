package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetHistoryFile returns the path to the user's history file.
func GetHistoryFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get user home directory: %w", err)
	}
	borgDir := filepath.Join(home, ".borg")
	if _, err := os.Stat(borgDir); os.IsNotExist(err) {
		if err := os.MkdirAll(borgDir, 0755); err != nil {
			return "", fmt.Errorf("could not create .borg directory: %w", err)
		}
	}
	return filepath.Join(borgDir, "history"), nil
}

// AppendToHistory appends a command to the history file.
func AppendToHistory(command string) error {
	historyFile, err := GetHistoryFile()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(historyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("could not open history file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(command + "\n"); err != nil {
		return fmt.Errorf("could not write to history file: %w", err)
	}

	return nil
}

// ReadLastHistoryEntry reads the last command from the history file.
func ReadLastHistoryEntry() (string, error) {
	historyFile, err := GetHistoryFile()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(historyFile)
	if err != nil {
		return "", fmt.Errorf("could not read history file: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("history is empty")
	}

	return lines[len(lines)-1], nil
}
