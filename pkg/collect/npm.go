package collect

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/Snider/Borg/pkg/datanode"
)

// NPMRegistryURL is the base URL for the npm registry.
const NPMRegistryURL = "https://registry.npmjs.org"

// NPMCollector is a collector for npm packages.
type NPMCollector struct {
	client *http.Client
}

// NewNPMCollector creates a new NPMCollector.
func NewNPMCollector() *NPMCollector {
	return &NPMCollector{
		client: http.DefaultClient,
	}
}

// Collect fetches an npm package and returns a DataNode.
func (c *NPMCollector) Collect(packageName string) (*datanode.DataNode, error) {
	meta, err := c.fetchPackageMetadata(packageName)
	if err != nil {
		return nil, fmt.Errorf("could not fetch package metadata: %w", err)
	}

	dn := datanode.New()
	metadata, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("could not marshal metadata: %w", err)
	}
	dn.AddData("metadata.json", metadata)

	for version, data := range meta.Versions {
		if err := c.fetchAndAddTarball(dn, data.Dist.Tarball, version+".tgz"); err != nil {
			// It is a valid use case to only collect metadata
			log.Printf("could not fetch tarball for version %s: %v", version, err)
		}
	}

	return dn, nil
}

func (c *NPMCollector) fetchAndAddTarball(dn *datanode.DataNode, url, filename string) error {
	resp, err := c.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	data, err := c.readBody(resp.Body)
	if err != nil {
		return err
	}
	dn.AddData(filename, data)
	return nil
}

func (c *NPMCollector) fetchPackageMetadata(packageName string) (*NPMPackage, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/%s", NPMRegistryURL, packageName))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	var pkg NPMPackage
	if err := json.NewDecoder(resp.Body).Decode(&pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}

func (c *NPMCollector) readBody(body io.Reader) ([]byte, error) {
	return io.ReadAll(body)
}

// NPMPackage represents the metadata for an npm package.
type NPMPackage struct {
	Name     string                    `json:"name"`
	Versions map[string]NPMVersionData `json:"versions"`
}

// NPMVersionData represents the metadata for a specific version of an npm package.
type NPMVersionData struct {
	Dist struct {
		Tarball string `json:"tarball"`
	} `json:"dist"`
}
