// Package smsg implements Secure Message encryption using password-based ChaCha20-Poly1305.
// SMSG (Secure Message) enables encrypted message exchange where the recipient
// decrypts using a pre-shared password. Useful for secure support replies,
// confidential documents, and any scenario requiring password-protected content.
//
// Format versions:
//   - v1: JSON with base64-encoded attachments (legacy)
//   - v2: Binary format with zstd compression (current)
//   - v3: Streaming with LTHN rolling keys (planned)
//
// Encryption note: Nonces are embedded in ciphertext, not transmitted separately.
// See smsg.go header comment for details.
package smsg

import (
	"encoding/base64"
	"errors"
	"time"
)

// Magic bytes for SMSG format
const Magic = "SMSG"

// Version of the SMSG format
const Version = "1.0"

// Errors
var (
	ErrInvalidMagic     = errors.New("invalid SMSG magic")
	ErrInvalidPayload   = errors.New("invalid SMSG payload")
	ErrDecryptionFailed = errors.New("decryption failed (wrong password?)")
	ErrPasswordRequired = errors.New("password is required")
	ErrEmptyMessage     = errors.New("message cannot be empty")
	ErrStreamKeyExpired = errors.New("stream key expired (outside rolling window)")
	ErrNoValidKey       = errors.New("no valid wrapped key found for current date")
	ErrLicenseRequired  = errors.New("license is required for stream decryption")
)

// Attachment represents a file attached to the message
type Attachment struct {
	Name     string `json:"name"`
	Content  string `json:"content,omitempty"` // base64-encoded (v1) or empty (v2, populated on decrypt)
	MimeType string `json:"mime,omitempty"`
	Size     int    `json:"size,omitempty"` // binary size in bytes
}

// PKIInfo contains public key information for authenticated replies
type PKIInfo struct {
	PublicKey   string `json:"public_key"`            // base64-encoded X25519 public key
	KeyID       string `json:"key_id,omitempty"`      // optional key identifier
	Algorithm   string `json:"algorithm,omitempty"`   // e.g., "x25519"
	Fingerprint string `json:"fingerprint,omitempty"` // SHA256 fingerprint of public key
}

// Message represents the decrypted message content
type Message struct {
	// Core message content
	Subject string `json:"subject,omitempty"`
	Body    string `json:"body"`

	// Optional attachments
	Attachments []Attachment `json:"attachments,omitempty"`

	// PKI for authenticated replies
	ReplyKey *PKIInfo `json:"reply_key,omitempty"`

	// Metadata
	From      string            `json:"from,omitempty"`
	Timestamp int64             `json:"timestamp,omitempty"`
	Meta      map[string]string `json:"meta,omitempty"`
}

// NewMessage creates a new message with the given body
func NewMessage(body string) *Message {
	return &Message{
		Body: body,
		Meta: make(map[string]string),
	}
}

// WithSubject sets the message subject
func (m *Message) WithSubject(subject string) *Message {
	m.Subject = subject
	return m
}

// WithFrom sets the sender
func (m *Message) WithFrom(from string) *Message {
	m.From = from
	return m
}

// WithTimestamp sets the timestamp
func (m *Message) WithTimestamp(ts int64) *Message {
	m.Timestamp = ts
	return m
}

// AddAttachment adds a file attachment (content is base64-encoded)
func (m *Message) AddAttachment(name, content, mimeType string) *Message {
	m.Attachments = append(m.Attachments, Attachment{
		Name:     name,
		Content:  content,
		MimeType: mimeType,
		Size:     len(content), // base64 size for v1 compatibility
	})
	return m
}

// AddBinaryAttachment adds a raw binary attachment (for v2 format)
// The content will be base64-encoded for API compatibility
func (m *Message) AddBinaryAttachment(name string, data []byte, mimeType string) *Message {
	m.Attachments = append(m.Attachments, Attachment{
		Name:     name,
		Content:  base64.StdEncoding.EncodeToString(data),
		MimeType: mimeType,
		Size:     len(data), // actual binary size
	})
	return m
}

// WithReplyKey sets the PKI public key for authenticated replies
func (m *Message) WithReplyKey(publicKeyB64 string) *Message {
	m.ReplyKey = &PKIInfo{
		PublicKey: publicKeyB64,
		Algorithm: "x25519",
	}
	return m
}

