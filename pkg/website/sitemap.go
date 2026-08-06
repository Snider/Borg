package website

import (
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// SitemapURL represents a single URL entry in a sitemap.
type SitemapURL struct {
	Loc string `xml:"loc"`
}

// URLSet represents a standard sitemap.
type URLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	URLs    []SitemapURL `xml:"url"`
}

// SitemapIndex represents a sitemap index file.
type SitemapIndex struct {
	XMLName  xml.Name     `xml:"sitemapindex"`
	Sitemaps []SitemapURL `xml:"sitemap"`
}

// FetchAndParseSitemap fetches and parses a sitemap from the given URL.
// It handles standard sitemaps, sitemap indexes, and gzipped sitemaps.
func FetchAndParseSitemap(sitemapURL string, client *http.Client) ([]string, error) {
	resp, err := client.Get(sitemapURL)
	if err != nil {
		return nil, fmt.Errorf("error fetching sitemap: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status for sitemap: %s", resp.Status)
	}

	var reader io.Reader = resp.Body
	if strings.HasSuffix(sitemapURL, ".gz") {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error creating gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("error reading sitemap body: %w", err)
	}

	// Try parsing as a sitemap index first
	var sitemapIndex SitemapIndex
	if err := xml.Unmarshal(body, &sitemapIndex); err == nil && len(sitemapIndex.Sitemaps) > 0 {
		var allURLs []string
		for _, sitemap := range sitemapIndex.Sitemaps {
			urls, err := FetchAndParseSitemap(sitemap.Loc, client)
			if err != nil {
				// In a real-world scenario, you might want to handle this more gracefully
				return nil, fmt.Errorf("error parsing sitemap from index '%s': %w", sitemap.Loc, err)
			}
			allURLs = append(allURLs, urls...)
		}
		return allURLs, nil
	}

	// If not a sitemap index, try parsing as a standard sitemap
	var urlSet URLSet
	if err := xml.Unmarshal(body, &urlSet); err == nil {
		var urls []string
		for _, u := range urlSet.URLs {
			urls = append(urls, u.Loc)
		}
		return urls, nil
	}

	return nil, fmt.Errorf("failed to parse sitemap XML from %s", sitemapURL)
}

// DiscoverSitemap attempts to discover the sitemap URL for a given base URL.
func DiscoverSitemap(baseURL string, client *http.Client) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	// Make sure we're at the root of the domain
	u.Path = ""
	u.RawQuery = ""
	u.Fragment = ""


	sitemapPaths := []string{"sitemap.xml", "sitemap_index.xml", "sitemap.xml.gz", "sitemap_index.xml.gz"}

	for _, path := range sitemapPaths {
		sitemapURL := u.String() + "/" + path
		resp, err := client.Head(sitemapURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			// Ensure we close the body, even for a HEAD request
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			return sitemapURL, nil
		}
		if resp != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}

	return "", fmt.Errorf("sitemap not found for %s", baseURL)
}
