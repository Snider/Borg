package collect

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Snider/Borg/pkg/datanode"
)

// CargoRegistryURL is the base URL for the cargo registry.
const CargoRegistryURL = "https://crates.io/api/v1"

// CargoCollector is a collector for cargo packages.
type CargoCollector struct {
	client *http.Client
}

// NewCargoCollector creates a new CargoCollector.
func NewCargoCollector() *CargoCollector {
	return &CargoCollector{
		client: &http.Client{},
	}
}

// Collect fetches a cargo package and returns a DataNode.
func (c *CargoCollector) Collect(crateName string) (*datanode.DataNode, error) {
	meta, err := c.fetchCrateMetadata(crateName)
	if err != nil {
		return nil, fmt.Errorf("could not fetch crate metadata: %w", err)
	}

	dn := datanode.New()
	metadata, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("could not marshal metadata: %w", err)
	}
	dn.AddData("metadata.json", metadata)

	for _, version := range meta.Versions {
		if err := c.fetchAndAddCrate(dn, version.DlPath, version.Num+".crate"); err != nil {
			return nil, fmt.Errorf("could not fetch crate for version %s: %w", version.Num, err)
		}
	}

	return dn, nil
}

func (c *CargoCollector) fetchCrateMetadata(crateName string) (*CargoCrate, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/crates/%s", CargoRegistryURL, crateName), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "git/oxide-0.38.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	var crate CargoCrate
	if err := json.NewDecoder(resp.Body).Decode(&crate); err != nil {
		return nil, err
	}
	return &crate, nil
}

func (c *CargoCollector) fetchAndAddCrate(dn *datanode.DataNode, downloadURL, filename string) error {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://crates.io%s", downloadURL), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "git/oxide-0.38.0")

	resp, err := c.client.Do(req)
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
	dn.AddData(filename, data)
	return nil
}

// CargoCrate represents the metadata for a cargo crate.
type CargoCrate struct {
	Crate    CargoCrateData    `json:"crate"`
	Versions []CargoVersionData `json:"versions"`
}

// CargoCrateData represents the metadata for a cargo crate.
type CargoCrateData struct {
	Name string `json:"name"`
}

// CargoVersionData represents the metadata for a specific version of a cargo crate.
type CargoVersionData struct {
	Num    string `json:"num"`
	DlPath string `json:"dl_path"`
}
