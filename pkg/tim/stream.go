package tim

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"

	borgtrix "github.com/Snider/Borg/pkg/trix"
)

const (
	blockSize  = 1024 * 1024 // 1 MiB plaintext blocks
	saltSize   = 16
	nonceSize  = 12 // chacha20poly1305.NonceSize
	lengthSize = 4
	headerSize = 33 // 4 (magic) + 1 (version) + 16 (salt) + 12 (argon2 params)
)

var (
	stimMagic = [4]byte{'S', 'T', 'I', 'M'}

	ErrInvalidMagic      = errors.New("invalid STIM magic header")
	ErrUnsupportedVersion = errors.New("unsupported STIM version")
	ErrStreamDecrypt     = errors.New("stream decryption failed")
)

// StreamEncrypt reads plaintext from r and writes STIM v2 chunked AEAD
// encrypted data to w. Each 1 MiB block is independently encrypted with
// ChaCha20-Poly1305 using a unique random nonce.
func StreamEncrypt(r io.Reader, w io.Writer, password string) error {
	// Generate random salt
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key using Argon2id with default params
	params := borgtrix.DefaultArgon2Params()
	key := borgtrix.DeriveKeyArgon2(password, salt)

	// Create AEAD cipher
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return fmt.Errorf("failed to create AEAD: %w", err)
	}

	// Write header: magic(4) + version(1) + salt(16) + argon2params(12) = 33 bytes
	header := make([]byte, headerSize)
	copy(header[0:4], stimMagic[:])
	header[4] = 2 // version
	copy(header[5:21], salt)
	copy(header[21:33], params.Encode())

	if _, err := w.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Encrypt data in blocks
	buf := make([]byte, blockSize)
	nonce := make([]byte, nonceSize)

	for {
		n, readErr := io.ReadFull(r, buf)

		if n > 0 {
			// Generate unique nonce for this block
			if _, err := rand.Read(nonce); err != nil {
				return fmt.Errorf("failed to generate nonce: %w", err)
			}

			// Encrypt: ciphertext includes the Poly1305 auth tag (16 bytes)
			ciphertext := aead.Seal(nil, nonce, buf[:n], nil)

			// Write [nonce(12)][length(4)][ciphertext(n+16)]
			if _, err := w.Write(nonce); err != nil {
				return fmt.Errorf("failed to write nonce: %w", err)
			}

			lenBuf := make([]byte, lengthSize)
			binary.LittleEndian.PutUint32(lenBuf, uint32(len(ciphertext)))
			if _, err := w.Write(lenBuf); err != nil {
				return fmt.Errorf("failed to write length: %w", err)
			}

			if _, err := w.Write(ciphertext); err != nil {
				return fmt.Errorf("failed to write ciphertext: %w", err)
			}
		}

		if readErr != nil {
			if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
				break
			}
			return fmt.Errorf("failed to read input: %w", readErr)
		}
	}

	// Write EOF marker: [nonce(12)][length=0(4)]
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("failed to generate EOF nonce: %w", err)
	}
	if _, err := w.Write(nonce); err != nil {
		return fmt.Errorf("failed to write EOF nonce: %w", err)
	}

	eofLen := make([]byte, lengthSize)
	// length is already zero (zero-value)
	if _, err := w.Write(eofLen); err != nil {
		return fmt.Errorf("failed to write EOF length: %w", err)
	}

	return nil
}

// StreamDecrypt reads STIM v2 chunked AEAD encrypted data from r and writes
// the decrypted plaintext to w. Returns an error if the header is invalid,
// the password is wrong, or data has been tampered with.
func StreamDecrypt(r io.Reader, w io.Writer, password string) error {
	// Read header
	header := make([]byte, headerSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	// Validate magic
	if header[0] != stimMagic[0] || header[1] != stimMagic[1] ||
		header[2] != stimMagic[2] || header[3] != stimMagic[3] {
		return ErrInvalidMagic
	}

	// Validate version
	if header[4] != 2 {
		return fmt.Errorf("%w: got %d", ErrUnsupportedVersion, header[4])
	}

	// Extract salt and params
	salt := header[5:21]
	params := borgtrix.DecodeArgon2Params(header[21:33])

	// Derive key using stored params
	key := deriveKeyWithParams(password, salt, params)

	// Create AEAD cipher
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return fmt.Errorf("failed to create AEAD: %w", err)
	}

	// Decrypt blocks
	nonce := make([]byte, nonceSize)
	lenBuf := make([]byte, lengthSize)

	for {
		// Read nonce
		if _, err := io.ReadFull(r, nonce); err != nil {
			return fmt.Errorf("failed to read block nonce: %w", err)
		}

		// Read length
		if _, err := io.ReadFull(r, lenBuf); err != nil {
			return fmt.Errorf("failed to read block length: %w", err)
		}

		ctLen := binary.LittleEndian.Uint32(lenBuf)

		// EOF marker: length == 0
		if ctLen == 0 {
			return nil
		}

		// Read ciphertext
		ciphertext := make([]byte, ctLen)
		if _, err := io.ReadFull(r, ciphertext); err != nil {
			return fmt.Errorf("failed to read ciphertext: %w", err)
		}

		// Decrypt and authenticate
		plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrStreamDecrypt, err)
		}

		if _, err := w.Write(plaintext); err != nil {
			return fmt.Errorf("failed to write plaintext: %w", err)
		}
	}
}

// deriveKeyWithParams derives a 32-byte key using Argon2id with specific
// parameters read from the STIM header (rather than using defaults).
func deriveKeyWithParams(password string, salt []byte, params borgtrix.Argon2Params) []byte {
	return argon2.IDKey([]byte(password), salt, params.Time, params.Memory, uint8(params.Threads), 32)
}
