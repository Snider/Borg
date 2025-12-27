// Package smsg implements Secure Message encryption using password-based ChaCha20-Poly1305.
// SMSG (Secure Message) enables encrypted message exchange where the recipient
// decrypts using a pre-shared password. Useful for secure support replies,
// confidential documents, and any scenario requiring password-protected content.
package smsg

import (
	"errors"
)

// Magic bytes for SMSG format
const Magic = "SMSG"

// Version of the SMSG format
const Version = "1.0"

// Errors
var (
	ErrInvalidMagic      = errors.New("invalid SMSG magic")
	ErrInvalidPayload    = errors.New("invalid SMSG payload")
	ErrDecryptionFailed  = errors.New("decryption failed (wrong password?)")
	ErrPasswordRequired  = errors.New("password is required")
	ErrEmptyMessage      = errors.New("message cannot be empty")
)

// Attachment represents a file attached to the message
type Attachment struct {
	Name     string `json:"name"`
	Content  string `json:"content"`  // base64-encoded
	MimeType string `json:"mime,omitempty"`
	Size     int    `json:"size,omitempty"`
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

// AddAttachment adds a file attachment
func (m *Message) AddAttachment(name, content, mimeType string) *Message {
	m.Attachments = append(m.Attachments, Attachment{
		Name:     name,
		Content:  content,
		MimeType: mimeType,
		Size:     len(content),
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

// Header represents the SMSG container header
type Header struct {
	Version   string `json:"version"`
	Algorithm string `json:"algorithm"`
	Hint      string `json:"hint,omitempty"` // optional password hint
}
