package smsg

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Snider/Enchantrix/pkg/enchantrix"
	"github.com/Snider/Enchantrix/pkg/trix"
)

// DeriveKey derives a 32-byte key from a password using SHA-256.
func DeriveKey(password string) []byte {
	hash := sha256.Sum256([]byte(password))
	return hash[:]
}

// Encrypt encrypts a message with a password.
// Returns the encrypted SMSG container bytes.
func Encrypt(msg *Message, password string) ([]byte, error) {
	if password == "" {
		return nil, ErrPasswordRequired
	}
	if msg.Body == "" && len(msg.Attachments) == 0 {
		return nil, ErrEmptyMessage
	}

	// Set timestamp if not set
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().Unix()
	}

	// Serialize message to JSON
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	// Derive key and create sigil
	key := DeriveKey(password)
	sigil, err := enchantrix.NewChaChaPolySigil(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create sigil: %w", err)
	}

	// Encrypt
	encrypted, err := sigil.In(payload)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	// Create container header
	headerMap := map[string]interface{}{
		"version":   Version,
		"algorithm": "chacha20poly1305",
	}

	// Create trix container
	t := &trix.Trix{
		Header:  headerMap,
		Payload: encrypted,
	}

	return trix.Encode(t, Magic, nil)
}

// EncryptBase64 encrypts and returns base64-encoded result
func EncryptBase64(msg *Message, password string) (string, error) {
	encrypted, err := Encrypt(msg, password)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// EncryptWithHint encrypts with an optional password hint in the header
func EncryptWithHint(msg *Message, password, hint string) ([]byte, error) {
	if password == "" {
		return nil, ErrPasswordRequired
	}
	if msg.Body == "" && len(msg.Attachments) == 0 {
		return nil, ErrEmptyMessage
	}

	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().Unix()
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	key := DeriveKey(password)
	sigil, err := enchantrix.NewChaChaPolySigil(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create sigil: %w", err)
	}

	encrypted, err := sigil.In(payload)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	headerMap := map[string]interface{}{
		"version":   Version,
		"algorithm": "chacha20poly1305",
	}
	if hint != "" {
		headerMap["hint"] = hint
	}

	t := &trix.Trix{
		Header:  headerMap,
		Payload: encrypted,
	}

	return trix.Encode(t, Magic, nil)
}

// EncryptWithManifest encrypts with public manifest metadata in the clear text header
// The manifest is visible without decryption, enabling content discovery and indexing
func EncryptWithManifest(msg *Message, password string, manifest *Manifest) ([]byte, error) {
	if password == "" {
		return nil, ErrPasswordRequired
	}
	if msg.Body == "" && len(msg.Attachments) == 0 {
		return nil, ErrEmptyMessage
	}

	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().Unix()
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	key := DeriveKey(password)
	sigil, err := enchantrix.NewChaChaPolySigil(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create sigil: %w", err)
	}

	encrypted, err := sigil.In(payload)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	// Build header with manifest
	headerMap := map[string]interface{}{
		"version":   Version,
		"algorithm": "chacha20poly1305",
	}
	if manifest != nil {
		headerMap["manifest"] = manifest
	}

	t := &trix.Trix{
		Header:  headerMap,
		Payload: encrypted,
	}

	return trix.Encode(t, Magic, nil)
}

// EncryptWithManifestBase64 encrypts with manifest and returns base64
func EncryptWithManifestBase64(msg *Message, password string, manifest *Manifest) (string, error) {
	encrypted, err := EncryptWithManifest(msg, password, manifest)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// Decrypt decrypts an SMSG container with a password
func Decrypt(data []byte, password string) (*Message, error) {
	if password == "" {
		return nil, ErrPasswordRequired
	}

	// Decode trix container
	t, err := trix.Decode(data, Magic, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidMagic, err)
	}

	// Derive key and create sigil
	key := DeriveKey(password)
	sigil, err := enchantrix.NewChaChaPolySigil(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create sigil: %w", err)
	}

	// Decrypt
	decrypted, err := sigil.Out(t.Payload)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	// Parse message
	var msg Message
	if err := json.Unmarshal(decrypted, &msg); err != nil {
		return nil, fmt.Errorf("%w: invalid message format", ErrInvalidPayload)
	}

	return &msg, nil
}

// DecryptBase64 decrypts a base64-encoded SMSG
func DecryptBase64(encoded, password string) (*Message, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base64", ErrInvalidPayload)
	}
	return Decrypt(data, password)
}

// GetInfo extracts header info without decrypting
func GetInfo(data []byte) (*Header, error) {
	t, err := trix.Decode(data, Magic, nil)
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
	if v, ok := t.Header["hint"].(string); ok {
		header.Hint = v
	}

	// Extract manifest if present
	if manifestData, ok := t.Header["manifest"]; ok && manifestData != nil {
		// Re-marshal and unmarshal to properly convert the map to Manifest struct
		manifestBytes, err := json.Marshal(manifestData)
		if err == nil {
			var manifest Manifest
			if err := json.Unmarshal(manifestBytes, &manifest); err == nil {
				header.Manifest = &manifest
			}
		}
	}

	return header, nil
}

// GetInfoBase64 extracts header info from base64-encoded SMSG
func GetInfoBase64(encoded string) (*Header, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base64", ErrInvalidPayload)
	}
	return GetInfo(data)
}

// Validate checks if data is a valid SMSG container (without decrypting)
func Validate(data []byte) error {
	_, err := trix.Decode(data, Magic, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidMagic, err)
	}
	return nil
}

// QuickEncrypt is a convenience function for simple message encryption
func QuickEncrypt(body, password string) (string, error) {
	msg := NewMessage(body)
	return EncryptBase64(msg, password)
}

// QuickDecrypt is a convenience function for simple message decryption
func QuickDecrypt(encoded, password string) (string, error) {
	msg, err := DecryptBase64(encoded, password)
	if err != nil {
		return "", err
	}
	return msg.Body, nil
}
