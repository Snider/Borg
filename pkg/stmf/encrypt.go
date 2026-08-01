package stmf

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/Snider/Enchantrix/pkg/enchantrix"
	"github.com/Snider/Enchantrix/pkg/trix"
)

// Encrypt encrypts form data using the server's public key.
// It performs X25519 ECDH key exchange with an ephemeral keypair,
// derives a symmetric key, and encrypts with ChaCha20-Poly1305.
//
// The result is a STMF container that can be base64-encoded for transmission.
func Encrypt(data *FormData, serverPublicKey []byte) ([]byte, error) {
	// Load server's public key
	serverPub, err := LoadPublicKey(serverPublicKey)
	if err != nil {
		return nil, err
	}

	return EncryptWithKey(data, serverPub)
}

// EncryptBase64 encrypts form data and returns a base64-encoded string
func EncryptBase64(data *FormData, serverPublicKey []byte) (string, error) {
	encrypted, err := Encrypt(data, serverPublicKey)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// EncryptWithKey encrypts form data using a pre-loaded public key
func EncryptWithKey(data *FormData, serverPublicKey *ecdh.PublicKey) ([]byte, error) {
	// Generate ephemeral keypair for this encryption
	ephemeralPrivate, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ephemeral key: %w", err)
	}
	ephemeralPublic := ephemeralPrivate.PublicKey()

	// Perform ECDH key exchange
	sharedSecret, err := ephemeralPrivate.ECDH(serverPublicKey)
	if err != nil {
		return nil, fmt.Errorf("ECDH failed: %w", err)
	}

	// Derive symmetric key using SHA-256 (same pattern as pkg/trix)
	symmetricKey := sha256.Sum256(sharedSecret)

	// Create ChaCha20-Poly1305 sigil
	sigil, err := enchantrix.NewChaChaPolySigil(symmetricKey[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create sigil: %w", err)
	}

	// Serialize form data to JSON
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal form data: %w", err)
	}

	// Encrypt the payload
	encrypted, err := sigil.In(payload)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	// Build STMF container
	// The nonce is included in the encrypted data by ChaChaPolySigil,
	// but we include the ephemeral public key in the header
	header := Header{
		Version:     Version,
		Algorithm:   "x25519-chacha20poly1305",
		EphemeralPK: base64.StdEncoding.EncodeToString(ephemeralPublic.Bytes()),
		Nonce:       "", // Nonce is embedded in ciphertext by Enchantrix
	}

	// Convert header to map for trix
	headerMap := map[string]interface{}{
		"version":      header.Version,
		"algorithm":    header.Algorithm,
		"ephemeral_pk": header.EphemeralPK,
	}

	// Create trix container
	t := &trix.Trix{
		Header:  headerMap,
		Payload: encrypted,
	}

	// Encode with STMF magic
	return trix.Encode(t, Magic, nil)
}

// EncryptMap is a convenience function to encrypt a simple key-value map
func EncryptMap(fields map[string]string, serverPublicKey []byte) ([]byte, error) {
	data := NewFormData()
	for name, value := range fields {
		data.AddField(name, value)
	}
	return Encrypt(data, serverPublicKey)
}

// EncryptMapBase64 encrypts a map and returns base64
func EncryptMapBase64(fields map[string]string, serverPublicKey []byte) (string, error) {
	encrypted, err := EncryptMap(fields, serverPublicKey)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}
