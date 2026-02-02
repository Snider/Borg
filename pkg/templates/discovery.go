package templates

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed builtin/*.yaml
var EmbeddedTemplates embed.FS

// GetUserTemplateDir returns the path to the user's template directory.
func GetUserTemplateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get user home directory: %w", err)
	}
	return filepath.Join(home, ".borg", "templates"), nil
}

// ListUserTemplates lists all templates in the given directory.
func ListUserTemplates(dir string) ([]string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return []string{}, nil
	}

	var templates []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml") {
			// Return just the filename without extension
			templates = append(templates, strings.TrimSuffix(info.Name(), filepath.Ext(info.Name())))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not walk template directory: %w", err)
	}
	return templates, nil
}

// ListBuiltinTemplates lists all built-in templates.
func ListBuiltinTemplates() ([]string, error) {
	var templates []string
	entries, err := EmbeddedTemplates.ReadDir("builtin")
	if err != nil {
		return nil, fmt.Errorf("could not read embedded templates: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && (filepath.Ext(entry.Name()) == ".yaml" || filepath.Ext(entry.Name()) == ".yml") {
			templates = append(templates, strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
		}
	}
	return templates, nil
}
