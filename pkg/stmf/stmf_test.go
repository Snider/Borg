package stmf

import (
	"encoding/base64"
	"testing"
)

func TestKeyPairGeneration(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// X25519 keys are 32 bytes
	if len(kp.PublicKey()) != 32 {
		t.Errorf("Public key length = %d, want 32", len(kp.PublicKey()))
	}
	if len(kp.PrivateKey()) != 32 {
		t.Errorf("Private key length = %d, want 32", len(kp.PrivateKey()))
	}

	// Base64 encoding should work
	pubB64 := kp.PublicKeyBase64()
	privB64 := kp.PrivateKeyBase64()

	if pubB64 == "" || privB64 == "" {
		t.Error("Base64 encoding returned empty string")
	}

	// Should be able to decode back
	pubBytes, err := base64.StdEncoding.DecodeString(pubB64)
	if err != nil {
		t.Errorf("Failed to decode public key base64: %v", err)
	}
	if len(pubBytes) != 32 {
		t.Errorf("Decoded public key length = %d, want 32", len(pubBytes))
	}
}

func TestLoadKeyPair(t *testing.T) {
	// Generate a keypair
	original, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Load it back from bytes
	loaded, err := LoadKeyPair(original.PrivateKey())
	if err != nil {
		t.Fatalf("LoadKeyPair failed: %v", err)
	}

	// Public keys should match
	if string(loaded.PublicKey()) != string(original.PublicKey()) {
		t.Error("Loaded public key doesn't match original")
	}
}

