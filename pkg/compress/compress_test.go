package compress

import (
	"bytes"
	"testing"
)

func TestGzip_Good(t *testing.T) {
	originalData := []byte("hello, gzip world")
	compressed, err := Compress(originalData, "gz")
	if err != nil {
		t.Fatalf("gzip compression failed: %v", err)
	}
	if bytes.Equal(originalData, compressed) {
		t.Fatal("gzip compressed data is the same as the original")
	}

	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("gzip decompression failed: %v", err)
	}

	if !bytes.Equal(originalData, decompressed) {
		t.Errorf("gzip decompressed data does not match original data")
	}
}

func TestXz_Good(t *testing.T) {
	originalData := []byte("hello, xz world")
	compressed, err := Compress(originalData, "xz")
	if err != nil {
		t.Fatalf("xz compression failed: %v", err)
	}
	if bytes.Equal(originalData, compressed) {
		t.Fatal("xz compressed data is the same as the original")
	}

	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("xz decompression failed: %v", err)
	}

	if !bytes.Equal(originalData, decompressed) {
		t.Errorf("xz decompressed data does not match original data")
	}
}

func TestNone_Good(t *testing.T) {
	originalData := []byte("hello, none world")
	compressed, err := Compress(originalData, "none")
	if err != nil {
		t.Fatalf("'none' compression failed: %v", err)
	}
	if !bytes.Equal(originalData, compressed) {
		t.Errorf("'none' compression should not change data")
	}

	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("'none' decompression failed: %v", err)
	}

	if !bytes.Equal(originalData, decompressed) {
		t.Errorf("'none' decompressed data does not match original data")
	}
}

func TestCompress_Bad(t *testing.T) {
	originalData := []byte("test")
	// The function should return the original data for an unknown format.
	compressed, err := Compress(originalData, "invalid-format")
	if err != nil {
		t.Fatalf("expected no error for invalid compression format, got %v", err)
	}
	if !bytes.Equal(originalData, compressed) {
		t.Errorf("expected original data for unknown format, got %q", compressed)
	}
}

func TestDecompress_Bad(t *testing.T) {
	// A truncated gzip stream should cause a decompression error.
	originalData := []byte("hello, gzip world")
	compressed, _ := Compress(originalData, "gz")
	truncated := compressed[:len(compressed)-5] // Corrupt the stream

	_, err := Decompress(truncated)
	if err == nil {
		t.Fatal("expected an error when decompressing a truncated stream, got nil")
	}
}

func TestCompress_Ugly(t *testing.T) {
	// Test compressing empty data
	originalData := []byte{}
	compressed, err := Compress(originalData, "gz")
	if err != nil {
		t.Fatalf("compressing empty data failed: %v", err)
	}

	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("decompressing empty compressed data failed: %v", err)
	}

	if !bytes.Equal(originalData, decompressed) {
		t.Errorf("expected empty data, got %q", decompressed)
	}
}

func TestDecompress_Ugly(t *testing.T) {
	// Test decompressing empty byte slice
	result, err := Decompress([]byte{})
	if err != nil {
		t.Fatalf("decompressing an empty slice should not produce an error, got %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result from decompressing empty slice, got %q", result)
	}
}
