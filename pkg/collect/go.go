package collect

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Snider/Borg/pkg/datanode"
)

// GoProxyURL is the base URL for the Go module proxy.
const GoProxyURL = "https://proxy.golang.org"

// GoCollector is a collector for Go modules.
type GoCollector struct {
	client *http.Client
}

// NewGoCollector creates a new GoCollector.
func NewGoCollector() *GoCollector {
	return &GoCollector{
		client: http.DefaultClient,
	}
}

// Collect fetches a Go module and returns a DataNode.
func (c *GoCollector) Collect(modulePath string) (*datanode.DataNode, error) {
	versions, err := c.fetchModuleVersions(modulePath)
	if err != nil {
		return nil, fmt.Errorf("could not fetch module versions: %w", err)
	}

	dn := datanode.New()
	for _, version := range versions {
		if err := c.fetchAndAddSource(dn, modulePath, version); err != nil {
			return nil, fmt.Errorf("could not fetch source for version %s: %w", version, err)
		}
	}

	return dn, nil
}

func (c *GoCollector) fetchModuleVersions(modulePath string) ([]string, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/%s/@v/list", GoProxyURL, modulePath))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(body), "\n"), nil
}

func (c *GoCollector) fetchAndAddSource(dn *datanode.DataNode, modulePath, version string) error {
	resp, err := c.client.Get(fmt.Sprintf("%s/%s/@v/%s.zip", GoProxyURL, modulePath, version))
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

	dn.AddData(version+".zip", data)
	return nil
}
