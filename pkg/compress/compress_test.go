package compress

import (
	"bytes"
	"testing"
)

func TestCompressDecompress(t *testing.T) {
	testData := []byte("hello, world")

	// Test gzip compression
	compressedGz, err := Compress(testData, "gz")
	if err != nil {
		t.Fatalf("gzip compression failed: %v", err)
	}

	decompressedGz, err := Decompress(compressedGz)
	if err != nil {
		t.Fatalf("gzip decompression failed: %v", err)
	}

	if !bytes.Equal(testData, decompressedGz) {
		t.Errorf("gzip decompressed data does not match original data")
	}

	// Test xz compression
	compressedXz, err := Compress(testData, "xz")
	if err != nil {
		t.Fatalf("xz compression failed: %v", err)
	}

	decompressedXz, err := Decompress(compressedXz)
	if err != nil {
		t.Fatalf("xz decompression failed: %v", err)
	}

	if !bytes.Equal(testData, decompressedXz) {
		t.Errorf("xz decompressed data does not match original data")
	}

	// Test no compression
	compressedNone, err := Compress(testData, "none")
	if err != nil {
		t.Fatalf("no compression failed: %v", err)
	}

	if !bytes.Equal(testData, compressedNone) {
		t.Errorf("no compression data does not match original data")
	}

	decompressedNone, err := Decompress(compressedNone)
	if err != nil {
		t.Fatalf("no compression decompression failed: %v", err)
	}

	if !bytes.Equal(testData, decompressedNone) {
		t.Errorf("no compression decompressed data does not match original data")
	}
}
