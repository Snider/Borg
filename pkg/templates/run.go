package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// FindTemplate finds a template by name. It first searches the user's template
// directory, then falls back to the built-in templates.
func FindTemplate(name string) (string, []byte, error) {
	userTemplateDir, err := GetUserTemplateDir()
	if err != nil {
		return "", nil, err
	}

	// Check user templates first
	for _, ext := range []string{".yaml", ".yml"} {
		templatePath := filepath.Join(userTemplateDir, name+ext)
		if _, err := os.Stat(templatePath); err == nil {
			data, err := os.ReadFile(templatePath)
			if err != nil {
				return "", nil, fmt.Errorf("could not read user template: %w", err)
			}
			return templatePath, data, nil
		}
	}

	// Check built-in templates
	for _, ext := range []string{".yaml", ".yml"} {
		templatePath := name + ext
		data, err := EmbeddedTemplates.ReadFile(filepath.Join("builtin", templatePath))
		if err == nil {
			return "builtin:" + templatePath, data, nil
		}
	}

	return "", nil, fmt.Errorf("template '%s' not found", name)
}

// LoadTemplate loads and parses a template from a byte slice.
func LoadTemplate(data []byte) (*Template, error) {
	var tmpl Template
	err := yaml.Unmarshal(data, &tmpl)
	if err != nil {
		return nil, fmt.Errorf("could not parse template file: %w", err)
	}

	return &tmpl, nil
}

// Substitute replaces variables in a string.
func Substitute(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}
	return s
}
