package stmf

import (
	"crypto/ecdh"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/Snider/Enchantrix/pkg/enchantrix"
	"github.com/Snider/Enchantrix/pkg/trix"
)

// Decrypt decrypts a STMF payload using the server's private key.
// It extracts the ephemeral public key from the header, performs ECDH,
// and decrypts with ChaCha20-Poly1305.
func Decrypt(stmfData []byte, serverPrivateKey []byte) (*FormData, error) {
	// Load server's private key
	serverPriv, err := LoadPrivateKey(serverPrivateKey)
	if err != nil {
		return nil, err
	}

	return DecryptWithKey(stmfData, serverPriv)
}

// DecryptBase64 decrypts a base64-encoded STMF payload
func DecryptBase64(encoded string, serverPrivateKey []byte) (*FormData, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base64: %v", ErrInvalidPayload, err)
	}
	return Decrypt(data, serverPrivateKey)
}

// DecryptWithKey decrypts a STMF payload using a pre-loaded private key
func DecryptWithKey(stmfData []byte, serverPrivateKey *ecdh.PrivateKey) (*FormData, error) {
	// Decode the trix container
	t, err := trix.Decode(stmfData, Magic, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidMagic, err)
	}

	// Extract ephemeral public key from header
	ephemeralPKBase64, ok := t.Header["ephemeral_pk"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: missing ephemeral_pk in header", ErrInvalidPayload)
	}

	ephemeralPKBytes, err := base64.StdEncoding.DecodeString(ephemeralPKBase64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid ephemeral_pk base64: %v", ErrInvalidPayload, err)
	}

	// Load ephemeral public key
	ephemeralPub, err := LoadPublicKey(ephemeralPKBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid ephemeral public key: %v", ErrInvalidPayload, err)
	}

	// Perform ECDH key exchange (server private * ephemeral public = shared secret)
	sharedSecret, err := serverPrivateKey.ECDH(ephemeralPub)
	if err != nil {
		return nil, fmt.Errorf("ECDH failed: %w", err)
	}

	// Derive symmetric key using SHA-256 (same as encryption)
	symmetricKey := sha256.Sum256(sharedSecret)

	// Create ChaCha20-Poly1305 sigil
	sigil, err := enchantrix.NewChaChaPolySigil(symmetricKey[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create sigil: %w", err)
	}

	// Decrypt the payload
	decrypted, err := sigil.Out(t.Payload)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	// Unmarshal form data
	var formData FormData
	if err := json.Unmarshal(decrypted, &formData); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON payload: %v", ErrInvalidPayload, err)
	}

	return &formData, nil
}

// DecryptToMap is a convenience function that returns the form data as a simple map
func DecryptToMap(stmfData []byte, serverPrivateKey []byte) (map[string]string, error) {
	formData, err := Decrypt(stmfData, serverPrivateKey)
	if err != nil {
		return nil, err
	}
	return formData.ToMap(), nil
}

// DecryptBase64ToMap decrypts base64 and returns a map
func DecryptBase64ToMap(encoded string, serverPrivateKey []byte) (map[string]string, error) {
	formData, err := DecryptBase64(encoded, serverPrivateKey)
	if err != nil {
		return nil, err
	}
	return formData.ToMap(), nil
}

// ValidatePayload checks if the data is a valid STMF container without decrypting
func ValidatePayload(stmfData []byte) error {
	t, err := trix.Decode(stmfData, Magic, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidMagic, err)
	}

	// Check required header fields
	if _, ok := t.Header["ephemeral_pk"].(string); !ok {
		return fmt.Errorf("%w: missing ephemeral_pk", ErrInvalidPayload)
	}

	if _, ok := t.Header["algorithm"].(string); !ok {
		return fmt.Errorf("%w: missing algorithm", ErrInvalidPayload)
	}

	return nil
}

// GetPayloadInfo extracts metadata from a STMF payload without decrypting
func GetPayloadInfo(stmfData []byte) (*Header, error) {
	t, err := trix.Decode(stmfData, Magic, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidMagic, err)
	}

	header := &Header{}

	if v, ok := t.Header["version"].(string); ok {
		header.Version = v
	}
	if v, ok := t.Header["algorithm"].(string); ok {
		header.Algorithm = v
	}
	if v, ok := t.Header["ephemeral_pk"].(string); ok {
		header.EphemeralPK = v
	}
	if v, ok := t.Header["nonce"].(string); ok {
		header.Nonce = v
	}

	return header, nil
}
