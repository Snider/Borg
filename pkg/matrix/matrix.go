package matrix

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io/fs"

	"github.com/Snider/Borg/pkg/datanode"
)

// TerminalIsolationMatrix represents a runc bundle.
type TerminalIsolationMatrix struct {
	Config []byte
	RootFS *datanode.DataNode
}

// New creates a new, empty TerminalIsolationMatrix.
func New() (*TerminalIsolationMatrix, error) {
	// Use the default runc spec as a starting point.
	// This can be customized later.
	spec, err := defaultConfig()
	if err != nil {
		return nil, err
	}

	specBytes, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}

	return &TerminalIsolationMatrix{
		Config: specBytes,
		RootFS: datanode.New(),
	}, nil
}

// FromDataNode creates a new TerminalIsolationMatrix from a DataNode.
func FromDataNode(dn *datanode.DataNode) (*TerminalIsolationMatrix, error) {
	m, err := New()
	if err != nil {
		return nil, err
	}
	m.RootFS = dn
	return m, nil
}

// ToTar serializes the TerminalIsolationMatrix to a tarball.
func (m *TerminalIsolationMatrix) ToTar() ([]byte, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	// Add the config.json file.
	hdr := &tar.Header{
		Name: "config.json",
		Mode: 0600,
		Size: int64(len(m.Config)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write(m.Config); err != nil {
		return nil, err
	}

	// Add the rootfs files.
	err := m.RootFS.Walk(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		file, err := m.RootFS.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		info, err := file.Stat()
		if err != nil {
			return err
		}

		hdr := &tar.Header{
			Name: "rootfs/" + path,
			Mode: 0600,
			Size: info.Size(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(file); err != nil {
			return err
		}

		if _, err := tw.Write(buf.Bytes()); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
