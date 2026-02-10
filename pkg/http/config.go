package http

import (
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

// Config represents the rate limiting configuration.
type Config struct {
	Defaults Rate            `yaml:"defaults"`
	Domains  map[string]Rate `yaml:"domains"`
}

// Rate represents a rate limit.
type Rate struct {
	RequestsPerSecond float64 `yaml:"requests_per_second"`
	Burst             int     `yaml:"burst"`
	Reason            string  `yaml:"reason,omitempty"`
}

// ParseConfig parses a configuration file.
func ParseConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// GetRate returns the rate limit for a given domain.
func (c *Config) GetRate(domain string) Rate {
	// Check for an exact match first.
	if rate, ok := c.Domains[domain]; ok {
		return rate
	}

	// Check for a wildcard match.
	parts := strings.Split(domain, ".")
	for i := 1; i < len(parts); i++ {
		wildcard := "*." + strings.Join(parts[i:], ".")
		if rate, ok := c.Domains[wildcard]; ok {
			return rate
		}
	}

	// Return the default rate.
	return c.Defaults
}
