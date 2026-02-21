package trix

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
)

func TestDeriveKey(t *testing.T) {
	// Test key length
	key := DeriveKey("password")
	if len(key) != 32 {
		t.Errorf("DeriveKey() returned key of length %d, want 32", len(key))
	}

	// Same password should produce same key
	key2 := DeriveKey("password")
	for i := range key {
		if key[i] != key2[i] {
			t.Error("DeriveKey() not deterministic")
			break
		}
	}

	// Different password should produce different key
	key3 := DeriveKey("different")
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

func TestToTrix(t *testing.T) {
	t.Run("without password", func(t *testing.T) {
		dn := datanode.New()
		dn.AddData("test.txt", []byte("Hello, World!"))

		data, err := ToTrix(dn, "")
		if err != nil {
			t.Fatalf("ToTrix() error = %v", err)
		}

		// Verify magic number
		if len(data) < 4 || string(data[:4]) != "TRIX" {
			t.Errorf("Expected magic 'TRIX', got '%s'", string(data[:4]))
		}
	})

	t.Run("with password", func(t *testing.T) {
		dn := datanode.New()
		dn.AddData("test.txt", []byte("Hello, World!"))

		data, err := ToTrix(dn, "secret")
		if err != nil {
			t.Fatalf("ToTrix() error = %v", err)
		}

		// Verify magic number
		if len(data) < 4 || string(data[:4]) != "TRIX" {
			t.Errorf("Expected magic 'TRIX', got '%s'", string(data[:4]))
		}
	})
}

func TestFromTrix(t *testing.T) {
	t.Run("without password round-trip", func(t *testing.T) {
		dn := datanode.New()
		dn.AddData("test.txt", []byte("Hello, World!"))

		data, err := ToTrix(dn, "")
		if err != nil {
			t.Fatalf("ToTrix() error = %v", err)
		}

		restored, err := FromTrix(data, "")
		if err != nil {
			t.Fatalf("FromTrix() error = %v", err)
		}

		// Verify file exists
		file, err := restored.Open("test.txt")
		if err != nil {
			t.Fatalf("Failed to open test.txt: %v", err)
		}
		defer file.Close()
	})

	t.Run("with password returns error", func(t *testing.T) {
		dn := datanode.New()
		dn.AddData("test.txt", []byte("Hello, World!"))

		data, err := ToTrix(dn, "")
		if err != nil {
			t.Fatalf("ToTrix() error = %v", err)
		}

		// FromTrix with password should return error (decryption disabled)
		_, err = FromTrix(data, "password")
		if err == nil {
			t.Error("Expected error when providing password to FromTrix")
		}
	})

	t.Run("invalid data", func(t *testing.T) {
		_, err := FromTrix([]byte("invalid"), "")
		if err == nil {
			t.Error("Expected error for invalid data")
		}
	})
}

func TestToTrixChaCha(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dn := datanode.New()
		dn.AddData("test.txt", []byte("Hello, World!"))

		data, err := ToTrixChaCha(dn, "password")
		if err != nil {
			t.Fatalf("ToTrixChaCha() error = %v", err)
		}

		// Verify magic number
		if len(data) < 4 || string(data[:4]) != "TRIX" {
			t.Errorf("Expected magic 'TRIX', got '%s'", string(data[:4]))
		}
	})

	t.Run("empty password", func(t *testing.T) {
		dn := datanode.New()
		dn.AddData("test.txt", []byte("Hello, World!"))

		_, err := ToTrixChaCha(dn, "")
		if err != ErrPasswordRequired {
			t.Errorf("Expected ErrPasswordRequired, got %v", err)
		}
	})
}