// WithReplyKeyInfo sets full PKI information
func (m *Message) WithReplyKeyInfo(pki *PKIInfo) *Message {
	m.ReplyKey = pki
	return m
}

// SetMeta sets a metadata value
func (m *Message) SetMeta(key, value string) *Message {
	if m.Meta == nil {
		m.Meta = make(map[string]string)
	}
	m.Meta[key] = value
	return m
}

// GetAttachment finds an attachment by name
func (m *Message) GetAttachment(name string) *Attachment {
	for i := range m.Attachments {
		if m.Attachments[i].Name == name {
			return &m.Attachments[i]
		}
	}
	return nil
}

// Track represents a track marker in a release (like CD chapters)
type Track struct {
	Title    string  `json:"title"`
	Start    float64 `json:"start"`               // start time in seconds
	End      float64 `json:"end,omitempty"`       // end time in seconds (0 = until next track)
	Type     string  `json:"type,omitempty"`      // intro, verse, chorus, drop, outro, etc.
	TrackNum int     `json:"track_num,omitempty"` // track number for multi-track releases
}

// Manifest contains public metadata visible without decryption
// This enables content discovery, indexing, and preview
type Manifest struct {
	// Content identification
	Title  string `json:"title,omitempty"`
	Artist string `json:"artist,omitempty"`
	Album  string `json:"album,omitempty"`
	Genre  string `json:"genre,omitempty"`
	Year   int    `json:"year,omitempty"`

	// Release info
	ReleaseType string `json:"release_type,omitempty"` // single, album, ep, mix
	Duration    int    `json:"duration,omitempty"`     // total duration in seconds
	Format      string `json:"format,omitempty"`       // dapp.fm/v1, etc.

	// License expiration (for streaming/rental models)
	ExpiresAt   int64  `json:"expires_at,omitempty"`   // Unix timestamp when license expires (0 = never)
	IssuedAt    int64  `json:"issued_at,omitempty"`    // Unix timestamp when license was issued
	LicenseType string `json:"license_type,omitempty"` // perpetual, rental, stream, preview

	// Track list (like CD master)
	Tracks []Track `json:"tracks,omitempty"`

	// Artist links - direct to artist, skip the middlemen
	Links map[string]string `json:"links,omitempty"` // platform -> URL (bandcamp, soundcloud, website, etc.)

	// Custom metadata
	Tags  []string          `json:"tags,omitempty"`
	Extra map[string]string `json:"extra,omitempty"`
}

// NewManifest creates a new manifest with title
func NewManifest(title string) *Manifest {
	return &Manifest{
		Title:       title,
		Links:       make(map[string]string),
		Extra:       make(map[string]string),
		LicenseType: "perpetual",
	}
}

// WithExpiration sets the license expiration time
func (m *Manifest) WithExpiration(expiresAt int64) *Manifest {
	m.ExpiresAt = expiresAt
	if m.LicenseType == "perpetual" {
		m.LicenseType = "rental"
	}
	return m
}

// WithRentalDuration sets expiration relative to issue time
func (m *Manifest) WithRentalDuration(durationSeconds int64) *Manifest {
	if m.IssuedAt == 0 {
		m.IssuedAt = time.Now().Unix()
	}
	m.ExpiresAt = m.IssuedAt + durationSeconds
	m.LicenseType = "rental"
	return m
}

// WithStreamingAccess sets up for streaming (short expiration, e.g., 24 hours)
func (m *Manifest) WithStreamingAccess(hours int) *Manifest {
	m.IssuedAt = time.Now().Unix()
	m.ExpiresAt = m.IssuedAt + int64(hours*3600)
	m.LicenseType = "stream"
	return m
}

// WithPreviewAccess sets up for preview (very short, e.g., 30 seconds)
func (m *Manifest) WithPreviewAccess(seconds int) *Manifest {
	m.IssuedAt = time.Now().Unix()
	m.ExpiresAt = m.IssuedAt + int64(seconds)
	m.LicenseType = "preview"
	return m
}

// IsExpired checks if the license has expired
func (m *Manifest) IsExpired() bool {
	if m.ExpiresAt == 0 {
		return false // No expiration = perpetual
	}
	return time.Now().Unix() > m.ExpiresAt
}

