package tim

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
)

func TestNew_Good(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("New() returned an error: %v", err)
	}
	if m == nil {
		t.Fatal("New() returned a nil matrix")
	}
	if m.Config == nil {
		t.Error("New() returned a matrix with a nil config")
	}
	if m.RootFS == nil {
		t.Error("New() returned a matrix with a nil RootFS")
	}

	// Verify the config is valid JSON
	if !json.Valid(m.Config) {
		t.Error("New() returned a matrix with invalid JSON config")
	}
}

func TestFromDataNode_Good(t *testing.T) {
	dn := datanode.New()
	dn.AddData("test.txt", []byte("hello world"))
	m, err := FromDataNode(dn)
	if err != nil {
		t.Fatalf("FromDataNode() returned an error: %v", err)
	}
	if m == nil {
		t.Fatal("FromDataNode() returned a nil matrix")
	}
	if m.RootFS != dn {
		t.Error("FromDataNode() did not set the RootFS correctly")
	}
	if m.Config == nil {
		t.Error("FromDataNode() did not create a default config")
	}
}

func TestFromDataNode_Bad(t *testing.T) {
	_, err := FromDataNode(nil)
	if err == nil {
		t.Fatal("expected error when passing a nil datanode, but got nil")
	}
	if !errors.Is(err, ErrDataNodeRequired) {
		t.Errorf("expected ErrDataNodeRequired, got %v", err)
	}
}

func TestToTar_Good(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("New() returned an error: %v", err)
	}
	m.RootFS.AddData("test.txt", []byte("hello world"))
	tarball, err := m.ToTar()
	if err != nil {
		t.Fatalf("ToTar() returned an error: %v", err)
	}
	if tarball == nil {
		t.Fatal("ToTar() returned a nil tarball")
	}

	tr := tar.NewReader(bytes.NewReader(tarball))
	found := make(map[string]bool)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("failed to read tar header: %v", err)
		}
		found[header.Name] = true
	}

	expectedFiles := []string{"config.json", "rootfs/", "rootfs/test.txt"}
	for _, f := range expectedFiles {
		if !found[f] {
			t.Errorf("%s not found in matrix tarball", f)
		}
	}
}

func TestToTar_Ugly(t *testing.T) {
	t.Run("Empty RootFS", func(t *testing.T) {
		m, _ := New()
		tarball, err := m.ToTar()
		if err != nil {
			t.Fatalf("ToTar() with empty rootfs returned an error: %v", err)
		}
		tr := tar.NewReader(bytes.NewReader(tarball))
		found := make(map[string]bool)
		for {
			header, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("failed to read tar header: %v", err)
			}
			found[header.Name] = true
		}
		if !found["config.json"] {
			t.Error("config.json not found in matrix")
		}
		if !found["rootfs/"] {
			t.Error("rootfs/ directory not found in matrix")
		}
		if len(found) != 2 {
			t.Errorf("expected 2 files in tar, but found %d", len(found))
		}
	})

	t.Run("Nil Config", func(t *testing.T) {
		m, _ := New()
		m.Config = nil // This should not happen in practice
		_, err := m.ToTar()
		if err == nil {
			t.Fatal("expected error when Config is nil, but got nil")
		}
		if !errors.Is(err, ErrConfigIsNil) {
			t.Errorf("expected ErrConfigIsNil, got %v", err)
		}
	})
}
