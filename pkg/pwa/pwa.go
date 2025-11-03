package pwa

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

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

	type Manifest struct {
		StartURL string `json:"start_url"`
		Icons    []struct {
			Src string `json:"src"`
		} `json:"icons"`
	}

	downloadAndAdd := func(assetURL string) error {
		if bar != nil {
			bar.Add(1)
		}
		resp, err := p.client.Get(assetURL)
		if err != nil {
			return fmt.Errorf("failed to download %s: %w", assetURL, err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read body of %s: %w", assetURL, err)
		}

		u, err := url.Parse(assetURL)
		if err != nil {
			return fmt.Errorf("failed to parse asset URL %s: %w", assetURL, err)
		}
		dn.AddData(strings.TrimPrefix(u.Path, "/"), body)
		return nil
	}

	// Download manifest
	if err := downloadAndAdd(manifestURL); err != nil {
		return nil, err
	}

	// Parse manifest and download assets
	var manifestPath string
	u, parseErr := url.Parse(manifestURL)
	if parseErr != nil {
		manifestPath = "manifest.json"
	} else {
		manifestPath = strings.TrimPrefix(u.Path, "/")
	}

	manifestFile, err := dn.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest from datanode: %w", err)
	}
	defer manifestFile.Close()

	manifestData, err := io.ReadAll(manifestFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest from datanode: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Download start_url
	startURL, err := p.resolveURL(manifestURL, manifest.StartURL)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve start_url: %w", err)
	}
	if err := downloadAndAdd(startURL.String()); err != nil {
		return nil, err
	}

	// Download icons
	for _, icon := range manifest.Icons {
		iconURL, err := p.resolveURL(manifestURL, icon.Src)
		if err != nil {
			// Skip icons with bad URLs
			continue
		}
		downloadAndAdd(iconURL.String())
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
