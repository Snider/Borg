package pwa

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/net/html"
)

// Common fallback paths for PWA manifests
var manifestFallbackPaths = []string{
	"/manifest.json",
	"/manifest.webmanifest",
	"/site.webmanifest",
	"/app.webmanifest",
}

// PWAClient is an interface for interacting with PWAs.
type PWAClient interface {
	FindManifest(pwaURL string) (string, error)
	DownloadAndPackagePWA(pwaURL, manifestURL string, bar *progressbar.ProgressBar) (*datanode.DataNode, error)
}

// NewPWAClient creates a new PWAClient.
func NewPWAClient() PWAClient {
	return &pwaClient{client: http.DefaultClient}
}

type pwaClient struct {
	client *http.Client
}

// FindManifest finds the manifest for a PWA.
// It first looks for a <link rel="manifest"> tag in the HTML,
// then tries common fallback paths if not found.
func (p *pwaClient) FindManifest(pwaURL string) (string, error) {
	resp, err := p.client.Get(pwaURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("failed to fetch PWA page: status code %d", resp.StatusCode)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", err
	}

	var manifestURL string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			var isManifest bool
			var href string
			for _, a := range n.Attr {
				if a.Key == "rel" && a.Val == "manifest" {
					isManifest = true
				}
				if a.Key == "href" {
					href = a.Val
				}
			}
			if isManifest && href != "" {
				manifestURL = href
				return
			}
		}
		for c := n.FirstChild; c != nil && manifestURL == ""; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	// If manifest found via link tag, resolve and return
	if manifestURL != "" {
		resolvedURL, err := p.resolveURL(pwaURL, manifestURL)
		if err != nil {
			return "", err
		}
		return resolvedURL.String(), nil
	}

	// Try fallback paths
	baseURL, err := url.Parse(pwaURL)
	if err != nil {
		return "", err
	}

	for _, path := range manifestFallbackPaths {
		testURL := &url.URL{
			Scheme: baseURL.Scheme,
			Host:   baseURL.Host,
			Path:   path,
		}
		resp, err := p.client.Get(testURL.String())
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return testURL.String(), nil
		}
	}

	return "", fmt.Errorf("manifest not found (checked HTML and fallback paths: %v)", manifestFallbackPaths)
}

// Manifest represents a PWA manifest with all common fields.
type Manifest struct {
	Name            string `json:"name"`
	ShortName       string `json:"short_name"`
	StartURL        string `json:"start_url"`
	Scope           string `json:"scope"`
	Display         string `json:"display"`
	BackgroundColor string `json:"background_color"`
	ThemeColor      string `json:"theme_color"`
	Description     string `json:"description"`
	Icons           []struct {
		Src   string `json:"src"`
		Sizes string `json:"sizes"`
		Type  string `json:"type"`
	} `json:"icons"`
	Screenshots []struct {
		Src   string `json:"src"`
		Sizes string `json:"sizes"`
		Type  string `json:"type"`
	} `json:"screenshots"`
	Shortcuts []struct {
		Name  string `json:"name"`
		URL   string `json:"url"`
		Icons []struct {
			Src string `json:"src"`
		} `json:"icons"`
	} `json:"shortcuts"`
	RelatedApplications []struct {
		Platform string `json:"platform"`
		URL      string `json:"url"`
		ID       string `json:"id"`
	} `json:"related_applications"`
	ServiceWorker struct {
		Src   string `json:"src"`
		Scope string `json:"scope"`
	} `json:"serviceworker"`
}

