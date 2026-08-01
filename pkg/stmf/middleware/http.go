// Package middleware provides HTTP middleware for automatic STMF decryption.
package middleware

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"

	"github.com/Snider/Borg/pkg/stmf"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// FormDataKey is the context key for the decrypted FormData
	FormDataKey contextKey = "stmf_form_data"

	// MetadataKey is the context key for the form metadata
	MetadataKey contextKey = "stmf_metadata"
)

// Config holds the middleware configuration
type Config struct {
	// PrivateKey is the server's X25519 private key (32 bytes)
	PrivateKey []byte

	// FieldName is the form field name containing the STMF payload
	// Defaults to "_stmf_payload" if empty
	FieldName string

	// OnError is called when decryption fails
	// If nil, returns 400 Bad Request
	OnError func(w http.ResponseWriter, r *http.Request, err error)

	// OnMissingPayload is called when the STMF field is not present
	// If nil, the request passes through unchanged
	OnMissingPayload func(w http.ResponseWriter, r *http.Request)

	// PopulateForm controls whether decrypted fields are added to r.Form
	// Defaults to true
	PopulateForm *bool
}

// DefaultConfig returns a Config with default values
func DefaultConfig(privateKey []byte) Config {
	populateForm := true
	return Config{
		PrivateKey:   privateKey,
		FieldName:    stmf.DefaultFieldName,
		PopulateForm: &populateForm,
	}
}

// Middleware creates an HTTP middleware that decrypts STMF payloads.
// It looks for the STMF payload in the configured field name,
// decrypts it, and populates r.Form with the decrypted fields.
func Middleware(cfg Config) func(http.Handler) http.Handler {
	if cfg.FieldName == "" {
		cfg.FieldName = stmf.DefaultFieldName
	}
	if cfg.PopulateForm == nil {
		populateForm := true
		cfg.PopulateForm = &populateForm
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only process POST/PUT/PATCH requests
			if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
				next.ServeHTTP(w, r)
				return
			}

			// Parse the form
			if err := r.ParseMultipartForm(32 << 20); err != nil {
				// Try regular form parsing
				if err := r.ParseForm(); err != nil {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Look for STMF payload
			payloadB64 := r.FormValue(cfg.FieldName)
			if payloadB64 == "" {
				if cfg.OnMissingPayload != nil {
					cfg.OnMissingPayload(w, r)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// Decode base64
			payloadBytes, err := base64.StdEncoding.DecodeString(payloadB64)
			if err != nil {
				handleError(w, r, cfg, stmf.ErrInvalidPayload)
				return
			}

			// Decrypt
			formData, err := stmf.Decrypt(payloadBytes, cfg.PrivateKey)
			if err != nil {
				handleError(w, r, cfg, err)
				return
			}

			// Store in context
			ctx := r.Context()
			ctx = context.WithValue(ctx, FormDataKey, formData)
			if formData.Metadata != nil {
				ctx = context.WithValue(ctx, MetadataKey, formData.Metadata)
			}

			// Populate r.Form with decrypted fields
			if *cfg.PopulateForm {
				if r.Form == nil {
					r.Form = make(url.Values)
				}
				for _, field := range formData.Fields {
					r.Form.Set(field.Name, field.Value)
				}
				// Also populate PostForm
				if r.PostForm == nil {
					r.PostForm = make(url.Values)
				}
				for _, field := range formData.Fields {
					r.PostForm.Set(field.Name, field.Value)
				}
			}

			// Remove the encrypted payload field
			if r.Form != nil {
				delete(r.Form, cfg.FieldName)
			}
			if r.PostForm != nil {
				delete(r.PostForm, cfg.FieldName)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// handleError calls the error handler or returns 400
func handleError(w http.ResponseWriter, r *http.Request, cfg Config, err error) {
	if cfg.OnError != nil {
		cfg.OnError(w, r, err)
		return
	}
	http.Error(w, "Invalid encrypted payload", http.StatusBadRequest)
}

// Simple creates a simple middleware with just a private key
func Simple(privateKey []byte) func(http.Handler) http.Handler {
	return Middleware(DefaultConfig(privateKey))
}

// SimpleBase64 creates a simple middleware with a base64-encoded private key
func SimpleBase64(privateKeyB64 string) (func(http.Handler) http.Handler, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(privateKeyB64)
	if err != nil {
		return nil, err
	}
	return Simple(keyBytes), nil
}

// GetFormData retrieves the decrypted FormData from the request context
func GetFormData(r *http.Request) *stmf.FormData {
	if fd, ok := r.Context().Value(FormDataKey).(*stmf.FormData); ok {
		return fd
	}
	return nil
}

// GetMetadata retrieves the form metadata from the request context
func GetMetadata(r *http.Request) map[string]string {
	if md, ok := r.Context().Value(MetadataKey).(map[string]string); ok {
		return md
	}
	return nil
}

// HasSTMFPayload checks if the request contains a STMF payload
func HasSTMFPayload(r *http.Request, fieldName string) bool {
	if fieldName == "" {
		fieldName = stmf.DefaultFieldName
	}
	return r.FormValue(fieldName) != ""
}
