// Package tim provides types and functions for creating and manipulating
// Terminal Isolation Matrix (.tim) files, which are runc-compatible container
// bundles.
package tim

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"

	"github.com/Snider/Borg/pkg/datanode"
)

var (
	// ErrDataNodeRequired is returned when a DataNode is required but not provided.
	ErrDataNodeRequired = errors.New("datanode is required")
	// ErrConfigIsNil is returned when the config is nil.
	ErrConfigIsNil = errors.New("config is nil")
)

// TIM represents a runc-compatible container bundle. It consists of a runc
// configuration file (config.json) and a root filesystem (rootfs).
type TIM struct {
	// Config is the JSON representation of the runc configuration.
	Config []byte
	// RootFS is an in-memory filesystem representing the container's root.
	RootFS *datanode.DataNode
}

// New creates a new, empty TIM with a default runc
// configuration.
//
// Example:
//
//	m, err := tim.New()
//	if err != nil {
//		// handle error
//	}
//	m.RootFS.AddData("hello.txt", []byte("hello world"))
func New() (*TIM, error) {
	// Use the default runc spec as a starting point.
	// This can be customized later.
	spec, err := defaultConfigVar()
	if err != nil {
		return nil, err
	}

	specBytes, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}

	return &TIM{
		Config: specBytes,
		RootFS: datanode.New(),
	}, nil
}

// FromDataNode creates a new TIM using the provided DataNode
// as the root filesystem. It uses a default runc configuration.
//
// Example:
//
//	dn := datanode.New()
//	dn.AddData("my-file.txt", []byte("hello"))
//	m, err := tim.FromDataNode(dn)
//	if err != nil {
//		// handle error
//	}
func FromDataNode(dn *datanode.DataNode) (*TIM, error) {
	if dn == nil {
		return nil, ErrDataNodeRequired
	}
	m, err := New()
	if err != nil {
		return nil, err
	}
	m.RootFS = dn
	return m, nil
}

// ToTar serializes the TIM into a tar archive. The resulting
// tarball will contain a config.json file and a rootfs directory, making it
// compatible with runc.
//
// Example:
//
//	// Assuming 'm' is a *tim.TIM
//	tarData, err := m.ToTar()
//	if err != nil {
//		// handle error
//	}
//	err = os.WriteFile("my-bundle.tar", tarData, 0644)
//	if err != nil {
//		// handle error
//	}
func (m *TIM) ToTar() ([]byte, error) {
	if m.Config == nil {
		return nil, ErrConfigIsNil
	}
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

	// Add the rootfs directory.
	hdr = &tar.Header{
		Name:     "rootfs/",
		Mode:     0755,
		Typeflag: tar.TypeDir,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}

	// Add the rootfs files.
	err := m.RootFS.Walk(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// If the root directory doesn't exist (i.e. empty datanode), it's not an error.
			if path == "." && errors.Is(err, fs.ErrNotExist) {
				return nil
			}
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
