// Package pwa provides functionality for discovering and downloading Progressive
// Web Application (PWA) assets.
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

// PWAClient defines the interface for interacting with Progressive Web Apps.
// This allows for mocking the client in tests.
type PWAClient interface {
	// FindManifest discovers the web app manifest URL for a given PWA URL.
	FindManifest(pwaURL string) (string, error)
	// DownloadAndPackagePWA downloads all the assets of a PWA, including the
	// manifest, start URL, and icons, and packages them into a DataNode.
	DownloadAndPackagePWA(pwaURL, manifestURL string, bar *progressbar.ProgressBar) (*datanode.DataNode, error)
}

// NewPWAClient creates and returns a new PWAClient that uses a default
// http.Client.
//
// Example:
//
//	client := pwa.NewPWAClient()
//	manifestURL, err := client.FindManifest("https://example.com")
//	if err != nil {
//		// handle error
//	}
func NewPWAClient() PWAClient {
	return &pwaClient{client: http.DefaultClient}
}

type pwaClient struct {
	client *http.Client
}

// FindManifest discovers the web app manifest URL for a given PWA URL. It does
// this by fetching the PWA's HTML and looking for a <link rel="manifest"> tag.
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

// DownloadAndPackagePWA downloads all the assets of a PWA, including the
// manifest, start URL, and icons, and packages them into a DataNode. It
// downloads the assets concurrently for performance.
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

// MockPWAClient is a mock implementation of the PWAClient interface, used for
// testing. It allows setting a predefined manifest URL, DataNode, and error to
// be returned by its methods.
type MockPWAClient struct {
	// ManifestURL is the manifest URL to be returned by FindManifest.
	ManifestURL string
	// DN is the DataNode to be returned by DownloadAndPackagePWA.
	DN *datanode.DataNode
	// Err is the error to be returned by the mock methods.
	Err error
}

// NewMockPWAClient creates a new MockPWAClient with the given manifest URL,
// DataNode, and error. This is a convenience function for creating a mock PWA
// client for tests.
//
// Example:
//
//	mockDN := datanode.New()
//	mockDN.AddData("manifest.json", []byte("{}"))
//	mockClient := pwa.NewMockPWAClient("https://example.com/manifest.json", mockDN, nil)
//	// use mockClient in tests
func NewMockPWAClient(manifestURL string, dn *datanode.DataNode, err error) PWAClient {
	return &MockPWAClient{
		ManifestURL: manifestURL,
		DN:          dn,
		Err:         err,
	}
}

// FindManifest is the mock implementation of the PWAClient interface. It
// returns the pre-configured manifest URL and error.
func (m *MockPWAClient) FindManifest(pwaURL string) (string, error) {
	return m.ManifestURL, m.Err
}

// DownloadAndPackagePWA is the mock implementation of the PWAClient interface.
// It returns the pre-configured DataNode and error.
func (m *MockPWAClient) DownloadAndPackagePWA(pwaURL, manifestURL string, bar *progressbar.ProgressBar) (*datanode.DataNode, error) {
	return m.DN, m.Err
}
