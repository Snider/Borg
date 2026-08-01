package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Snider/Borg/pkg/stmf"
)

func TestMiddleware(t *testing.T) {
	// Generate server keypair
	serverKP, err := stmf.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Create form data and encrypt it
	formData := stmf.NewFormData().
		AddField("email", "test@example.com").
		AddFieldWithType("password", "secret123", "password")

	encryptedB64, err := stmf.EncryptBase64(formData, serverKP.PublicKey())
	if err != nil {
		t.Fatalf("EncryptBase64 failed: %v", err)
	}

	// Create middleware
	mw := Simple(serverKP.PrivateKey())

	// Create test handler
	var capturedEmail, capturedPassword string
	var capturedFormData *stmf.FormData
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedEmail = r.FormValue("email")
		capturedPassword = r.FormValue("password")
		capturedFormData = GetFormData(r)
		w.WriteHeader(http.StatusOK)
	}))

	// Create request with encrypted payload
	form := url.Values{}
	form.Set(stmf.DefaultFieldName, encryptedB64)

	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify decrypted fields are in r.FormValue
	if capturedEmail != "test@example.com" {
		t.Errorf("email = %q, want %q", capturedEmail, "test@example.com")
	}
	if capturedPassword != "secret123" {
		t.Errorf("password = %q, want %q", capturedPassword, "secret123")
	}

	// Verify context FormData
	if capturedFormData == nil {
		t.Error("FormData not in context")
	} else if capturedFormData.Get("email") != "test@example.com" {
		t.Error("FormData email incorrect")
	}
}

func TestMiddlewarePassThrough(t *testing.T) {
	serverKP, err := stmf.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	mw := Simple(serverKP.PrivateKey())

	var called bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	// Request without STMF payload should pass through
	form := url.Values{}
	form.Set("regular_field", "value")

	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("Handler was not called for request without STMF payload")
	}
}

func TestMiddlewareGetRequest(t *testing.T) {
	serverKP, err := stmf.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	mw := Simple(serverKP.PrivateKey())

	var called bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	// GET request should pass through without processing
	req := httptest.NewRequest(http.MethodGet, "/page", nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("Handler was not called for GET request")
	}
}

func TestMiddlewareInvalidPayload(t *testing.T) {
	serverKP, err := stmf.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	mw := Simple(serverKP.PrivateKey())

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for invalid payload")
	}))

	// Request with invalid STMF payload
	form := url.Values{}
	form.Set(stmf.DefaultFieldName, "invalid-not-base64!!!!")

	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestMiddlewareWrongKey(t *testing.T) {
	serverKP, err := stmf.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	wrongKP, err := stmf.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Encrypt with server's public key
	formData := stmf.NewFormData().AddField("test", "value")
	encryptedB64, _ := stmf.EncryptBase64(formData, serverKP.PublicKey())

	// But use wrong private key in middleware
	mw := Simple(wrongKP.PrivateKey())

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for wrong key")
	}))

	form := url.Values{}
	form.Set(stmf.DefaultFieldName, encryptedB64)

	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestMiddlewareCustomErrorHandler(t *testing.T) {
	serverKP, err := stmf.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	var errorHandlerCalled bool
	cfg := DefaultConfig(serverKP.PrivateKey())
	cfg.OnError = func(w http.ResponseWriter, r *http.Request, err error) {
		errorHandlerCalled = true
		http.Error(w, "Custom error", http.StatusUnprocessableEntity)
	}

	mw := Middleware(cfg)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	}))

	form := url.Values{}
	form.Set(stmf.DefaultFieldName, "invalid")

	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !errorHandlerCalled {
		t.Error("Custom error handler was not called")
	}
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
}

func TestMiddlewareWithMetadata(t *testing.T) {
	serverKP, err := stmf.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	formData := stmf.NewFormData().
		AddField("email", "test@example.com").
		SetMetadata("origin", "https://example.com").
		SetMetadata("timestamp", "1234567890")

	encryptedB64, _ := stmf.EncryptBase64(formData, serverKP.PublicKey())

	mw := Simple(serverKP.PrivateKey())

	var capturedMetadata map[string]string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMetadata = GetMetadata(r)
		w.WriteHeader(http.StatusOK)
	}))

	form := url.Values{}
	form.Set(stmf.DefaultFieldName, encryptedB64)

	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if capturedMetadata == nil {
		t.Fatal("Metadata not in context")
	}
	if capturedMetadata["origin"] != "https://example.com" {
		t.Errorf("origin = %q, want %q", capturedMetadata["origin"], "https://example.com")
	}
}

func TestSimpleBase64(t *testing.T) {
	serverKP, err := stmf.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	mw, err := SimpleBase64(serverKP.PrivateKeyBase64())
	if err != nil {
		t.Fatalf("SimpleBase64 failed: %v", err)
	}

	// Create and encrypt form data
	formData := stmf.NewFormData().AddField("test", "value")
	encryptedB64, _ := stmf.EncryptBase64(formData, serverKP.PublicKey())

	var capturedValue string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedValue = r.FormValue("test")
		w.WriteHeader(http.StatusOK)
	}))

	form := url.Values{}
	form.Set(stmf.DefaultFieldName, encryptedB64)

	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if capturedValue != "value" {
		t.Errorf("test = %q, want %q", capturedValue, "value")
	}
}
