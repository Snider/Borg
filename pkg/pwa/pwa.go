package pwa

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

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

// FindManifestURL finds the manifest URL from a given HTML page.
func FindManifestURL(pageURL string) (string, error) {
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

// DownloadAndPackagePWA downloads all assets of a PWA and packages them into a tarball.
func DownloadAndPackagePWA(baseURL string, manifestURL string) ([]byte, error) {
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

	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	// Add the manifest to the archive
	hdr := &tar.Header{
		Name: "manifest.json",
		Mode: 0600,
		Size: int64(len(manifestBody)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write(manifestBody); err != nil {
		return nil, err
	}

	// Add the start_url to the archive
	if manifest.StartURL != "" {
		startURLAbs, err := resolveURL(manifestAbsURL.String(), manifest.StartURL)
		if err != nil {
			return nil, fmt.Errorf("could not resolve start_url: %w", err)
		}
		err = downloadAndAddFileToTar(tw, startURLAbs, manifest.StartURL)
		if err != nil {
			return nil, fmt.Errorf("failed to download start_url asset: %w", err)
		}
	}

	// Add the icons to the archive
	for _, icon := range manifest.Icons {
		iconURLAbs, err := resolveURL(manifestAbsURL.String(), icon.Src)
		if err != nil {
			fmt.Printf("Warning: could not resolve icon URL %s: %v\n", icon.Src, err)
			continue
		}
		err = downloadAndAddFileToTar(tw, iconURLAbs, icon.Src)
		if err != nil {
			fmt.Printf("Warning: failed to download icon %s: %v\n", icon.Src, err)
		}
	}

	// Add the base HTML to the archive
	baseURLAbs, _ := url.Parse(baseURL)
	err = downloadAndAddFileToTar(tw, baseURLAbs, "index.html")
	if err != nil {
		return nil, fmt.Errorf("failed to download base HTML: %w", err)
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

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

func downloadAndAddFileToTar(tw *tar.Writer, fileURL *url.URL, internalPath string) error {
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

	hdr := &tar.Header{
		Name: path.Clean(internalPath),
		Mode: 0600,
		Size: int64(len(data)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write(data); err != nil {
		return err
	}

	return nil
}
