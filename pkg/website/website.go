package website

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/Snider/Borg/pkg/circuitbreaker"
	"github.com/Snider/Borg/pkg/datanode"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/net/html"
)

var DownloadAndPackageWebsite = downloadAndPackageWebsite

// DownloadOptions configures the website downloader.
type DownloadOptions struct {
	URL                  string
	MaxDepth             int
	ProgressBar          *progressbar.ProgressBar
	EnableCircuitBreaker bool
	CBSettings           circuitbreaker.Settings
}

// Downloader is a recursive website downloader.
type Downloader struct {
	baseURL         *url.URL
	dn              *datanode.DataNode
	visited         map[string]bool
	maxDepth        int
	progressBar     *progressbar.ProgressBar
	client          *http.Client
	errors          []error
	cbEnabled       bool
	cbSettings      circuitbreaker.Settings
	circuitBreakers map[string]*circuitbreaker.CircuitBreaker
	cbMutex         sync.Mutex
}

// NewDownloader creates a new Downloader.
func NewDownloader(opts DownloadOptions) *Downloader {
	return NewDownloaderWithClient(opts, http.DefaultClient)
}

// NewDownloaderWithClient creates a new Downloader with a custom http.Client.
func NewDownloaderWithClient(opts DownloadOptions, client *http.Client) *Downloader {
	return &Downloader{
		dn:              datanode.New(),
		visited:         make(map[string]bool),
		maxDepth:        opts.MaxDepth,
		client:          client,
		errors:          make([]error, 0),
		cbEnabled:       opts.EnableCircuitBreaker,
		cbSettings:      opts.CBSettings,
		circuitBreakers: make(map[string]*circuitbreaker.CircuitBreaker),
	}
}

// downloadAndPackageWebsite downloads a website and packages it into a DataNode.
func downloadAndPackageWebsite(opts DownloadOptions) (*datanode.DataNode, error) {
	baseURL, err := url.Parse(opts.URL)
	if err != nil {
		return nil, err
	}

	d := NewDownloader(opts)
	d.baseURL = baseURL
	d.progressBar = opts.ProgressBar
	d.crawl(opts.URL, 0)

	if len(d.errors) > 0 {
		var errs []string
		for _, e := range d.errors {
			errs = append(errs, e.Error())
		}
		return nil, fmt.Errorf("failed to download website:\n%s", strings.Join(errs, "\n"))
	}

	return d.dn, nil
}

func (d *Downloader) getCircuitBreaker(host string) *circuitbreaker.CircuitBreaker {
	d.cbMutex.Lock()
	defer d.cbMutex.Unlock()

	if cb, ok := d.circuitBreakers[host]; ok {
		return cb
	}

	cb := circuitbreaker.New(host, d.cbSettings)
	d.circuitBreakers[host] = cb
	return cb
}

func (d *Downloader) fetchURL(pageURL string) (*http.Response, error) {
	if !d.cbEnabled {
		resp, err := d.client.Get(pageURL)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 400 {
			resp.Body.Close()
			return nil, fmt.Errorf("bad status for %s: %s", pageURL, resp.Status)
		}
		return resp, nil
	}

	u, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	cb := d.getCircuitBreaker(u.Hostname())
	result, err := cb.Execute(func() (interface{}, error) {
		resp, err := d.client.Get(pageURL)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 400 {
			resp.Body.Close()
			return nil, fmt.Errorf("bad status for %s: %s", pageURL, resp.Status)
		}
		return resp, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*http.Response), nil
}

func (d *Downloader) crawl(pageURL string, depth int) {
	if depth > d.maxDepth || d.visited[pageURL] {
		return
	}
	d.visited[pageURL] = true
	if d.progressBar != nil {
		d.progressBar.Add(1)
	}

	resp, err := d.fetchURL(pageURL)
	if err != nil {
		d.errors = append(d.errors, fmt.Errorf("Error getting %s: %w", pageURL, err))
		return
	}
	defer resp.Body.Close()

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
	d.visited[assetURL] = true
	if d.progressBar != nil {
		d.progressBar.Add(1)
	}

	resp, err := d.fetchURL(assetURL)
	if err != nil {
		d.errors = append(d.errors, fmt.Errorf("Error getting asset %s: %w", assetURL, err))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		d.errors = append(d.errors, fmt.Errorf("Error reading body of asset %s: %w", assetURL, err))
		return
	}

	relPath := d.getRelativePath(assetURL)
	d.dn.AddData(relPath, body)
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