func TestFromTrixChaCha(t *testing.T) {
	t.Run("round-trip", func(t *testing.T) {
		dn := datanode.New()
		dn.AddData("test.txt", []byte("Hello, World!"))
		dn.AddData("subdir/nested.txt", []byte("Nested content"))

		password := "testpassword123"

		// Encrypt
		data, err := ToTrixChaCha(dn, password)
		if err != nil {
			t.Fatalf("ToTrixChaCha() error = %v", err)
		}

		// Decrypt
		restored, err := FromTrixChaCha(data, password)
		if err != nil {
			t.Fatalf("FromTrixChaCha() error = %v", err)
		}

		// Verify files exist
		file, err := restored.Open("test.txt")
		if err != nil {
			t.Fatalf("Failed to open test.txt: %v", err)
		}
		file.Close()

		file, err = restored.Open("subdir/nested.txt")
		if err != nil {
			t.Fatalf("Failed to open subdir/nested.txt: %v", err)
		}
		file.Close()
	})

	t.Run("empty password", func(t *testing.T) {
		_, err := FromTrixChaCha([]byte("data"), "")
		if err != ErrPasswordRequired {
			t.Errorf("Expected ErrPasswordRequired, got %v", err)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		dn := datanode.New()
		dn.AddData("test.txt", []byte("Hello, World!"))

		data, err := ToTrixChaCha(dn, "correct")
		if err != nil {
			t.Fatalf("ToTrixChaCha() error = %v", err)
		}

		_, err = FromTrixChaCha(data, "wrong")
		if err == nil {
			t.Error("Expected error with wrong password")
		}
	})

	t.Run("invalid data", func(t *testing.T) {
		_, err := FromTrixChaCha([]byte("invalid"), "password")
		if err == nil {
			t.Error("Expected error for invalid data")
		}
	})
}

func TestToTrixChaChaWithLargeData(t *testing.T) {
	dn := datanode.New()

	// Add large file
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	dn.AddData("large.bin", largeData)

	password := "largetest"

	// Encrypt
	data, err := ToTrixChaCha(dn, password)
	if err != nil {
		t.Fatalf("ToTrixChaCha() error = %v", err)
	}

	// Decrypt
	restored, err := FromTrixChaCha(data, password)
	if err != nil {
		t.Fatalf("FromTrixChaCha() error = %v", err)
	}

	// Verify file exists
	_, err = restored.Open("large.bin")
	if err != nil {
		t.Fatalf("Failed to open large.bin: %v", err)
	}
}

// --- Argon2id key derivation tests ---

func TestDeriveKeyArgon2_Good(t *testing.T) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		t.Fatalf("failed to generate salt: %v", err)
	}

	key := DeriveKeyArgon2("test-password", salt)
	if len(key) != 32 {
		t.Fatalf("expected 32-byte key, got %d bytes", len(key))
	}
}

func TestDeriveKeyArgon2_Deterministic_Good(t *testing.T) {
	salt := []byte("fixed-salt-value")

	key1 := DeriveKeyArgon2("same-password", salt)
	key2 := DeriveKeyArgon2("same-password", salt)

	if !bytes.Equal(key1, key2) {
		t.Fatal("same password and salt must produce the same key")
	}
}

func TestDeriveKeyArgon2_DifferentSalt_Good(t *testing.T) {
	salt1 := []byte("salt-one-value!!")
	salt2 := []byte("salt-two-value!!")

	key1 := DeriveKeyArgon2("same-password", salt1)
	key2 := DeriveKeyArgon2("same-password", salt2)

	if bytes.Equal(key1, key2) {
		t.Fatal("different salts must produce different keys")
	}
}

func TestDeriveKeyLegacy_Good(t *testing.T) {
	key1 := DeriveKey("backward-compat")
	key2 := DeriveKey("backward-compat")

	if len(key1) != 32 {
		t.Fatalf("expected 32-byte key, got %d bytes", len(key1))
	}
	if !bytes.Equal(key1, key2) {
		t.Fatal("legacy DeriveKey must be deterministic")
	}
}

func TestArgon2Params_Good(t *testing.T) {
	params := DefaultArgon2Params()

	// Non-zero values
	if params.Time == 0 {
		t.Fatal("Time must be non-zero")
	}
	if params.Memory == 0 {
		t.Fatal("Memory must be non-zero")
	}
	if params.Threads == 0 {
		t.Fatal("Threads must be non-zero")
	}

	// Encode produces 12 bytes (3 x uint32 LE)
	encoded := params.Encode()
	if len(encoded) != 12 {
		t.Fatalf("expected 12-byte encoding, got %d bytes", len(encoded))
	}

	// Round-trip: Decode must recover original values
	decoded := DecodeArgon2Params(encoded)
	if decoded.Time != params.Time {
		t.Fatalf("Time mismatch: got %d, want %d", decoded.Time, params.Time)
	}
	if decoded.Memory != params.Memory {
		t.Fatalf("Memory mismatch: got %d, want %d", decoded.Memory, params.Memory)
	}
	if decoded.Threads != params.Threads {
		t.Fatalf("Threads mismatch: got %d, want %d", decoded.Threads, params.Threads)
	}
}
