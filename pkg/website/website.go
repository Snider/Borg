package website

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/net/html"
	"golang.org/x/time/rate"
	"sync"
)

var DownloadAndPackageWebsite = downloadAndPackageWebsite

// Downloader is a recursive website downloader.
type Downloader struct {
	baseURL     *url.URL
	dn          *datanode.DataNode
	visited     map[string]bool
	maxDepth    int
	parallel    int
	progressBar *progressbar.ProgressBar
	client      *http.Client
	errors      []error
	mu          sync.Mutex
	limiter     *rate.Limiter
}

// NewDownloader creates a new Downloader.
func NewDownloader(maxDepth, parallel int, rateLimit float64) *Downloader {
	return NewDownloaderWithClient(maxDepth, parallel, rateLimit, http.DefaultClient)
}

// NewDownloaderWithClient creates a new Downloader with a custom http.Client.
func NewDownloaderWithClient(maxDepth, parallel int, rateLimit float64, client *http.Client) *Downloader {
	var limiter *rate.Limiter
	if rateLimit > 0 {
		limiter = rate.NewLimiter(rate.Limit(rateLimit), 1)
	}
	return &Downloader{
		dn:          datanode.New(),
		visited:     make(map[string]bool),
		maxDepth:    maxDepth,
		parallel:    parallel,
		client:      client,
		errors:      make([]error, 0),
		limiter:     limiter,
	}
}

// downloadAndPackageWebsite downloads a website and packages it into a DataNode.
func downloadAndPackageWebsite(ctx context.Context, startURL string, maxDepth, parallel int, rateLimit float64, bar *progressbar.ProgressBar) (*datanode.DataNode, error) {
	baseURL, err := url.Parse(startURL)
	if err != nil {
		return nil, err
	}

	d := NewDownloader(maxDepth, parallel, rateLimit)
	d.baseURL = baseURL
	d.progressBar = bar
	d.crawl(ctx, startURL)

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if len(d.errors) > 0 {
		var errs []string
		for _, e := range d.errors {
			errs = append(errs, e.Error())
		}
		return nil, fmt.Errorf("failed to download website:\n%s", strings.Join(errs, "\n"))
	}

	return d.dn, nil
}

type crawlJob struct {
	url   string
	depth int
}

func (d *Downloader) crawl(ctx context.Context, startURL string) {
	var wg sync.WaitGroup
	var jobWg sync.WaitGroup
	jobChan := make(chan crawlJob, 100)

	for i := 0; i < d.parallel; i++ {
		wg.Add(1)
		go d.worker(ctx, &wg, &jobWg, jobChan)
	}

	jobWg.Add(1)
	jobChan <- crawlJob{url: startURL, depth: 0}

	go func() {
		jobWg.Wait()
		close(jobChan)
	}()

	wg.Wait()
}

func (d *Downloader) worker(ctx context.Context, wg *sync.WaitGroup, jobWg *sync.WaitGroup, jobChan chan crawlJob) {
	defer wg.Done()
	for job := range jobChan {
		func() {
			defer jobWg.Done()

			select {
			case <-ctx.Done():
				return
			default:
			}

			if job.depth > d.maxDepth {
				return
			}

			d.mu.Lock()
			if d.visited[job.url] {
				d.mu.Unlock()
				return
			}
			d.visited[job.url] = true
			d.mu.Unlock()

			if d.progressBar != nil {
				d.progressBar.Add(1)
			}

			if d.limiter != nil {
				d.limiter.Wait(ctx)
			}

			req, err := http.NewRequestWithContext(ctx, "GET", job.url, nil)
			if err != nil {
				d.addError(fmt.Errorf("Error creating request for %s: %w", job.url, err))
				return
			}
			resp, err := d.client.Do(req)
			if err != nil {
				d.addError(fmt.Errorf("Error getting %s: %w", job.url, err))
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 400 {
				d.addError(fmt.Errorf("bad status for %s: %s", job.url, resp.Status))
				return
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				d.addError(fmt.Errorf("Error reading body of %s: %w", job.url, err))
				return
			}

			relPath := d.getRelativePath(job.url)
			d.mu.Lock()
			d.dn.AddData(relPath, body)
			d.mu.Unlock()

			if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
				return
			}

			doc, err := html.Parse(strings.NewReader(string(body)))
			if err != nil {
				d.addError(fmt.Errorf("Error parsing HTML of %s: %w", job.url, err))
				return
			}

			var f func(*html.Node)
			f = func(n *html.Node) {
				if n.Type == html.ElementNode {
					for _, a := range n.Attr {
						if a.Key == "href" || a.Key == "src" {
							link, err := d.resolveURL(job.url, a.Val)
							if err != nil {
								continue
							}
							if d.isLocal(link) {
							select {
							case <-ctx.Done():
								return
							default:
								jobWg.Add(1)
								jobChan <- crawlJob{url: link, depth: job.depth + 1}
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
		}()
	}
}

func (d *Downloader) addError(err error) {
	d.mu.Lock()
	d.errors = append(d.errors, err)
	d.mu.Unlock()
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
