package datanode

import (
	"archive/tar"
	"bytes"
	"testing"
)

func TestFromTar(t *testing.T) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	testData := "hello world"

	hdr := &tar.Header{
		Name: "test.txt",
		Mode: 0600,
		Size: int64(len(testData)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("failed to write header: %v", err)
	}
	if _, err := tw.Write([]byte(testData)); err != nil {
		t.Fatalf("failed to write content: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	dn, err := FromTar(buf.Bytes())
	if err != nil {
		t.Fatalf("FromTar failed: %v", err)
	}
	file, err := dn.Open("test.txt")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer file.Close()
}
