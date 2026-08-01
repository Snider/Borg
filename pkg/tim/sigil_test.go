package tim

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Snider/Borg/pkg/trix"
)

func TestToFromSigil(t *testing.T) {
	// Create a TIM with some data
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	m.RootFS.AddData("hello.txt", []byte("Hello, World!"))
	m.RootFS.AddData("subdir/nested.txt", []byte("Nested content"))

	password := "testpassword123"

	// Encrypt
	stim, err := m.ToSigil(password)
	if err != nil {
		t.Fatalf("ToSigil() error = %v", err)
	}

	// Verify magic number
	if len(stim) < 4 || string(stim[:4]) != "STIM" {
		t.Errorf("Expected magic 'STIM', got '%s'", string(stim[:4]))
	}

	// Decrypt
	restored, err := FromSigil(stim, password)
	if err != nil {
		t.Fatalf("FromSigil() error = %v", err)
	}

	// Verify config matches
	if string(restored.Config) != string(m.Config) {
		t.Error("Config mismatch after round-trip")
	}

	// Verify rootfs file exists
	file, err := restored.RootFS.Open("hello.txt")
	if err != nil {
		t.Fatalf("Failed to open hello.txt: %v", err)
	}
	defer file.Close()
}

func TestFromSigilWrongPassword(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	stim, err := m.ToSigil("correct")
	if err != nil {
		t.Fatalf("ToSigil() error = %v", err)
	}

	_, err = FromSigil(stim, "wrong")
	if err == nil {
		t.Error("Expected error with wrong password")
	}
}

func TestToSigilEmptyPassword(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = m.ToSigil("")
	if err != ErrPasswordRequired {
		t.Errorf("Expected ErrPasswordRequired, got %v", err)
	}
}

func TestFromSigilEmptyPassword(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	stim, err := m.ToSigil("password")
	if err != nil {
		t.Fatalf("ToSigil() error = %v", err)
	}

	_, err = FromSigil(stim, "")
	if err != ErrPasswordRequired {
		t.Errorf("Expected ErrPasswordRequired, got %v", err)
	}
}

func TestCache(t *testing.T) {
	// Create a temporary directory for the cache
	tempDir, err := os.MkdirTemp("", "borg-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	password := "cachepassword"
	cache, err := NewCache(tempDir, password)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}

	// Create a TIM
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	m.RootFS.AddData("test.txt", []byte("Cache test content"))

	// Store
	if err := cache.Store("mytim", m); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Verify file exists
	if !cache.Exists("mytim") {
		t.Error("Exists() returned false for stored TIM")
	}

	// List
	names, err := cache.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(names) != 1 || names[0] != "mytim" {
		t.Errorf("List() = %v, want [mytim]", names)
	}

	// Load
	loaded, err := cache.Load("mytim")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify content
	file, err := loaded.RootFS.Open("test.txt")
	if err != nil {
		t.Fatalf("Failed to open test.txt: %v", err)
	}
	defer file.Close()

	// Delete
	if err := cache.Delete("mytim"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if cache.Exists("mytim") {
		t.Error("Exists() returned true after Delete()")
	}
}

func TestCacheEmptyPassword(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "borg-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	_, err = NewCache(tempDir, "")
	if err != ErrPasswordRequired {
		t.Errorf("Expected ErrPasswordRequired, got %v", err)
	}
}

func TestDeriveKey(t *testing.T) {
	key := trix.DeriveKey("password")
	if len(key) != 32 {
		t.Errorf("DeriveKey() returned key of length %d, want 32", len(key))
	}

	// Same password should produce same key
	key2 := trix.DeriveKey("password")
	for i := range key {
		if key[i] != key2[i] {
			t.Error("DeriveKey() not deterministic")
			break
		}
	}

	// Different password should produce different key
	key3 := trix.DeriveKey("different")
	same := true
	for i := range key {
		if key[i] != key3[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("DeriveKey() produced same key for different passwords")
	}
}

func TestToSigilWithLargeData(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Add large file
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	m.RootFS.AddData("large.bin", largeData)

	password := "largetest"

	// Encrypt
	stim, err := m.ToSigil(password)
	if err != nil {
		t.Fatalf("ToSigil() error = %v", err)
	}

	// Decrypt
	restored, err := FromSigil(stim, password)
	if err != nil {
		t.Fatalf("FromSigil() error = %v", err)
	}

	// Verify file exists
	_, err = restored.RootFS.Open("large.bin")
	if err != nil {
		t.Fatalf("Failed to open large.bin: %v", err)
	}
}

func TestRunEncryptedFileNotFound(t *testing.T) {
	err := RunEncrypted("/nonexistent/path.stim", "password")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestRunEncryptedWithTempFile(t *testing.T) {
	// Create a TIM
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	m.RootFS.AddData("test.txt", []byte("test"))

	// Encrypt
	password := "runtest"
	stim, err := m.ToSigil(password)
	if err != nil {
		t.Fatalf("ToSigil() error = %v", err)
	}

	// Write to temp file
	tempFile := filepath.Join(t.TempDir(), "test.stim")
	if err := os.WriteFile(tempFile, stim, 0600); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	// RunEncrypted will fail because runc is not available in test,
	// but it should at least decrypt successfully
	err = RunEncrypted(tempFile, password)
	// We expect an error about runc not being found, not about decryption
	if err != nil && err.Error() != "" {
		// Check it's not a decryption error
		if err.Error() == ErrDecryptionFailed.Error() {
			t.Errorf("Unexpected decryption error: %v", err)
		}
	}
}

func TestCacheSize(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "borg-cache-size-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	password := "sizetest"
	cache, err := NewCache(tempDir, password)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}

	// Create and store a TIM
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	m.RootFS.AddData("test.txt", []byte("Size test content"))

	if err := cache.Store("sizetest", m); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Get size
	size, err := cache.Size("sizetest")
	if err != nil {
		t.Fatalf("Size() error = %v", err)
	}

	if size <= 0 {
		t.Errorf("Size() = %d, want > 0", size)
	}
}

func TestCacheSizeNotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "borg-cache-size-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewCache(tempDir, "password")
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}

	_, err = cache.Size("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent TIM")
	}
}