func TestLoadKeyPairBase64(t *testing.T) {
	original, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	loaded, err := LoadKeyPairBase64(original.PrivateKeyBase64())
	if err != nil {
		t.Fatalf("LoadKeyPairBase64 failed: %v", err)
	}

	if loaded.PublicKeyBase64() != original.PublicKeyBase64() {
		t.Error("Loaded public key doesn't match original")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	// Generate server keypair
	serverKP, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Create form data
	formData := NewFormData().
		AddField("email", "test@example.com").
		AddFieldWithType("password", "secret123", "password").
		SetMetadata("origin", "https://example.com")

	// Encrypt with server's public key
	encrypted, err := Encrypt(formData, serverKP.PublicKey())
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Decrypt with server's private key
	decrypted, err := Decrypt(encrypted, serverKP.PrivateKey())
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	// Verify fields
	if decrypted.Get("email") != "test@example.com" {
		t.Errorf("email = %q, want %q", decrypted.Get("email"), "test@example.com")
	}
	if decrypted.Get("password") != "secret123" {
		t.Errorf("password = %q, want %q", decrypted.Get("password"), "secret123")
	}

	// Verify metadata
	if decrypted.Metadata["origin"] != "https://example.com" {
		t.Errorf("origin = %q, want %q", decrypted.Metadata["origin"], "https://example.com")
	}

	// Verify field type preserved
	pwField := decrypted.GetField("password")
	if pwField == nil {
		t.Error("password field not found")
	} else if pwField.Type != "password" {
		t.Errorf("password type = %q, want %q", pwField.Type, "password")
	}
}

func TestEncryptDecryptBase64RoundTrip(t *testing.T) {
	serverKP, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	formData := NewFormData().
		AddField("username", "johndoe").
		AddField("action", "login")

	// Encrypt to base64
	encryptedB64, err := EncryptBase64(formData, serverKP.PublicKey())
	if err != nil {
		t.Fatalf("EncryptBase64 failed: %v", err)
	}

	// Should be valid base64
	if _, err := base64.StdEncoding.DecodeString(encryptedB64); err != nil {
		t.Fatalf("Encrypted output is not valid base64: %v", err)
	}

	// Decrypt from base64
	decrypted, err := DecryptBase64(encryptedB64, serverKP.PrivateKey())
	if err != nil {
		t.Fatalf("DecryptBase64 failed: %v", err)
	}

	if decrypted.Get("username") != "johndoe" {
		t.Errorf("username = %q, want %q", decrypted.Get("username"), "johndoe")
	}
}

func TestEncryptMapRoundTrip(t *testing.T) {
	serverKP, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	input := map[string]string{
		"name":  "John Doe",
		"email": "john@example.com",
		"phone": "+1234567890",
	}

	encrypted, err := EncryptMap(input, serverKP.PublicKey())
	if err != nil {
		t.Fatalf("EncryptMap failed: %v", err)
	}

	output, err := DecryptToMap(encrypted, serverKP.PrivateKey())
	if err != nil {
		t.Fatalf("DecryptToMap failed: %v", err)
	}

	for key, want := range input {
		if got := output[key]; got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestMultipleEncryptionsAreDifferent(t *testing.T) {
	// Each encryption should use a different ephemeral key
	serverKP, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	formData := NewFormData().AddField("test", "value")

	enc1, err := Encrypt(formData, serverKP.PublicKey())
	if err != nil {
		t.Fatalf("First Encrypt failed: %v", err)
	}

	enc2, err := Encrypt(formData, serverKP.PublicKey())
	if err != nil {
		t.Fatalf("Second Encrypt failed: %v", err)
	}

	// Encryptions should be different (different ephemeral keys)
	if string(enc1) == string(enc2) {
		t.Error("Two encryptions of same data produced identical output (should use different ephemeral keys)")
	}

	// But both should decrypt to the same value
	dec1, _ := Decrypt(enc1, serverKP.PrivateKey())
	dec2, _ := Decrypt(enc2, serverKP.PrivateKey())

	if dec1.Get("test") != dec2.Get("test") {
		t.Error("Decrypted values don't match")
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	serverKP, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	wrongKP, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair for wrong key failed: %v", err)
	}

	formData := NewFormData().AddField("secret", "data")
	encrypted, err := Encrypt(formData, serverKP.PublicKey())
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Decrypting with wrong key should fail
	_, err = Decrypt(encrypted, wrongKP.PrivateKey())
	if err == nil {
		t.Error("Decrypt with wrong key should have failed")
	}
}

func TestValidatePayload(t *testing.T) {
	serverKP, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	formData := NewFormData().AddField("test", "value")
	encrypted, err := Encrypt(formData, serverKP.PublicKey())
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Valid payload should pass validation
	if err := ValidatePayload(encrypted); err != nil {
		t.Errorf("ValidatePayload failed for valid payload: %v", err)
	}

	// Invalid data should fail
	if err := ValidatePayload([]byte("not a valid payload")); err == nil {
		t.Error("ValidatePayload should fail for invalid data")
	}
}

func TestGetPayloadInfo(t *testing.T) {
	serverKP, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	formData := NewFormData().AddField("test", "value")
	encrypted, err := Encrypt(formData, serverKP.PublicKey())
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	info, err := GetPayloadInfo(encrypted)
	if err != nil {
		t.Fatalf("GetPayloadInfo failed: %v", err)
	}

	if info.Version != Version {
		t.Errorf("Version = %q, want %q", info.Version, Version)
	}
	if info.Algorithm != "x25519-chacha20poly1305" {
		t.Errorf("Algorithm = %q, want %q", info.Algorithm, "x25519-chacha20poly1305")
	}
	if info.EphemeralPK == "" {
		t.Error("EphemeralPK is empty")
	}
}

func TestFormDataMethods(t *testing.T) {
	fd := NewFormData()

	// Test AddField
	fd.AddField("name", "John")
	if fd.Get("name") != "John" {
		t.Error("AddField/Get failed")
	}

	// Test AddFieldWithType
	fd.AddFieldWithType("password", "secret", "password")
	field := fd.GetField("password")
	if field == nil || field.Type != "password" {
		t.Error("AddFieldWithType failed to preserve type")
	}

	// Test AddFile
	fd.AddFile("doc", "base64data", "document.pdf", "application/pdf")
	fileField := fd.GetField("doc")
	if fileField == nil {
		t.Error("AddFile failed")
	} else {
		if fileField.Filename != "document.pdf" {
			t.Error("Filename not preserved")
		}
		if fileField.MimeType != "application/pdf" {
			t.Error("MimeType not preserved")
		}
	}

	// Test SetMetadata
	fd.SetMetadata("origin", "https://test.com")
	if fd.Metadata["origin"] != "https://test.com" {
		t.Error("SetMetadata failed")
	}

	// Test GetAll with multiple values
	fd.AddField("tag", "one")
	fd.AddField("tag", "two")
	tags := fd.GetAll("tag")
	if len(tags) != 2 {
		t.Errorf("GetAll returned %d values, want 2", len(tags))
	}

	// Test ToMap
	m := fd.ToMap()
	if m["name"] != "John" {
		t.Error("ToMap failed")
	}
}

func TestFileFieldRoundTrip(t *testing.T) {
	serverKP, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Simulate file upload with base64 content
	fileContent := base64.StdEncoding.EncodeToString([]byte("Hello, World!"))

	formData := NewFormData().
		AddField("description", "My document").
		AddFile("upload", fileContent, "hello.txt", "text/plain")

	encrypted, err := Encrypt(formData, serverKP.PublicKey())
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(encrypted, serverKP.PrivateKey())
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	uploadField := decrypted.GetField("upload")
	if uploadField == nil {
		t.Fatal("upload field not found")
	}

	if uploadField.Type != "file" {
		t.Errorf("Type = %q, want %q", uploadField.Type, "file")
	}
	if uploadField.Filename != "hello.txt" {
		t.Errorf("Filename = %q, want %q", uploadField.Filename, "hello.txt")
	}
	if uploadField.MimeType != "text/plain" {
		t.Errorf("MimeType = %q, want %q", uploadField.MimeType, "text/plain")
	}
	if uploadField.Value != fileContent {
		t.Error("File content not preserved")
	}
}