// DownloadAndPackagePWA downloads and packages a PWA into a DataNode.
// It downloads the manifest, all referenced assets, and parses HTML pages
// for additional linked resources (CSS, JS, images).
func (p *pwaClient) DownloadAndPackagePWA(pwaURL, manifestURL string, bar *progressbar.ProgressBar) (*datanode.DataNode, error) {
	dn := datanode.New()
	var wg sync.WaitGroup
	var errs []error
	var mu sync.Mutex
	downloaded := make(map[string]bool)

	var downloadAndAdd func(assetURL string, parseHTML bool)
	downloadAndAdd = func(assetURL string, parseHTML bool) {
		defer wg.Done()
		if bar != nil {
			bar.Add(1)
		}

		// Skip if already downloaded
		mu.Lock()
		if downloaded[assetURL] {
			mu.Unlock()
			return
		}
		downloaded[assetURL] = true
		mu.Unlock()

		resp, err := p.client.Get(assetURL)
		if err != nil {
			mu.Lock()
			errs = append(errs, fmt.Errorf("failed to download %s: %w", assetURL, err))
			mu.Unlock()
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			mu.Lock()
			errs = append(errs, fmt.Errorf("failed to download %s: status code %d", assetURL, resp.StatusCode))
			mu.Unlock()
			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			mu.Lock()
			errs = append(errs, fmt.Errorf("failed to read body of %s: %w", assetURL, err))
			mu.Unlock()
			return
		}

		u, err := url.Parse(assetURL)
		if err != nil {
			mu.Lock()
			errs = append(errs, fmt.Errorf("failed to parse asset URL %s: %w", assetURL, err))
			mu.Unlock()
			return
		}

		path := strings.TrimPrefix(u.Path, "/")
		if path == "" {
			path = "index.html"
		}
		dn.AddData(path, body)

		// Parse HTML for additional assets
		if parseHTML && isHTMLContent(resp.Header.Get("Content-Type"), body) {
			additionalAssets := p.extractAssetsFromHTML(assetURL, body)
			for _, asset := range additionalAssets {
				mu.Lock()
				if !downloaded[asset] {
					wg.Add(1)
					go downloadAndAdd(asset, false) // Don't recursively parse HTML
				}
				mu.Unlock()
			}
		}
	}

	// Download manifest first, synchronously.
	resp, err := p.client.Get(manifestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to download manifest: status code %d", resp.StatusCode)
	}

	manifestData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest body: %w", err)
	}

	u, _ := url.Parse(manifestURL)
	dn.AddData(strings.TrimPrefix(u.Path, "/"), manifestData)
	downloaded[manifestURL] = true

	// Parse manifest and collect all assets.
	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	assetsToDownload := []string{}
	htmlPages := []string{}

	// Start URL (HTML page)
	if manifest.StartURL != "" {
		startURL, err := p.resolveURL(manifestURL, manifest.StartURL)
		if err == nil {
			htmlPages = append(htmlPages, startURL.String())
		}
	} else {
		// If no start_url, use the PWA URL itself
		htmlPages = append(htmlPages, pwaURL)
	}

	// Icons
	for _, icon := range manifest.Icons {
		if icon.Src != "" {
			iconURL, err := p.resolveURL(manifestURL, icon.Src)
			if err == nil {
				assetsToDownload = append(assetsToDownload, iconURL.String())
			}
		}
	}

	// Screenshots
	for _, screenshot := range manifest.Screenshots {
		if screenshot.Src != "" {
			screenshotURL, err := p.resolveURL(manifestURL, screenshot.Src)
			if err == nil {
				assetsToDownload = append(assetsToDownload, screenshotURL.String())
			}
		}
	}

	// Shortcuts and their icons
	for _, shortcut := range manifest.Shortcuts {
		if shortcut.URL != "" {
			shortcutURL, err := p.resolveURL(manifestURL, shortcut.URL)
			if err == nil {
				htmlPages = append(htmlPages, shortcutURL.String())
			}
		}
		for _, icon := range shortcut.Icons {
			if icon.Src != "" {
				iconURL, err := p.resolveURL(manifestURL, icon.Src)
				if err == nil {
					assetsToDownload = append(assetsToDownload, iconURL.String())
				}
			}
		}
	}

	// Service worker
	if manifest.ServiceWorker.Src != "" {
		swURL, err := p.resolveURL(manifestURL, manifest.ServiceWorker.Src)
		if err == nil {
			assetsToDownload = append(assetsToDownload, swURL.String())
		}
	}

	// Download HTML pages first (with asset extraction)
	for _, page := range htmlPages {
		wg.Add(1)
		go downloadAndAdd(page, true)
	}
	wg.Wait()

	// Download remaining assets
	for _, asset := range assetsToDownload {
		if !downloaded[asset] {
			wg.Add(1)
			go downloadAndAdd(asset, false)
		}
	}
	wg.Wait()

	// Try to detect service worker from HTML if not in manifest
	if manifest.ServiceWorker.Src == "" {
		swURL := p.detectServiceWorker(pwaURL, dn)
		if swURL != "" && !downloaded[swURL] {
			wg.Add(1)
			go downloadAndAdd(swURL, false)
			wg.Wait()
		}
	}

	if len(errs) > 0 {
		var errStrings []string
		for _, e := range errs {
			errStrings = append(errStrings, e.Error())
		}
		return dn, fmt.Errorf("%s", strings.Join(errStrings, "; "))
	}

	return dn, nil
}

