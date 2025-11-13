package matrix

import (
	"archive/tar"
	"bytes"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
)

func TestNew(t *testing.T) {
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
}

func TestFromDataNode(t *testing.T) {
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
}

func TestToTar(t *testing.T) {
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
	foundConfig := false
	foundRootFS := false
	foundTestFile := false
	for {
		header, err := tr.Next()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatalf("failed to read tar header: %v", err)
		}

		switch header.Name {
		case "config.json":
			foundConfig = true
		case "rootfs/":
			foundRootFS = true
		case "rootfs/test.txt":
			foundTestFile = true
		}
	}

	if !foundConfig {
		t.Error("config.json not found in matrix")
	}
	if !foundRootFS {
		t.Error("rootfs/ not found in matrix")
	}
	if !foundTestFile {
		t.Error("rootfs/test.txt not found in matrix")
	}
}
