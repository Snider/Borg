package tim

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"
)

func TestStreamRoundTrip_Good(t *testing.T) {
	plaintext := []byte("Hello, STIM v2 streaming encryption!")
	password := "test-password-123"

	// Encrypt
	var cipherBuf bytes.Buffer
	if err := StreamEncrypt(bytes.NewReader(plaintext), &cipherBuf, password); err != nil {
		t.Fatalf("StreamEncrypt() error = %v", err)
	}

	// Verify header magic
	encrypted := cipherBuf.Bytes()
	if len(encrypted) < 5 {
		t.Fatal("encrypted output too short for header")
	}
	if string(encrypted[:4]) != "STIM" {
		t.Errorf("expected magic 'STIM', got %q", string(encrypted[:4]))
	}
	if encrypted[4] != 2 {
		t.Errorf("expected version 2, got %d", encrypted[4])
	}

	// Decrypt
	var plainBuf bytes.Buffer
	if err := StreamDecrypt(bytes.NewReader(encrypted), &plainBuf, password); err != nil {
		t.Fatalf("StreamDecrypt() error = %v", err)
	}

	if !bytes.Equal(plainBuf.Bytes(), plaintext) {
		t.Errorf("round-trip mismatch:\n  got:  %q\n  want: %q", plainBuf.Bytes(), plaintext)
	}
}

func TestStreamRoundTrip_Large_Good(t *testing.T) {
	// 3 MiB of pseudo-random data spans multiple 1 MiB blocks
	plaintext := make([]byte, 3*1024*1024)
	if _, err := rand.Read(plaintext); err != nil {
		t.Fatalf("failed to generate random data: %v", err)
	}

	password := "large-data-password"

	// Encrypt
	var cipherBuf bytes.Buffer
	if err := StreamEncrypt(bytes.NewReader(plaintext), &cipherBuf, password); err != nil {
		t.Fatalf("StreamEncrypt() error = %v", err)
	}

	// Decrypt
	var plainBuf bytes.Buffer
	if err := StreamDecrypt(bytes.NewReader(cipherBuf.Bytes()), &plainBuf, password); err != nil {
		t.Fatalf("StreamDecrypt() error = %v", err)
	}

	if !bytes.Equal(plainBuf.Bytes(), plaintext) {
		t.Errorf("round-trip mismatch: got %d bytes, want %d bytes", plainBuf.Len(), len(plaintext))
	}
}

func TestStreamEncrypt_Empty_Good(t *testing.T) {
	password := "empty-test"

	// Encrypt empty input
	var cipherBuf bytes.Buffer
	if err := StreamEncrypt(bytes.NewReader(nil), &cipherBuf, password); err != nil {
		t.Fatalf("StreamEncrypt() error = %v", err)
	}

	// Decrypt
	var plainBuf bytes.Buffer
	if err := StreamDecrypt(bytes.NewReader(cipherBuf.Bytes()), &plainBuf, password); err != nil {
		t.Fatalf("StreamDecrypt() error = %v", err)
	}

	if plainBuf.Len() != 0 {
		t.Errorf("expected empty output, got %d bytes", plainBuf.Len())
	}
}

func TestStreamDecrypt_WrongPassword_Bad(t *testing.T) {
	plaintext := []byte("secret data that should not decrypt with wrong key")
	correctPassword := "correct-password"
	wrongPassword := "wrong-password"

	// Encrypt with correct password
	var cipherBuf bytes.Buffer
	if err := StreamEncrypt(bytes.NewReader(plaintext), &cipherBuf, correctPassword); err != nil {
		t.Fatalf("StreamEncrypt() error = %v", err)
	}

	// Attempt decrypt with wrong password
	var plainBuf bytes.Buffer
	err := StreamDecrypt(bytes.NewReader(cipherBuf.Bytes()), &plainBuf, wrongPassword)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong password, got nil")
	}
}

func TestStreamDecrypt_Truncated_Bad(t *testing.T) {
	plaintext := []byte("data that will be truncated after encryption")
	password := "truncation-test"

	// Encrypt
	var cipherBuf bytes.Buffer
	if err := StreamEncrypt(bytes.NewReader(plaintext), &cipherBuf, password); err != nil {
		t.Fatalf("StreamEncrypt() error = %v", err)
	}

	encrypted := cipherBuf.Bytes()

	// Truncate to just past the header (33 bytes) but before the full first block
	if len(encrypted) > 40 {
		truncated := encrypted[:40]
		var plainBuf bytes.Buffer
		err := StreamDecrypt(bytes.NewReader(truncated), &plainBuf, password)
		if err == nil {
			t.Fatal("expected error when decrypting truncated data, got nil")
		}
	}

	// Truncate mid-way through the ciphertext
	if len(encrypted) > headerSize+nonceSize+lengthSize+5 {
		midpoint := headerSize + nonceSize + lengthSize + 5
		truncated := encrypted[:midpoint]
		var plainBuf bytes.Buffer
		err := StreamDecrypt(bytes.NewReader(truncated), &plainBuf, password)
		if err == nil {
			t.Fatal("expected error when decrypting mid-block truncated data, got nil")
		}
	}
}

func TestStreamDecrypt_InvalidMagic_Bad(t *testing.T) {
	// Construct data with wrong magic
	data := []byte("NOPE\x02")
	data = append(data, make([]byte, 28)...) // pad to header size

	var plainBuf bytes.Buffer
	err := StreamDecrypt(bytes.NewReader(data), &plainBuf, "password")
	if err == nil {
		t.Fatal("expected error for invalid magic, got nil")
	}
}

func TestStreamDecrypt_InvalidVersion_Bad(t *testing.T) {
	// Construct data with wrong version
	data := []byte("STIM\x01")
	data = append(data, make([]byte, 28)...) // pad to header size

	var plainBuf bytes.Buffer
	err := StreamDecrypt(bytes.NewReader(data), &plainBuf, "password")
	if err == nil {
		t.Fatal("expected error for unsupported version, got nil")
	}
}

func TestStreamDecrypt_ShortHeader_Bad(t *testing.T) {
	// Too short to contain full header
	data := []byte("STIM\x02")
	var plainBuf bytes.Buffer
	err := StreamDecrypt(bytes.NewReader(data), &plainBuf, "password")
	if err == nil {
		t.Fatal("expected error for short header, got nil")
	}
}

func TestStreamEncrypt_WriterError_Bad(t *testing.T) {
	plaintext := []byte("test data")
	// Use a writer that fails after a few bytes
	w := &limitedWriter{limit: 5}
	err := StreamEncrypt(bytes.NewReader(plaintext), w, "password")
	if err == nil {
		t.Fatal("expected error when writer fails, got nil")
	}
}

// limitedWriter fails after writing limit bytes.
type limitedWriter struct {
	limit   int
	written int
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	remaining := w.limit - w.written
	if remaining <= 0 {
		return 0, io.ErrShortWrite
	}
	if len(p) > remaining {
		w.written += remaining
		return remaining, io.ErrShortWrite
	}
	w.written += len(p)
	return len(p), nil
}