// extractAssetsFromHTML parses HTML and extracts linked assets.
func (p *pwaClient) extractAssetsFromHTML(baseURL string, htmlContent []byte) []string {
	var assets []string
	doc, err := html.Parse(strings.NewReader(string(htmlContent)))
	if err != nil {
		return assets
	}

	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode {
			var href string
			switch n.Data {
			case "link":
				// CSS stylesheets and icons
				var rel, linkHref string
				for _, a := range n.Attr {
					if a.Key == "rel" {
						rel = a.Val
					}
					if a.Key == "href" {
						linkHref = a.Val
					}
				}
				if linkHref != "" && (rel == "stylesheet" || rel == "icon" || rel == "apple-touch-icon" || rel == "shortcut icon") {
					href = linkHref
				}
			case "script":
				// JavaScript files
				for _, a := range n.Attr {
					if a.Key == "src" && a.Val != "" {
						href = a.Val
						break
					}
				}
			case "img":
				// Images
				for _, a := range n.Attr {
					if a.Key == "src" && a.Val != "" {
						href = a.Val
						break
					}
				}
			}

			if href != "" && !strings.HasPrefix(href, "data:") {
				resolved, err := p.resolveURL(baseURL, href)
				if err == nil {
					assets = append(assets, resolved.String())
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(doc)

	return assets
}

// detectServiceWorker tries to find service worker registration in HTML/JS.
func (p *pwaClient) detectServiceWorker(baseURL string, dn *datanode.DataNode) string {
	// Look for common service worker registration patterns
	patterns := []string{
		`navigator\.serviceWorker\.register\(['"]([^'"]+)['"]`,
		`serviceWorker\.register\(['"]([^'"]+)['"]`,
	}

	// Check all downloaded HTML and JS files
	err := dn.Walk(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".html") || strings.HasSuffix(path, ".js") || path == "index.html" {
			file, err := dn.Open(path)
			if err != nil {
				return nil
			}
			defer file.Close()
			content, err := io.ReadAll(file)
			if err != nil {
				return nil
			}

			for _, pattern := range patterns {
				re := regexp.MustCompile(pattern)
				matches := re.FindSubmatch(content)
				if len(matches) > 1 {
					swPath := string(matches[1])
					resolved, err := p.resolveURL(baseURL, swPath)
					if err == nil {
						return fmt.Errorf("found:%s", resolved.String())
					}
				}
			}
		}
		return nil
	})

	if err != nil && strings.HasPrefix(err.Error(), "found:") {
		return strings.TrimPrefix(err.Error(), "found:")
	}

	return ""
}

// isHTMLContent checks if content is HTML based on Content-Type or content inspection.
func isHTMLContent(contentType string, body []byte) bool {
	if strings.Contains(contentType, "text/html") {
		return true
	}
	// Check for HTML doctype or html tag
	content := strings.ToLower(string(body[:min(len(body), 1024)]))
	return strings.Contains(content, "<!doctype html") || strings.Contains(content, "<html")
}

func (p *pwaClient) resolveURL(base, ref string) (*url.URL, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return nil, err
	}
	return baseURL.ResolveReference(refURL), nil
}

// MockPWAClient is a mock implementation of the PWAClient interface.
type MockPWAClient struct {
	ManifestURL string
	DN          *datanode.DataNode
	Err         error
}

// NewMockPWAClient creates a new MockPWAClient.
func NewMockPWAClient(manifestURL string, dn *datanode.DataNode, err error) PWAClient {
	return &MockPWAClient{
		ManifestURL: manifestURL,
		DN:          dn,
		Err:         err,
	}
}

// FindManifest mocks the finding of a PWA manifest.
func (m *MockPWAClient) FindManifest(pwaURL string) (string, error) {
	return m.ManifestURL, m.Err
}

// DownloadAndPackagePWA mocks the downloading and packaging of a PWA.
func (m *MockPWAClient) DownloadAndPackagePWA(pwaURL, manifestURL string, bar *progressbar.ProgressBar) (*datanode.DataNode, error) {
	return m.DN, m.Err
}
