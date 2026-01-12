package smsg

// V3 Streaming Support with LTHN Rolling Keys
//
// This file implements zero-trust streaming where:
//   - Content is encrypted once with a random CEK (Content Encryption Key)
//   - CEK is wrapped (encrypted) with time-bound stream keys
//   - Stream keys are derived using LTHN(date:license:fingerprint)
//   - Rolling window: today and tomorrow keys are valid (24-48hr window)
//   - Keys auto-expire - no revocation needed
//
// Server flow:
//   1. Generate random CEK
//   2. Encrypt content with CEK
//   3. For today & tomorrow: wrap CEK with DeriveStreamKey(date, license, fingerprint)
//   4. Store wrapped keys in header
//
// Client flow:
//   1. Derive stream key for today (or tomorrow)
//   2. Try to unwrap CEK from header
//   3. Decrypt content with CEK

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Snider/Enchantrix/pkg/crypt"
	"github.com/Snider/Enchantrix/pkg/enchantrix"
	"github.com/Snider/Enchantrix/pkg/trix"
)

// StreamParams contains the parameters needed for stream key derivation
type StreamParams struct {
	License     string  // User's license identifier
	Fingerprint string  // Device/session fingerprint
	Cadence     Cadence // Key rotation cadence (default: daily)
}

// DeriveStreamKey derives a 32-byte ChaCha key from date, license, and fingerprint.
// Uses LTHN hash which is rainbow-table resistant (salt derived from input itself).
//
// The derived key is: SHA256(LTHN("YYYY-MM-DD:license:fingerprint"))
func DeriveStreamKey(date, license, fingerprint string) []byte {
	// Build input string
	input := fmt.Sprintf("%s:%s:%s", date, license, fingerprint)

	// Use Enchantrix crypt service for LTHN hash
	cryptService := crypt.NewService()
	lthnHash := cryptService.Hash(crypt.LTHN, input)

	// LTHN returns hex string, hash it again to get 32 bytes for ChaCha
	key := sha256.Sum256([]byte(lthnHash))
	return key[:]
}

// GetRollingDates returns today and tomorrow's date strings in YYYY-MM-DD format
// This is the default daily cadence.
func GetRollingDates() (current, next string) {
	return GetRollingPeriods(CadenceDaily, time.Now().UTC())
}

// GetRollingDatesAt returns today and tomorrow relative to a specific time
func GetRollingDatesAt(t time.Time) (current, next string) {
	return GetRollingPeriods(CadenceDaily, t.UTC())
}

// GetRollingPeriods returns the current and next period strings based on cadence.
// The period string format varies by cadence:
//   - daily:  "2006-01-02"
//   - 12h:    "2006-01-02-AM" or "2006-01-02-PM"
//   - 6h:     "2006-01-02-00", "2006-01-02-06", "2006-01-02-12", "2006-01-02-18"
//   - 1h:     "2006-01-02-15" (hour in 24h format)
func GetRollingPeriods(cadence Cadence, t time.Time) (current, next string) {
	t = t.UTC()

	switch cadence {
	case CadenceHalfDay:
		// 12-hour periods: AM (00:00-11:59) and PM (12:00-23:59)
		date := t.Format("2006-01-02")
		if t.Hour() < 12 {
			current = date + "-AM"
			next = date + "-PM"
		} else {
			current = date + "-PM"
			next = t.AddDate(0, 0, 1).Format("2006-01-02") + "-AM"
		}

	case CadenceQuarter:
		// 6-hour periods: 00, 06, 12, 18
		date := t.Format("2006-01-02")
		hour := t.Hour()
		period := (hour / 6) * 6
		nextPeriod := period + 6

		current = fmt.Sprintf("%s-%02d", date, period)
		if nextPeriod >= 24 {
			next = fmt.Sprintf("%s-%02d", t.AddDate(0, 0, 1).Format("2006-01-02"), 0)
		} else {
			next = fmt.Sprintf("%s-%02d", date, nextPeriod)
		}

	case CadenceHourly:
		// Hourly periods
		current = t.Format("2006-01-02-15")
		next = t.Add(time.Hour).Format("2006-01-02-15")

	default: // CadenceDaily or empty
		current = t.Format("2006-01-02")
		next = t.AddDate(0, 0, 1).Format("2006-01-02")
	}

	return
}

// GetCadenceWindowDuration returns the duration of one period for a cadence
func GetCadenceWindowDuration(cadence Cadence) time.Duration {
	switch cadence {
	case CadenceHourly:
		return time.Hour
	case CadenceQuarter:
		return 6 * time.Hour
	case CadenceHalfDay:
		return 12 * time.Hour
	default: // CadenceDaily
		return 24 * time.Hour
	}
}

// WrapCEK wraps a Content Encryption Key with a stream key
// Returns base64-encoded wrapped key (includes nonce)
func WrapCEK(cek, streamKey []byte) (string, error) {
	sigil, err := enchantrix.NewChaChaPolySigil(streamKey)
	if err != nil {
		return "", fmt.Errorf("failed to create sigil: %w", err)
	}

	wrapped, err := sigil.In(cek)
	if err != nil {
		return "", fmt.Errorf("failed to wrap CEK: %w", err)
	}

	return base64.StdEncoding.EncodeToString(wrapped), nil
}