// TimeRemaining returns seconds until expiration (0 if perpetual, negative if expired)
func (m *Manifest) TimeRemaining() int64 {
	if m.ExpiresAt == 0 {
		return 0 // Perpetual
	}
	return m.ExpiresAt - time.Now().Unix()
}

// AddTrack adds a track marker to the manifest
func (m *Manifest) AddTrack(title string, start float64) *Manifest {
	m.Tracks = append(m.Tracks, Track{
		Title:    title,
		Start:    start,
		TrackNum: len(m.Tracks) + 1,
	})
	return m
}

// AddTrackFull adds a track with all details
func (m *Manifest) AddTrackFull(title string, start, end float64, trackType string) *Manifest {
	m.Tracks = append(m.Tracks, Track{
		Title:    title,
		Start:    start,
		End:      end,
		Type:     trackType,
		TrackNum: len(m.Tracks) + 1,
	})
	return m
}

// AddLink adds an artist link (platform -> URL)
func (m *Manifest) AddLink(platform, url string) *Manifest {
	if m.Links == nil {
		m.Links = make(map[string]string)
	}
	m.Links[platform] = url
	return m
}

// Format versions
const (
	FormatV1 = ""   // Original format: JSON with base64-encoded attachments
	FormatV2 = "v2" // Binary format: JSON header + raw binary attachments
	FormatV3 = "v3" // Streaming format: CEK wrapped with rolling LTHN keys
)

// Compression types
const (
	CompressionNone = ""     // No compression (default, backwards compatible)
	CompressionGzip = "gzip" // Gzip compression (stdlib, WASM compatible)
	CompressionZstd = "zstd" // Zstandard compression (faster, better ratio)
)

// Key derivation methods for v3 streaming
const (
	// KeyMethodDirect uses password directly (v1/v2 behavior)
	KeyMethodDirect = ""

	// KeyMethodLTHNRolling uses LTHN hash with rolling date windows
	// Key = SHA256(LTHN(date:license:fingerprint))
	// Valid keys: current period and next period (rolling window)
	KeyMethodLTHNRolling = "lthn-rolling"
)

// Cadence defines how often stream keys rotate
type Cadence string

const (
	// CadenceDaily rotates keys every 24 hours (default)
	// Date format: "2006-01-02"
	CadenceDaily Cadence = "daily"

	// CadenceHalfDay rotates keys every 12 hours
	// Date format: "2006-01-02-AM" or "2006-01-02-PM"
	CadenceHalfDay Cadence = "12h"

	// CadenceQuarter rotates keys every 6 hours
	// Date format: "2006-01-02-00", "2006-01-02-06", "2006-01-02-12", "2006-01-02-18"
	CadenceQuarter Cadence = "6h"

	// CadenceHourly rotates keys every hour
	// Date format: "2006-01-02-15" (24-hour format)
	CadenceHourly Cadence = "1h"
)

// WrappedKey represents a CEK (Content Encryption Key) wrapped with a time-bound stream key.
// The stream key is derived from LTHN(date:license:fingerprint) and is never transmitted.
// Only the wrapped CEK (which includes its own nonce) is stored in the header.
type WrappedKey struct {
	Date    string `json:"date"`    // ISO date "YYYY-MM-DD" for key derivation
	Wrapped string `json:"wrapped"` // base64([nonce][ChaCha(CEK, streamKey)])
}

// Header represents the SMSG container header
type Header struct {
	Version     string    `json:"version"`
	Algorithm   string    `json:"algorithm"`
	Format      string    `json:"format,omitempty"`      // v2 for binary, v3 for streaming, empty for v1 (base64)
	Compression string    `json:"compression,omitempty"` // gzip, zstd, or empty for none
	Hint        string    `json:"hint,omitempty"`        // optional password hint
	Manifest    *Manifest `json:"manifest,omitempty"`    // public metadata for discovery

	// V3 streaming fields
	KeyMethod   string       `json:"keyMethod,omitempty"`   // lthn-rolling for v3
	Cadence     Cadence      `json:"cadence,omitempty"`     // key rotation frequency (daily, 12h, 6h, 1h)
	WrappedKeys []WrappedKey `json:"wrappedKeys,omitempty"` // CEK wrapped with rolling keys
}
