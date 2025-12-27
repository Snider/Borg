// Package stmf implements Sovereign Form Encryption using X25519 ECDH + ChaCha20-Poly1305.
// STMF (STIM Form) enables client-side encryption of HTML form data using the server's
// public key, providing end-to-end encryption even against MITM proxies.
package stmf

import (
	"errors"
)

// Magic bytes for STMF format
const Magic = "STMF"

// Version of the STMF format
const Version = "1.0"

// DefaultFieldName is the form field name used for the encrypted payload
const DefaultFieldName = "_stmf_payload"

// Errors
var (
	ErrInvalidMagic       = errors.New("invalid STMF magic")
	ErrInvalidPayload     = errors.New("invalid STMF payload")
	ErrDecryptionFailed   = errors.New("decryption failed")
	ErrInvalidPublicKey   = errors.New("invalid public key")
	ErrInvalidPrivateKey  = errors.New("invalid private key")
	ErrKeyGenerationFailed = errors.New("key generation failed")
)

// FormField represents a single form field
type FormField struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Type     string `json:"type,omitempty"`     // text, password, file, etc.
	Filename string `json:"filename,omitempty"` // for file uploads
	MimeType string `json:"mime,omitempty"`     // for file uploads
}

// FormData represents the encrypted form payload
type FormData struct {
	Fields   []FormField       `json:"fields"`
	Metadata map[string]string `json:"meta,omitempty"`
}

// NewFormData creates a new empty FormData
func NewFormData() *FormData {
	return &FormData{
		Fields:   make([]FormField, 0),
		Metadata: make(map[string]string),
	}
}

// AddField adds a field to the form data
func (f *FormData) AddField(name, value string) *FormData {
	f.Fields = append(f.Fields, FormField{
		Name:  name,
		Value: value,
		Type:  "text",
	})
	return f
}

// AddFieldWithType adds a typed field to the form data
func (f *FormData) AddFieldWithType(name, value, fieldType string) *FormData {
	f.Fields = append(f.Fields, FormField{
		Name:  name,
		Value: value,
		Type:  fieldType,
	})
	return f
}

// AddFile adds a file field to the form data
func (f *FormData) AddFile(name, value, filename, mimeType string) *FormData {
	f.Fields = append(f.Fields, FormField{
		Name:     name,
		Value:    value,
		Type:     "file",
		Filename: filename,
		MimeType: mimeType,
	})
	return f
}

// SetMetadata sets a metadata value
func (f *FormData) SetMetadata(key, value string) *FormData {
	if f.Metadata == nil {
		f.Metadata = make(map[string]string)
	}
	f.Metadata[key] = value
	return f
}

// Get retrieves a field value by name
func (f *FormData) Get(name string) string {
	for _, field := range f.Fields {
		if field.Name == name {
			return field.Value
		}
	}
	return ""
}

// GetField retrieves a full field by name
func (f *FormData) GetField(name string) *FormField {
	for i := range f.Fields {
		if f.Fields[i].Name == name {
			return &f.Fields[i]
		}
	}
	return nil
}

// GetAll retrieves all values for a field name (for multi-select)
func (f *FormData) GetAll(name string) []string {
	var values []string
	for _, field := range f.Fields {
		if field.Name == name {
			values = append(values, field.Value)
		}
	}
	return values
}

// ToMap converts fields to a simple key-value map (last value wins for duplicates)
func (f *FormData) ToMap() map[string]string {
	result := make(map[string]string)
	for _, field := range f.Fields {
		result[field.Name] = field.Value
	}
	return result
}

// Header represents the STMF container header
type Header struct {
	Version     string `json:"version"`
	Algorithm   string `json:"algorithm"`
	EphemeralPK string `json:"ephemeral_pk"` // base64-encoded ephemeral public key
	Nonce       string `json:"nonce"`        // base64-encoded nonce
}