// UnwrapCEK unwraps a Content Encryption Key using a stream key
// Takes base64-encoded wrapped key, returns raw CEK bytes
func UnwrapCEK(wrappedB64 string, streamKey []byte) ([]byte, error) {
	wrapped, err := base64.StdEncoding.DecodeString(wrappedB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode wrapped key: %w", err)
	}

	sigil, err := enchantrix.NewChaChaPolySigil(streamKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create sigil: %w", err)
	}

	cek, err := sigil.Out(wrapped)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return cek, nil
}

// GenerateCEK generates a random 32-byte Content Encryption Key
func GenerateCEK() ([]byte, error) {
	cek := make([]byte, 32)
	if _, err := rand.Read(cek); err != nil {
		return nil, fmt.Errorf("failed to generate CEK: %w", err)
	}
	return cek, nil
}

// EncryptV3 encrypts a message using v3 streaming format with rolling keys.
// The content is encrypted with a random CEK, which is then wrapped with
// stream keys for today and tomorrow.
func EncryptV3(msg *Message, params *StreamParams, manifest *Manifest) ([]byte, error) {
	if params == nil || params.License == "" {
		return nil, ErrLicenseRequired
	}
	if msg.Body == "" && len(msg.Attachments) == 0 {
		return nil, ErrEmptyMessage
	}

	// Set timestamp if not set
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().Unix()
	}

	// Generate random CEK
	cek, err := GenerateCEK()
	if err != nil {
		return nil, err
	}

	// Determine cadence (default to daily if not specified)
	cadence := params.Cadence
	if cadence == "" {
		cadence = CadenceDaily
	}

	// Get rolling periods based on cadence
	current, next := GetRollingPeriods(cadence, time.Now().UTC())

	// Wrap CEK with current period's stream key
	currentKey := DeriveStreamKey(current, params.License, params.Fingerprint)
	wrappedCurrent, err := WrapCEK(cek, currentKey)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap CEK for current period: %w", err)
	}

	// Wrap CEK with next period's stream key
	nextKey := DeriveStreamKey(next, params.License, params.Fingerprint)
	wrappedNext, err := WrapCEK(cek, nextKey)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap CEK for next period: %w", err)
	}

	// Build v3 payload (similar to v2 but encrypted with CEK)
	payload, attachmentData, err := buildV3Payload(msg)
	if err != nil {
		return nil, err
	}

	// Compress payload
	compressed, err := zstdCompress(payload)
	if err != nil {
		return nil, fmt.Errorf("compression failed: %w", err)
	}

	// Encrypt with CEK
	sigil, err := enchantrix.NewChaChaPolySigil(cek)
	if err != nil {
		return nil, fmt.Errorf("failed to create sigil: %w", err)
	}

	encrypted, err := sigil.In(compressed)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	// Encrypt attachment data with CEK
	encryptedAttachments, err := sigil.In(attachmentData)
	if err != nil {
		return nil, fmt.Errorf("attachment encryption failed: %w", err)
	}

	// Create header with wrapped keys
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

	// Build v3 binary format: [4-byte json len][json header][encrypted payload][encrypted attachments]
	headerJSON, err := json.Marshal(headerMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal header: %w", err)
	}

	// Calculate total size
	totalSize := 4 + len(headerJSON) + 4 + len(encrypted) + len(encryptedAttachments)
	output := make([]byte, 0, totalSize)

	// Write header length (4 bytes, big-endian)
	headerLen := make([]byte, 4)
	binary.BigEndian.PutUint32(headerLen, uint32(len(headerJSON)))
	output = append(output, headerLen...)

	// Write header JSON
	output = append(output, headerJSON...)

	// Write encrypted payload length (4 bytes, big-endian)
	payloadLen := make([]byte, 4)
	binary.BigEndian.PutUint32(payloadLen, uint32(len(encrypted)))
	output = append(output, payloadLen...)

	// Write encrypted payload
	output = append(output, encrypted...)

	// Write encrypted attachments
	output = append(output, encryptedAttachments...)

	// Wrap in trix container
	t := &trix.Trix{
		Header:  headerMap,
		Payload: output,
	}

	return trix.Encode(t, Magic, nil)
}

