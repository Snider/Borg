package website

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/robots"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/net/html"
	"time"
)

var DownloadAndPackageWebsite = downloadAndPackageWebsite

// Downloader is a recursive website downloader.
type Downloader struct {
	baseURL     *url.URL
	dn          *datanode.DataNode
	visited     map[string]bool
	maxDepth    int
	progressBar *progressbar.ProgressBar
	client      *http.Client
	errors      []error
	robots      *robots.RobotsData
	userAgent   string
	minDelay    time.Duration
}

// NewDownloader creates a new Downloader.
func NewDownloader(maxDepth int) *Downloader {
	return NewDownloaderWithClient(maxDepth, http.DefaultClient)
}

// NewDownloaderWithClient creates a new Downloader with a custom http.Client.
func NewDownloaderWithClient(maxDepth int, client *http.Client) *Downloader {
	return &Downloader{
		dn:          datanode.New(),
		visited:     make(map[string]bool),
		maxDepth:    maxDepth,
		client:      client,
		errors:      make([]error, 0),
		userAgent:   "Borg/1.0",
	}
}

// downloadAndPackageWebsite downloads a website and packages it into a DataNode.
func downloadAndPackageWebsite(startURL string, maxDepth int, bar *progressbar.ProgressBar, userAgent string, ignoreRobots bool, minDelay time.Duration) (*datanode.DataNode, error) {
	baseURL, err := url.Parse(startURL)
	if err != nil {
		return nil, err
	}

	d := NewDownloader(maxDepth)
	d.baseURL = baseURL
	d.progressBar = bar
	d.userAgent = userAgent
	d.minDelay = minDelay

	if !ignoreRobots {
		robotsURL, err := baseURL.Parse("/robots.txt")
		if err == nil {
			resp, err := d.client.Get(robotsURL.String())
			if err == nil && resp.StatusCode == http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err == nil {
					d.robots, _ = robots.Parse(body, d.userAgent)
				}
			}
		}
	}

	d.crawl(startURL, 0)

	if len(d.errors) > 0 {
		var errs []string
		for _, e := range d.errors {
			errs = append(errs, e.Error())
		}
		return nil, fmt.Errorf("failed to download website:\n%s", strings.Join(errs, "\n"))
	}

	return d.dn, nil
}

func (d *Downloader) crawl(pageURL string, depth int) {
	if depth > d.maxDepth || d.visited[pageURL] {
		return
	}

	u, err := url.Parse(pageURL)
	if err != nil {
		d.errors = append(d.errors, fmt.Errorf("invalid URL %s: %w", pageURL, err))
		return
	}

	if d.robots != nil && !d.robots.IsAllowed(u.Path) {
		return
	}

	d.visited[pageURL] = true
	if d.progressBar != nil {
		d.progressBar.Add(1)
	}

	d.delay()

	req, _ := http.NewRequest("GET", pageURL, nil)
	req.Header.Set("User-Agent", d.userAgent)

	resp, err := d.client.Do(req)
	if err != nil {
		d.errors = append(d.errors, fmt.Errorf("Error getting %s: %w", pageURL, err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		d.errors = append(d.errors, fmt.Errorf("bad status for %s: %s", pageURL, resp.Status))
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		d.errors = append(d.errors, fmt.Errorf("Error reading body of %s: %w", pageURL, err))
		return
	}

	relPath := d.getRelativePath(pageURL)
	d.dn.AddData(relPath, body)

	// Don't try to parse non-html content
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
		return
	}

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		d.errors = append(d.errors, fmt.Errorf("Error parsing HTML of %s: %w", pageURL, err))
		return
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, a := range n.Attr {
				if a.Key == "href" || a.Key == "src" {
					link, err := d.resolveURL(pageURL, a.Val)
					if err != nil {
						continue
					}
					if d.isLocal(link) {
						if isAsset(link) {
							d.downloadAsset(link)
						} else {
							d.crawl(link, depth+1)
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
}

func (d *Downloader) downloadAsset(assetURL string) {
	if d.visited[assetURL] {
		return
	}

	u, err := url.Parse(assetURL)
	if err != nil {
		d.errors = append(d.errors, fmt.Errorf("invalid URL %s: %w", assetURL, err))
		return
	}

	if d.robots != nil && !d.robots.IsAllowed(u.Path) {
		return
	}

	d.visited[assetURL] = true
	if d.progressBar != nil {
		d.progressBar.Add(1)
	}

	d.delay()

	req, _ := http.NewRequest("GET", assetURL, nil)
	req.Header.Set("User-Agent", d.userAgent)

	resp, err := d.client.Do(req)
	if err != nil {
		d.errors = append(d.errors, fmt.Errorf("Error getting asset %s: %w", assetURL, err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		d.errors = append(d.errors, fmt.Errorf("bad status for asset %s: %s", assetURL, resp.Status))
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		d.errors = append(d.errors, fmt.Errorf("Error reading body of asset %s: %w", assetURL, err))
		return
	}

	relPath := d.getRelativePath(assetURL)
	d.dn.AddData(relPath, body)
}

func (d *Downloader) delay() {
	var delay time.Duration
	if d.robots != nil {
		delay = d.robots.CrawlDelay
	}
	if d.minDelay > delay {
		delay = d.minDelay
	}
	if delay > 0 {
		time.Sleep(delay)
	}
}

func (d *Downloader) getRelativePath(pageURL string) string {
	u, err := url.Parse(pageURL)
	if err != nil {
		return ""
	}
	path := strings.TrimPrefix(u.Path, "/")
	if path == "" {
		return "index.html"
	}
	return path
}

func (d *Downloader) resolveURL(base, ref string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return "", err
	}
	return baseURL.ResolveReference(refURL).String(), nil
}

func (d *Downloader) isLocal(pageURL string) bool {
	u, err := url.Parse(pageURL)
	if err != nil {
		return false
	}
	return u.Hostname() == d.baseURL.Hostname()
}

func isAsset(pageURL string) bool {
	ext := []string{".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico"}
	for _, e := range ext {
		if strings.HasSuffix(pageURL, e) {
			return true
		}
	}
	return false
}