func TestFromTar(t *testing.T) {
	// Create a TIM and convert to tar
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	m.RootFS.AddData("test.txt", []byte("Test content"))
	m.RootFS.AddData("subdir/nested.txt", []byte("Nested content"))

	tarData, err := m.ToTar()
	if err != nil {
		t.Fatalf("ToTar() error = %v", err)
	}

	// Parse the tar back to a TIM
	restored, err := FromTar(tarData)
	if err != nil {
		t.Fatalf("FromTar() error = %v", err)
	}

	// Verify config
	if string(restored.Config) != string(m.Config) {
		t.Error("Config mismatch after FromTar")
	}

	// Verify files
	file, err := restored.RootFS.Open("test.txt")
	if err != nil {
		t.Fatalf("Failed to open test.txt: %v", err)
	}
	file.Close()

	file, err = restored.RootFS.Open("subdir/nested.txt")
	if err != nil {
		t.Fatalf("Failed to open subdir/nested.txt: %v", err)
	}
	file.Close()
}

func TestFromTarInvalidData(t *testing.T) {
	_, err := FromTar([]byte("not a tar file"))
	if err == nil {
		t.Error("Expected error for invalid tar data")
	}
}

func TestFromTarNoConfig(t *testing.T) {
	// Create tar without config.json
	_, err := FromTar([]byte{})
	if err == nil {
		t.Error("Expected error for tar without config.json")
	}
}

func TestToTarNilConfig(t *testing.T) {
	m := &TerminalIsolationMatrix{
		Config: nil,
		RootFS: nil,
	}

	_, err := m.ToTar()
	if err != ErrConfigIsNil {
		t.Errorf("Expected ErrConfigIsNil, got %v", err)
	}
}

func TestToSigilNilConfig(t *testing.T) {
	m := &TerminalIsolationMatrix{
		Config: nil,
		RootFS: nil,
	}

	_, err := m.ToSigil("password")
	if err != ErrConfigIsNil {
		t.Errorf("Expected ErrConfigIsNil, got %v", err)
	}
}

func TestFromSigilInvalidData(t *testing.T) {
	_, err := FromSigil([]byte("invalid stim data"), "password")
	if err == nil {
		t.Error("Expected error for invalid stim data")
	}
}

func TestFromSigilTruncatedPayload(t *testing.T) {
	// Create valid STIM but truncate the payload
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	stim, err := m.ToSigil("password")
	if err != nil {
		t.Fatalf("ToSigil() error = %v", err)
	}

	// Truncate to just magic + partial header
	if len(stim) > 20 {
		_, err = FromSigil(stim[:20], "password")
		if err == nil {
			t.Error("Expected error for truncated payload")
		}
	}
}

func TestFromDataNodeNil(t *testing.T) {
	_, err := FromDataNode(nil)
	if err != ErrDataNodeRequired {
		t.Errorf("Expected ErrDataNodeRequired, got %v", err)
	}
}
