package robots

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		userAgent   string
		expected    *RobotsData
		expectedErr bool
	}{
		{
			name: "Specific user agent",
			content: `
				User-agent: BorgBot
				Disallow: /private/
				Crawl-delay: 2
			`,
			userAgent: "BorgBot/1.0",
			expected: &RobotsData{
				Disallow:   []string{"/private"},
				CrawlDelay: 2 * time.Second,
			},
		},
		{
			name: "Wildcard user agent",
			content: `
				User-agent: *
				Disallow: /admin/
			`,
			userAgent: "AnotherBot",
			expected: &RobotsData{
				Disallow: []string{"/admin"},
			},
		},
		{
			name: "Multiple disallow rules",
			content: `
				User-agent: *
				Disallow: /admin/
				Disallow: /login
			`,
			userAgent: "AnyBot",
			expected: &RobotsData{
				Disallow: []string{"/admin", "/login"},
			},
		},
		{
			name: "No rules for user agent",
			content: `
				User-agent: GoogleBot
				Disallow: /
			`,
			userAgent: "MyBot",
			expected:  &RobotsData{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			robotsData, err := Parse([]byte(tc.content), tc.userAgent)
			if (err != nil) != tc.expectedErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tc.expectedErr)
			}

			if len(robotsData.Disallow) != len(tc.expected.Disallow) {
				t.Fatalf("expected %d disallow rules, got %d", len(tc.expected.Disallow), len(robotsData.Disallow))
			}

			for i, rule := range tc.expected.Disallow {
				if robotsData.Disallow[i] != rule {
					t.Errorf("expected disallow rule %s, got %s", rule, robotsData.Disallow[i])
				}
			}

			if robotsData.CrawlDelay != tc.expected.CrawlDelay {
				t.Errorf("expected crawl delay %v, got %v", tc.expected.CrawlDelay, robotsData.CrawlDelay)
			}
		})
	}
}

func TestIsAllowed(t *testing.T) {
	testCases := []struct {
		name       string
		robotsData *RobotsData
		path       string
		allowed    bool
	}{
		{
			name: "Path is disallowed",
			robotsData: &RobotsData{
				Disallow: []string{"/private"},
			},
			path:    "/private/page.html",
			allowed: false,
		},
		{
			name: "Path is allowed",
			robotsData: &RobotsData{
				Disallow: []string{"/private"},
			},
			path:    "/public/page.html",
			allowed: true,
		},
		{
			name:       "No rules",
			robotsData: &RobotsData{},
			path:       "/any/page.html",
			allowed:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if allowed := tc.robotsData.IsAllowed(tc.path); allowed != tc.allowed {
				t.Errorf("IsAllowed(%s) = %v, want %v", tc.path, allowed, tc.allowed)
			}
		})
	}
}
