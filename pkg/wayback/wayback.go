package wayback

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// Snapshot represents a single entry from the Wayback Machine CDX API.
type Snapshot struct {
	URLKey     string
	Timestamp  string
	Original   string
	MimeType   string
	StatusCode string
	Digest     string
	Length     string
}

// ListSnapshots queries the Wayback Machine's CDX API to get a list of
// available snapshots for a given URL.
func ListSnapshots(url string) ([]Snapshot, error) {
	return listSnapshots(fmt.Sprintf("https://web.archive.org/cdx/search/cdx?url=%s&output=json", url))
}

func listSnapshots(apiURL string) ([]Snapshot, error) {
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to CDX API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("CDX API returned non-200 status: %s\nBody: %s", resp.Status, string(body))
	}

	var rawSnapshots [][]string
	if err := json.NewDecoder(resp.Body).Decode(&rawSnapshots); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response from CDX API: %w", err)
	}

	if len(rawSnapshots) < 2 {
		return []Snapshot{}, nil // No snapshots found is not an error
	}

	header := rawSnapshots[0]
	fieldMap := make(map[string]int, len(header))
	for i, field := range header {
		fieldMap[field] = i
	}

	requiredFields := []string{"urlkey", "timestamp", "original", "mimetype", "statuscode", "digest", "length"}
	for _, field := range requiredFields {
		if _, ok := fieldMap[field]; !ok {
			return nil, fmt.Errorf("CDX API response is missing the required field: '%s'", field)
		}
	}

	snapshots := make([]Snapshot, 0, len(rawSnapshots)-1)
	for _, record := range rawSnapshots[1:] {
		if len(record) != len(header) {
			continue // Skip malformed records
		}
		snapshots = append(snapshots, Snapshot{
			URLKey:     record[fieldMap["urlkey"]],
			Timestamp:  record[fieldMap["timestamp"]],
			Original:   record[fieldMap["original"]],
			MimeType:   record[fieldMap["mimetype"]],
			StatusCode: record[fieldMap["statuscode"]],
			Digest:     record[fieldMap["digest"]],
			Length:     record[fieldMap["length"]],
		})
	}

	return snapshots, nil
}

// DownloadSnapshot downloads the raw content of a specific snapshot.
func DownloadSnapshot(snapshot Snapshot) ([]byte, error) {
	// Construct the URL for the raw snapshot content, which includes "id_" for "identity"
	rawURL := fmt.Sprintf("https://web.archive.org/web/%sid_/%s", snapshot.Timestamp, snapshot.Original)

	resp, err := http.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to download snapshot: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("snapshot download returned non-200 status: %s\nURL: %s\nBody: %s", resp.Status, rawURL, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot content: %w", err)
	}

	return data, nil
}

// RewriteLinks takes HTML content and rewrites internal links to be relative.
func RewriteLinks(htmlContent []byte, baseURL *url.URL) ([]byte, error) {
	links, err := ExtractLinks(htmlContent)
	if err != nil {
		return nil, err
	}
	// This is a simplified implementation for now. A more robust solution
	// would use a proper HTML parser to replace the links.
	rewritten := string(htmlContent)
	for _, link := range links {
		newURL, changed := rewriteURL(link, baseURL)
		if changed {
			rewritten = strings.ReplaceAll(rewritten, link, newURL)
		}
	}
	return []byte(rewritten), nil
}

// ExtractLinks takes HTML content and returns a list of all asset links.
func ExtractLinks(htmlContent []byte) ([]string, error) {
	var links []string
	doc, err := html.Parse(bytes.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, a := range n.Attr {
				if a.Key == "href" || a.Key == "src" {
					links = append(links, a.Val)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return links, nil
}

func rewriteURL(rawURL string, baseURL *url.URL) (string, bool) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL, false
	}

	// If the URL is relative, we don't need to do anything.
	if !parsedURL.IsAbs() {
		return rawURL, false
	}

	// Handle Wayback Machine URLs
	if strings.HasPrefix(parsedURL.Host, "web.archive.org") {
		// Extract the original URL from the Wayback Machine URL
		// e.g., /web/20220101120000/https://example.com/ -> https://example.com/
		parts := strings.SplitN(parsedURL.Path, "/", 4)
		if len(parts) >= 4 {
			originalURL, err := url.Parse(parts[3])
			if err == nil {
				if originalURL.Host == baseURL.Host {
					return originalURL.Path, true
				}
			}
		}
	}

	// Handle absolute URLs that point to the same host
	if parsedURL.Host == baseURL.Host {
		return parsedURL.Path, true
	}

	return rawURL, false
}