// DecryptV3 decrypts a v3 streaming message using rolling keys.
// It tries today's key first, then tomorrow's key.
func DecryptV3(data []byte, params *StreamParams) (*Message, *Header, error) {
	if params == nil || params.License == "" {
		return nil, nil, ErrLicenseRequired
	}

	// Decode trix container
	t, err := trix.Decode(data, Magic, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode container: %w", err)
	}

	// Parse header
	headerJSON, err := json.Marshal(t.Header)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal header: %w", err)
	}

	var header Header
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, nil, fmt.Errorf("failed to parse header: %w", err)
	}

	// Verify v3 format
	if header.Format != FormatV3 {
		return nil, nil, fmt.Errorf("expected v3 format, got: %s", header.Format)
	}

	if header.KeyMethod != KeyMethodLTHNRolling {
		return nil, nil, fmt.Errorf("unsupported key method: %s", header.KeyMethod)
	}

	// Determine cadence from header (or use params, or default to daily)
	cadence := header.Cadence
	if cadence == "" && params.Cadence != "" {
		cadence = params.Cadence
	}
	if cadence == "" {
		cadence = CadenceDaily
	}

	// Try to unwrap CEK with rolling keys
	cek, err := tryUnwrapCEK(header.WrappedKeys, params, cadence)
	if err != nil {
		return nil, &header, err
	}

	// Parse v3 binary payload
	payload := t.Payload
	if len(payload) < 8 {
		return nil, &header, ErrInvalidPayload
	}

	// Read header length (skip - we already parsed from trix header)
	headerLen := binary.BigEndian.Uint32(payload[:4])
	pos := 4 + int(headerLen)

	if len(payload) < pos+4 {
		return nil, &header, ErrInvalidPayload
	}

	// Read encrypted payload length
	encryptedLen := binary.BigEndian.Uint32(payload[pos : pos+4])
	pos += 4

	if len(payload) < pos+int(encryptedLen) {
		return nil, &header, ErrInvalidPayload
	}

	// Extract encrypted payload and attachments
	encryptedPayload := payload[pos : pos+int(encryptedLen)]
	encryptedAttachments := payload[pos+int(encryptedLen):]

	// Decrypt with CEK
	sigil, err := enchantrix.NewChaChaPolySigil(cek)
	if err != nil {
		return nil, &header, fmt.Errorf("failed to create sigil: %w", err)
	}

	compressed, err := sigil.Out(encryptedPayload)
	if err != nil {
		return nil, &header, ErrDecryptionFailed
	}

	// Decompress
	var decompressed []byte
	if header.Compression == CompressionZstd {
		decompressed, err = zstdDecompress(compressed)
		if err != nil {
			return nil, &header, fmt.Errorf("decompression failed: %w", err)
		}
	} else {
		decompressed = compressed
	}

	// Parse message
	var msg Message
	if err := json.Unmarshal(decompressed, &msg); err != nil {
		return nil, &header, fmt.Errorf("failed to parse message: %w", err)
	}

	// Decrypt attachments if present
	if len(encryptedAttachments) > 0 {
		attachmentData, err := sigil.Out(encryptedAttachments)
		if err != nil {
			return nil, &header, fmt.Errorf("attachment decryption failed: %w", err)
		}

		// Restore attachment content from binary data
		if err := restoreV3Attachments(&msg, attachmentData); err != nil {
			return nil, &header, err
		}
	}

	return &msg, &header, nil
}

// tryUnwrapCEK attempts to unwrap the CEK using current or next period's key
func tryUnwrapCEK(wrappedKeys []WrappedKey, params *StreamParams, cadence Cadence) ([]byte, error) {
	current, next := GetRollingPeriods(cadence, time.Now().UTC())

	// Build map of available wrapped keys by period
	keysByPeriod := make(map[string]string)
	for _, wk := range wrappedKeys {
		keysByPeriod[wk.Date] = wk.Wrapped
	}

	// Try current period's key first
	if wrapped, ok := keysByPeriod[current]; ok {
		streamKey := DeriveStreamKey(current, params.License, params.Fingerprint)
		if cek, err := UnwrapCEK(wrapped, streamKey); err == nil {
			return cek, nil
		}
	}

	// Try next period's key
	if wrapped, ok := keysByPeriod[next]; ok {
		streamKey := DeriveStreamKey(next, params.License, params.Fingerprint)
		if cek, err := UnwrapCEK(wrapped, streamKey); err == nil {
			return cek, nil
		}
	}

	return nil, ErrNoValidKey
}

// buildV3Payload builds the message JSON and binary attachment data
func buildV3Payload(msg *Message) ([]byte, []byte, error) {
	// Create a copy of the message without attachment content
	msgCopy := *msg
	var attachmentData []byte

	for i := range msgCopy.Attachments {
		att := &msgCopy.Attachments[i]
		if att.Content != "" {
			// Decode base64 content to binary
			data, err := base64.StdEncoding.DecodeString(att.Content)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to decode attachment %s: %w", att.Name, err)
			}
			attachmentData = append(attachmentData, data...)
			att.Content = "" // Clear content, will be restored on decrypt
		}
	}

	// Marshal message (without attachment content)
	payload, err := json.Marshal(&msgCopy)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	return payload, attachmentData, nil
}

// restoreV3Attachments restores attachment content from decrypted binary data
func restoreV3Attachments(msg *Message, data []byte) error {
	offset := 0
	for i := range msg.Attachments {
		att := &msg.Attachments[i]
		if att.Size > 0 {
			if offset+att.Size > len(data) {
				return fmt.Errorf("attachment data truncated for %s", att.Name)
			}
			att.Content = base64.StdEncoding.EncodeToString(data[offset : offset+att.Size])
			offset += att.Size
		}
	}
	return nil
}
