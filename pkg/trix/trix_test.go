package trix

import (
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
