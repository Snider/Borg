package smsg

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Snider/Enchantrix/pkg/enchantrix"
	"github.com/Snider/Enchantrix/pkg/keyserver"
	"github.com/Snider/Enchantrix/pkg/trix"
)

// EncryptKS encrypts a message using the keyserver (V1 format).
// The key material never leaves the keyserver — only the key ID is passed.
func EncryptKS(ctx context.Context, msg *Message, keyID string, ks keyserver.KeyServer) ([]byte, error) {
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

	// Keyserver encrypts — key never leaves
	encrypted, err := ks.EncryptSMSG(ctx, keyID, payload)
	if err != nil {
		return nil, fmt.Errorf("keyserver encrypt: %w", err)
	}

	t := &trix.Trix{
		Header: map[string]interface{}{
			"version":   Version,
			"algorithm": "chacha20poly1305",
		},
		Payload: encrypted,
	}

	return trix.Encode(t, Magic, nil)
}

// DecryptKS decrypts an SMSG container using the keyserver.
// Handles both V1 and V2 formats. The key material never leaves the keyserver.
func DecryptKS(ctx context.Context, data []byte, keyID string, ks keyserver.KeyServer) (*Message, error) {
	t, err := trix.Decode(data, Magic, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidMagic, err)
	}

	// Keyserver decrypts — key never leaves
	decrypted, err := ks.DecryptSMSG(ctx, keyID, t.Payload)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	format := ""
	compression := ""
	if f, ok := t.Header["format"].(string); ok {
		format = f
	}
	if c, ok := t.Header["compression"].(string); ok {
		compression = c
	}

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

	if format == FormatV2 {
		return parseV2Payload(decrypted)
	}

	var msg Message
	if err := json.Unmarshal(decrypted, &msg); err != nil {
		return nil, fmt.Errorf("%w: invalid message format", ErrInvalidPayload)
	}

	return &msg, nil
}

// V3ParamsKS contains keyserver-aware parameters for V3 streaming.
type V3ParamsKS struct {
	License     string
	Fingerprint string
	Cadence     Cadence
}

// EncryptV3KS encrypts using V3 streaming format. Stream keys are derived and
// CEK wrapping is done via keyserver — stream key material never leaves.
// The CEK itself is a random per-message key used for content encryption.
func EncryptV3KS(ctx context.Context, msg *Message, params V3ParamsKS, manifest *Manifest, ks keyserver.KeyServer) ([]byte, error) {
	if params.License == "" {
		return nil, ErrLicenseRequired
	}
	if msg.Body == "" && len(msg.Attachments) == 0 {
		return nil, ErrEmptyMessage
	}

	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().Unix()
	}

	cadence := params.Cadence
	if cadence == "" {
		cadence = CadenceDaily
	}

	current, next := GetRollingPeriods(cadence, time.Now().UTC())

	// Generate random CEK (this is a per-message ephemeral key)
	cek, err := GenerateCEK()
	if err != nil {
		return nil, err
	}

	// Derive stream keys via keyserver — stream key material never leaves
	currentKeyID, err := ks.DeriveStreamKey(ctx, params.License, current, params.Fingerprint)
	if err != nil {
		return nil, fmt.Errorf("failed to derive current stream key: %w", err)
	}
	defer ks.DeleteKey(ctx, currentKeyID)

	nextKeyID, err := ks.DeriveStreamKey(ctx, params.License, next, params.Fingerprint)
	if err != nil {
		return nil, fmt.Errorf("failed to derive next stream key: %w", err)
	}
	defer ks.DeleteKey(ctx, nextKeyID)

	// Wrap CEK with stream keys via keyserver
	wrappedCurrent, err := ks.WrapCEK(ctx, currentKeyID, cek)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap CEK for current period: %w", err)
	}

	wrappedNext, err := ks.WrapCEK(ctx, nextKeyID, cek)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap CEK for next period: %w", err)
	}

	// Encrypt content with CEK
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	compressed, err := zstdCompress(payload)
	if err != nil {
		return nil, fmt.Errorf("compression failed: %w", err)
	}

	sigil, err := enchantrix.NewChaChaPolySigil(cek)
	if err != nil {
		return nil, fmt.Errorf("failed to create sigil: %w", err)
	}

	encrypted, err := sigil.In(compressed)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	// Build V3 header
	headerMap := map[string]interface{}{
		"version":     Version,
		"algorithm":   "chacha20poly1305",
		"format":      FormatV3,
		"compression": CompressionZstd,
		"keyMethod":   KeyMethodLTHNRolling,
		"cadence":     string(cadence),
		"wrappedKeys": []WrappedKey{
			{Date: current, Wrapped: wrappedCurrent},
			{Date: next, Wrapped: wrappedNext},
		},
	}

	if manifest != nil {
		if manifest.IssuedAt == 0 {
			manifest.IssuedAt = time.Now().Unix()
		}
		headerMap["manifest"] = manifest
	}

	t := &trix.Trix{
		Header:  headerMap,
		Payload: encrypted,
	}

	return trix.Encode(t, Magic, nil)
}

