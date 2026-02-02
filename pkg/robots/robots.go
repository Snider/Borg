package robots

import (
	"path"
	"strconv"
	"strings"
	"time"
)

// RobotsData holds the parsed robots.txt data for a specific user-agent.
type RobotsData struct {
	Disallow   []string
	CrawlDelay time.Duration
}

// IsAllowed checks if a given path is allowed by the robots.txt rules.
func (r *RobotsData) IsAllowed(p string) bool {
	// A more complete implementation would handle wildcards.
	// This is a simple path prefix match.
	for _, rule := range r.Disallow {
		if rule == "" {
			// An empty Disallow rule means nothing is disallowed by this rule.
			continue
		}
		if rule == "/" {
			// Disallow: / means disallow everything.
			return false
		}
		if strings.HasPrefix(p, rule) {
			return false
		}
	}
	return true
}

// Parse parses the content of a robots.txt file for a specific user-agent.
func Parse(content []byte, userAgent string) (*RobotsData, error) {
	lines := strings.Split(string(content), "\n")

	rules := make(map[string]*RobotsData)
	var currentUAs []string
	lastWasUA := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, "#"); idx != -1 {
			line = line[:idx]
		}
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch key {
		case "user-agent":
			if !lastWasUA {
				currentUAs = []string{} // New group
			}
			currentUAs = append(currentUAs, strings.ToLower(value))
			lastWasUA = true
		case "disallow", "crawl-delay":
			if len(currentUAs) == 0 {
				continue // Rule without a user-agent
			}

			for _, ua := range currentUAs {
				if rules[ua] == nil {
					rules[ua] = &RobotsData{}
				}
				if key == "disallow" {
					rules[ua].Disallow = append(rules[ua].Disallow, path.Clean("/"+value))
				} else if key == "crawl-delay" {
					if delay, err := strconv.ParseFloat(value, 64); err == nil {
						rules[ua].CrawlDelay = time.Duration(delay * float64(time.Second))
					}
				}
			}
			lastWasUA = false
		default:
			lastWasUA = false
		}
	}

	lowerUserAgent := strings.ToLower(userAgent)

	// Look for most specific match.
	bestMatch := ""
	for ua := range rules {
		if strings.Contains(lowerUserAgent, ua) {
			if len(ua) > len(bestMatch) {
				bestMatch = ua
			}
		}
	}
	if bestMatch != "" {
		return rules[bestMatch], nil
	}

	// Fallback to wildcard.
	if data, ok := rules["*"]; ok {
		return data, nil
	}

	return &RobotsData{}, nil
}
