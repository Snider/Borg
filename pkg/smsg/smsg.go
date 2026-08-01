package smsg

// SMSG (Secure Message) provides ChaCha20-Poly1305 authenticated encryption.
//
// IMPORTANT: Nonce handling for developers
// =========================================
// Enchantrix embeds the nonce directly in the ciphertext:
//
//     [24-byte nonce][encrypted data][16-byte auth tag]
//
// The nonce is NOT transmitted separately in headers. It is:
//   - Generated fresh (random) for each encryption
//   - Extracted automatically from ciphertext during decryption
//   - Safe to transmit (public) - only the KEY must remain secret
//
// This means wrapped keys, encrypted payloads, etc. are self-contained.
// You only need the correct key to decrypt - no nonce management required.
//
// See: github.com/Snider/Enchantrix/pkg/enchantrix/crypto_sigil.go

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/Snider/Enchantrix/pkg/enchantrix"
	"github.com/Snider/Enchantrix/pkg/trix"
	"github.com/klauspost/compress/zstd"
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
// Automatically handles both v1 (base64) and v2 (binary) formats
func Decrypt(data []byte, password string) (*Message, error) {
	if password == "" {
		return nil, ErrPasswordRequired
	}

	// Decode trix container
	t, err := trix.Decode(data, Magic, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidMagic, err)
	}

	// Extract format and compression from header
	format := ""
	compression := ""
	if f, ok := t.Header["format"].(string); ok {
		format = f
	}
	if c, ok := t.Header["compression"].(string); ok {
		compression = c
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

	// Decompress if needed
	switch compression {
	case CompressionGzip:
		decompressed, err := gzipDecompress(decrypted)
		if err != nil {
			return nil, fmt.Errorf("gzip decompression failed: %w", err)
		}
		decrypted = decompressed
	case CompressionZstd:
		decompressed, err := zstdDecompress(decrypted)
		if err != nil {
			return nil, fmt.Errorf("zstd decompression failed: %w", err)
		}
		decrypted = decompressed
	}

	// Parse based on format
	if format == FormatV2 {
		return parseV2Payload(decrypted)
	}

	// v1 format: plain JSON with base64 attachments
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
	if v, ok := t.Header["format"].(string); ok {
		header.Format = v
	}
	if v, ok := t.Header["compression"].(string); ok {
		header.Compression = v
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

// EncryptV2 encrypts a message using v2 binary format (smaller file size)
// Attachments are stored as raw binary instead of base64-encoded JSON
// Uses zstd compression by default (faster than gzip, better ratio)
func EncryptV2(msg *Message, password string) ([]byte, error) {
	return EncryptV2WithOptions(msg, password, nil, CompressionZstd)
}

// EncryptV2WithManifest encrypts with v2 binary format and public manifest
// Uses zstd compression by default (faster than gzip, better ratio)
func EncryptV2WithManifest(msg *Message, password string, manifest *Manifest) ([]byte, error) {
	return EncryptV2WithOptions(msg, password, manifest, CompressionZstd)
}

// EncryptV2WithOptions encrypts with full control over format options
func EncryptV2WithOptions(msg *Message, password string, manifest *Manifest, compression string) ([]byte, error) {
	if password == "" {
		return nil, ErrPasswordRequired
	}
	if msg.Body == "" && len(msg.Attachments) == 0 {
		return nil, ErrEmptyMessage
	}

	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().Unix()
	}

	// Build v2 payload: [4-byte JSON length][JSON][binary attachments...]
	payload, err := buildV2Payload(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to build v2 payload: %w", err)
	}

	// Apply compression if requested
	switch compression {
	case CompressionGzip:
		compressed, err := gzipCompress(payload)
		if err != nil {
			return nil, fmt.Errorf("gzip compression failed: %w", err)
		}
		payload = compressed
	case CompressionZstd:
		compressed, err := zstdCompress(payload)
		if err != nil {
			return nil, fmt.Errorf("zstd compression failed: %w", err)
		}
		payload = compressed
	}

	// Encrypt
	key := DeriveKey(password)
	sigil, err := enchantrix.NewChaChaPolySigil(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create sigil: %w", err)
	}

	encrypted, err := sigil.In(payload)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	// Build header
	headerMap := map[string]interface{}{
		"version":   Version,
		"algorithm": "chacha20poly1305",
		"format":    FormatV2,
	}
	if compression != CompressionNone {
		headerMap["compression"] = compression
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

// buildV2Payload creates the v2 binary payload structure
func buildV2Payload(msg *Message) ([]byte, error) {
	// Create a copy of the message with attachment content stripped
	// We'll append the binary data after the JSON
	msgCopy := *msg
	var binaryData [][]byte

	for i := range msgCopy.Attachments {
		att := &msgCopy.Attachments[i]
		if att.Content != "" {
			// Decode the base64 content to get binary
			data, err := base64.StdEncoding.DecodeString(att.Content)
			if err != nil {
				return nil, fmt.Errorf("invalid base64 in attachment %s: %w", att.Name, err)
			}
			binaryData = append(binaryData, data)
			att.Size = len(data) // Store actual binary size
			att.Content = ""     // Clear content from JSON
		} else {
			binaryData = append(binaryData, nil)
		}
	}

	// Serialize the message (without attachment content)
	jsonData, err := json.Marshal(&msgCopy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	// Build payload: [4-byte length][JSON][binary1][binary2]...
	var buf bytes.Buffer

	// Write JSON length as uint32 big-endian
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(jsonData))); err != nil {
		return nil, err
	}

	// Write JSON
	buf.Write(jsonData)

	// Write binary attachments
	for _, data := range binaryData {
		buf.Write(data)
	}

	return buf.Bytes(), nil
}

// parseV2Payload extracts message and binary attachments from v2 format
func parseV2Payload(data []byte) (*Message, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("payload too short")
	}

	// Read JSON length
	jsonLen := binary.BigEndian.Uint32(data[:4])
	if int(jsonLen) > len(data)-4 {
		return nil, fmt.Errorf("invalid JSON length")
	}

	// Parse JSON
	var msg Message
	if err := json.Unmarshal(data[4:4+jsonLen], &msg); err != nil {
		return nil, fmt.Errorf("failed to parse message JSON: %w", err)
	}

	// Read binary attachments
	offset := 4 + int(jsonLen)
	for i := range msg.Attachments {
		att := &msg.Attachments[i]
		if att.Size > 0 {
			if offset+att.Size > len(data) {
				return nil, fmt.Errorf("attachment %s: data truncated", att.Name)
			}
			// Re-encode as base64 for API compatibility
			att.Content = base64.StdEncoding.EncodeToString(data[offset : offset+att.Size])
			offset += att.Size
		}
	}

	return &msg, nil
}

// gzipCompress compresses data using gzip
func gzipCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// gzipDecompress decompresses gzip data
func gzipDecompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

// zstdCompress compresses data using zstd (faster than gzip, better ratio)
func zstdCompress(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, err
	}
	defer encoder.Close()
	return encoder.EncodeAll(data, nil), nil
}

// zstdDecompress decompresses zstd data
func zstdDecompress(data []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}
	defer decoder.Close()
	return decoder.DecodeAll(data, nil)
}