// DecryptV3KS decrypts a V3 streaming message. Stream keys are derived via
// keyserver for CEK unwrapping — stream key material never leaves.
func DecryptV3KS(ctx context.Context, data []byte, params V3ParamsKS, ks keyserver.KeyServer) (*Message, *Header, error) {
	if params.License == "" {
		return nil, nil, ErrLicenseRequired
	}

	t, err := trix.Decode(data, Magic, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode container: %w", err)
	}

	headerJSON, err := json.Marshal(t.Header)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal header: %w", err)
	}

	var header Header
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, nil, fmt.Errorf("failed to parse header: %w", err)
	}

	if header.Format != FormatV3 {
		return nil, nil, fmt.Errorf("expected v3 format, got: %s", header.Format)
	}

	cadence := header.Cadence
	if cadence == "" && params.Cadence != "" {
		cadence = params.Cadence
	}
	if cadence == "" {
		cadence = CadenceDaily
	}

	// Unwrap CEK using keyserver-derived stream keys
	cek, err := tryUnwrapCEKKS(ctx, header.WrappedKeys, params, cadence, ks)
	if err != nil {
		return nil, &header, err
	}

	// Decrypt payload with CEK
	sigil, err := enchantrix.NewChaChaPolySigil(cek)
	if err != nil {
		return nil, &header, fmt.Errorf("failed to create sigil: %w", err)
	}

	compressed, err := sigil.Out(t.Payload)
	if err != nil {
		return nil, &header, ErrDecryptionFailed
	}

	var decompressed []byte
	if header.Compression == CompressionZstd {
		decompressed, err = zstdDecompress(compressed)
		if err != nil {
			return nil, &header, fmt.Errorf("decompression failed: %w", err)
		}
	} else {
		decompressed = compressed
	}

	var msg Message
	if err := json.Unmarshal(decompressed, &msg); err != nil {
		return nil, &header, fmt.Errorf("failed to parse message: %w", err)
	}

	return &msg, &header, nil
}

// tryUnwrapCEKKS tries to unwrap CEK using keyserver-derived stream keys.
func tryUnwrapCEKKS(ctx context.Context, wrappedKeys []WrappedKey, params V3ParamsKS, cadence Cadence, ks keyserver.KeyServer) ([]byte, error) {
	current, next := GetRollingPeriods(cadence, time.Now().UTC())

	keysByPeriod := make(map[string]string)
	for _, wk := range wrappedKeys {
		keysByPeriod[wk.Date] = wk.Wrapped
	}

	// Try current period
	if wrapped, ok := keysByPeriod[current]; ok {
		streamKeyID, err := ks.DeriveStreamKey(ctx, params.License, current, params.Fingerprint)
		if err == nil {
			cek, unwrapErr := ks.UnwrapCEK(ctx, streamKeyID, wrapped)
			ks.DeleteKey(ctx, streamKeyID)
			if unwrapErr == nil {
				return cek, nil
			}
		}
	}

	// Try next period
	if wrapped, ok := keysByPeriod[next]; ok {
		streamKeyID, err := ks.DeriveStreamKey(ctx, params.License, next, params.Fingerprint)
		if err == nil {
			cek, unwrapErr := ks.UnwrapCEK(ctx, streamKeyID, wrapped)
			ks.DeleteKey(ctx, streamKeyID)
			if unwrapErr == nil {
				return cek, nil
			}
		}
	}

	return nil, ErrNoValidKey
}
