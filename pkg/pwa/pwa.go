package pwa

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/schollz/progressbar/v3"

	"golang.org/x/net/html"
)

// Manifest represents a simple PWA manifest structure.
type Manifest struct {
	Name      string `json:"name"`
	ShortName string `json:"short_name"`
	StartURL  string `json:"start_url"`
	Icons     []Icon `json:"icons"`
}

// Icon represents an icon in the PWA manifest.
type Icon struct {
	Src   string `json:"src"`
	Sizes string `json:"sizes"`
	Type  string `json:"type"`
}

// FindManifest finds the manifest URL from a given HTML page.
func FindManifest(pageURL string) (string, error) {
	resp, err := http.Get(pageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", err
	}

	var manifestPath string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			isManifest := false
			for _, a := range n.Attr {
				if a.Key == "rel" && a.Val == "manifest" {
					isManifest = true
					break
				}
			}
			if isManifest {
				for _, a := range n.Attr {
					if a.Key == "href" {
						manifestPath = a.Val
						return // exit once found
					}
				}
			}
		}
		for c := n.FirstChild; c != nil && manifestPath == ""; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	if manifestPath == "" {
		return "", fmt.Errorf("manifest not found")
	}

	resolvedURL, err := resolveURL(pageURL, manifestPath)
	if err != nil {
		return "", fmt.Errorf("could not resolve manifest URL: %w", err)
	}

	return resolvedURL.String(), nil
}

// DownloadAndPackagePWA downloads all assets of a PWA and packages them into a DataNode.
func DownloadAndPackagePWA(baseURL string, manifestURL string, bar *progressbar.ProgressBar) (*datanode.DataNode, error) {
	if bar == nil {
		return nil, fmt.Errorf("progress bar cannot be nil")
	}
	manifestAbsURL, err := resolveURL(baseURL, manifestURL)
	if err != nil {
		return nil, fmt.Errorf("could not resolve manifest URL: %w", err)
	}

	resp, err := http.Get(manifestAbsURL.String())
	if err != nil {
		return nil, fmt.Errorf("could not download manifest: %w", err)
	}
	defer resp.Body.Close()

	manifestBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read manifest body: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestBody, &manifest); err != nil {
		return nil, fmt.Errorf("could not parse manifest JSON: %w", err)
	}

	dn := datanode.New()
	dn.AddData("manifest.json", manifestBody)

	if manifest.StartURL != "" {
		startURLAbs, err := resolveURL(manifestAbsURL.String(), manifest.StartURL)
		if err != nil {
			return nil, fmt.Errorf("could not resolve start_url: %w", err)
		}
		err = downloadAndAddFile(dn, startURLAbs, manifest.StartURL, bar)
		if err != nil {
			return nil, fmt.Errorf("failed to download start_url asset: %w", err)
		}
	}

	for _, icon := range manifest.Icons {
		iconURLAbs, err := resolveURL(manifestAbsURL.String(), icon.Src)
		if err != nil {
			fmt.Printf("Warning: could not resolve icon URL %s: %v\n", icon.Src, err)
			continue
		}
		err = downloadAndAddFile(dn, iconURLAbs, icon.Src, bar)
		if err != nil {
			fmt.Printf("Warning: failed to download icon %s: %v\n", icon.Src, err)
		}
	}

	baseURLAbs, _ := url.Parse(baseURL)
	err = downloadAndAddFile(dn, baseURLAbs, "index.html", bar)
	if err != nil {
		return nil, fmt.Errorf("failed to download base HTML: %w", err)
	}

	return dn, nil
}

// resolveURL resolves ref against base and returns the absolute URL.
func resolveURL(base, ref string) (*url.URL, error) {
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

// downloadAndAddFile downloads the content at fileURL and adds it to the DataNode under internalPath.
func downloadAndAddFile(dn *datanode.DataNode, fileURL *url.URL, internalPath string, bar *progressbar.ProgressBar) error {
	resp, err := http.Get(fileURL.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	dn.AddData(path.Clean(internalPath), data)
	bar.Add(1)
	return nil
}
