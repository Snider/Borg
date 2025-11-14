package pwa

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/net/html"
)

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

	if manifestURL == "" {
		return "", fmt.Errorf("manifest not found")
	}

	resolvedURL, err := p.resolveURL(pwaURL, manifestURL)
	if err != nil {
		return "", err
	}

	return resolvedURL.String(), nil
}

// DownloadAndPackagePWA downloads and packages a PWA into a DataNode.
func (p *pwaClient) DownloadAndPackagePWA(pwaURL, manifestURL string, bar *progressbar.ProgressBar) (*datanode.DataNode, error) {
	dn := datanode.New()
	var wg sync.WaitGroup
	var errs []error
	var mu sync.Mutex

	type Manifest struct {
		StartURL string `json:"start_url"`
		Icons    []struct {
			Src string `json:"src"`
		} `json:"icons"`
	}

	downloadAndAdd := func(assetURL string) {
		defer wg.Done()
		if bar != nil {
			bar.Add(1)
		}
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
		dn.AddData(strings.TrimPrefix(u.Path, "/"), body)
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

	// Parse manifest and download assets concurrently.
	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	assetsToDownload := []string{}
	if manifest.StartURL != "" {
		startURL, err := p.resolveURL(manifestURL, manifest.StartURL)
		if err == nil {
			assetsToDownload = append(assetsToDownload, startURL.String())
		}
	}
	for _, icon := range manifest.Icons {
		if icon.Src != "" {
			iconURL, err := p.resolveURL(manifestURL, icon.Src)
			if err == nil {
				assetsToDownload = append(assetsToDownload, iconURL.String())
			}
		}
	}

	wg.Add(len(assetsToDownload))
	for _, asset := range assetsToDownload {
		go downloadAndAdd(asset)
	}
	wg.Wait()

	if len(errs) > 0 {
		var errStrings []string
		for _, e := range errs {
			errStrings = append(errStrings, e.Error())
		}
		return dn, fmt.Errorf("%s", strings.Join(errStrings, "; "))
	}

	return dn, nil
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
