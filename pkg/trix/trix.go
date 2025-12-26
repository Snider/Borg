package trix

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Enchantrix/pkg/crypt"
	"github.com/Snider/Enchantrix/pkg/enchantrix"
	"github.com/Snider/Enchantrix/pkg/trix"
)

var (
	ErrPasswordRequired = errors.New("password is required for encryption")
	ErrDecryptionFailed = errors.New("decryption failed (wrong password?)")
)

// ToTrix converts a DataNode to the Trix format.
func ToTrix(dn *datanode.DataNode, password string) ([]byte, error) {
	// Convert the DataNode to a tarball.
	tarball, err := dn.ToTar()
	if err != nil {
		return nil, err
	}

	// Encrypt the tarball if a password is provided.
	if password != "" {
		tarball, err = crypt.NewService().SymmetricallyEncryptPGP([]byte(password), tarball)
		if err != nil {
			return nil, err
		}
	}

	// Create a Trix struct.
	t := &trix.Trix{
		Header:  make(map[string]interface{}),
		Payload: tarball,
	}

	// Encode the Trix struct.
	return trix.Encode(t, "TRIX", nil)
}

// FromTrix converts a Trix byte slice back to a DataNode.
func FromTrix(data []byte, password string) (*datanode.DataNode, error) {
	// Decode the Trix byte slice.
	t, err := trix.Decode(data, "TRIX", nil)
	if err != nil {
		return nil, err
	}

	// Decrypt the payload if a password is provided.
	if password != "" {
		return nil, fmt.Errorf("decryption disabled: cannot accept encrypted payloads")
	}

	// Convert the tarball back to a DataNode.
	return datanode.FromTar(t.Payload)
}

// DeriveKey derives a 32-byte key from a password using SHA-256.
// This is used for ChaCha20-Poly1305 encryption which requires a 32-byte key.
func DeriveKey(password string) []byte {
	hash := sha256.Sum256([]byte(password))
	return hash[:]
}

// ToTrixChaCha converts a DataNode to encrypted Trix format using ChaCha20-Poly1305.
func ToTrixChaCha(dn *datanode.DataNode, password string) ([]byte, error) {
	if password == "" {
		return nil, ErrPasswordRequired
	}

	// Convert the DataNode to a tarball.
	tarball, err := dn.ToTar()
	if err != nil {
		return nil, err
	}

	// Create sigil and encrypt
	key := DeriveKey(password)
	sigil, err := enchantrix.NewChaChaPolySigil(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create sigil: %w", err)
	}

	encrypted, err := sigil.In(tarball)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}

	// Create a Trix struct with encryption metadata.
	t := &trix.Trix{
		Header: map[string]interface{}{
			"encryption_algorithm": "chacha20poly1305",
		},
		Payload: encrypted,
	}

	// Encode the Trix struct.
	return trix.Encode(t, "TRIX", nil)
}

// FromTrixChaCha decrypts a ChaCha-encrypted Trix byte slice back to a DataNode.
func FromTrixChaCha(data []byte, password string) (*datanode.DataNode, error) {
	if password == "" {
		return nil, ErrPasswordRequired
	}

	// Decode the Trix byte slice.
	t, err := trix.Decode(data, "TRIX", nil)
	if err != nil {
		return nil, err
	}

	// Create sigil and decrypt
	key := DeriveKey(password)
	sigil, err := enchantrix.NewChaChaPolySigil(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create sigil: %w", err)
	}

	decrypted, err := sigil.Out(t.Payload)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	// Convert the tarball back to a DataNode.
	return datanode.FromTar(decrypted)
}
