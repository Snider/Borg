package website

import (
	"bytes"
	"fmt"
	"github.com/Snider/Borg/pkg/mocks"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/schollz/progressbar/v3"

	"golang.org/x/net/html"
)

var getHTTPClient = func() *http.Client {
	if os.Getenv("BORG_PLEXSUS") == "0" {
		responses := map[string]*http.Response{
			"http://test.com": {
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`<html><body><a href="page2.html">Page 2</a></body></html>`)),
				Header:     make(http.Header),
			},
			"http://test.com/page2.html": {
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`<html><body>Hello</body></html>`)),
				Header:     make(http.Header),
			},
		}
		return mocks.NewMockClient(responses)
	}
	return http.DefaultClient
}

// Downloader is a recursive website downloader.
type Downloader struct {
	baseURL     *url.URL
	dn          *datanode.DataNode
	visited     map[string]bool
	maxDepth    int
	progressBar *progressbar.ProgressBar
	client      *http.Client
}

// NewDownloader creates a new Downloader.
func NewDownloader(maxDepth int) *Downloader {
	return &Downloader{
		dn:       datanode.New(),
		visited:  make(map[string]bool),
		maxDepth: maxDepth,
		client:   getHTTPClient(),
	}
}

// DownloadAndPackageWebsite downloads a website and packages it into a DataNode.
func DownloadAndPackageWebsite(startURL string, maxDepth int, bar *progressbar.ProgressBar) (*datanode.DataNode, error) {
	baseURL, err := url.Parse(startURL)
	if err != nil {
		return nil, err
	}

	d := NewDownloader(maxDepth)
	d.baseURL = baseURL
	d.progressBar = bar
	d.crawl(startURL, 0)

	return d.dn, nil
}

// crawl visits pageURL, saves its content, and follows local links up to maxDepth.
func (d *Downloader) crawl(pageURL string, depth int) {
	if depth > d.maxDepth || d.visited[pageURL] {
		return
	}
	d.visited[pageURL] = true
	if d.progressBar != nil {
		d.progressBar.Add(1)
	}

	resp, err := d.client.Get(pageURL)
	if err != nil {
		fmt.Printf("Error getting %s: %v\n", pageURL, err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading body of %s: %v\n", pageURL, err)
		return
	}

	relPath := d.getRelativePath(pageURL)
	d.dn.AddData(relPath, body)

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		fmt.Printf("Error parsing HTML of %s: %v\n", pageURL, err)
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

// downloadAsset fetches an asset by URL and stores it in the DataNode.
func (d *Downloader) downloadAsset(assetURL string) {
	if d.visited[assetURL] {
		return
	}
	d.visited[assetURL] = true
	if d.progressBar != nil {
		d.progressBar.Add(1)
	}

	resp, err := d.client.Get(assetURL)
	if err != nil {
		fmt.Printf("Error getting asset %s: %v\n", assetURL, err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading body of asset %s: %v\n", assetURL, err)
		return
	}

	relPath := d.getRelativePath(assetURL)
	d.dn.AddData(relPath, body)
}

// getRelativePath returns the path within the DataNode for the given page URL.
func (d *Downloader) getRelativePath(pageURL string) string {
	u, err := url.Parse(pageURL)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(u.Path, "/")
}

// resolveURL resolves ref against base and returns the absolute URL string.
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

// isLocal reports whether pageURL shares the same hostname as the base URL.
func (d *Downloader) isLocal(pageURL string) bool {
	u, err := url.Parse(pageURL)
	if err != nil {
		return false
	}
	return u.Hostname() == d.baseURL.Hostname()
}

// isAsset reports whether the URL likely points to a static asset by file extension.
func isAsset(pageURL string) bool {
	ext := []string{".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico"}
	for _, e := range ext {
		if strings.HasSuffix(pageURL, e) {
			return true
		}
	}
	return false
}
