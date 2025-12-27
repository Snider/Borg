package smsg

import (
	"encoding/base64"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	msg := NewMessage("Hello, this is a secure message!").
		WithSubject("Test Subject").
		WithFrom("support@example.com")

	password := "supersecret123"

	encrypted, err := Encrypt(msg, password)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(encrypted, password)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted.Body != msg.Body {
		t.Errorf("Body = %q, want %q", decrypted.Body, msg.Body)
	}
	if decrypted.Subject != msg.Subject {
		t.Errorf("Subject = %q, want %q", decrypted.Subject, msg.Subject)
	}
	if decrypted.From != msg.From {
		t.Errorf("From = %q, want %q", decrypted.From, msg.From)
	}
}

func TestBase64RoundTrip(t *testing.T) {
	msg := NewMessage("Base64 test message")
	password := "testpass"

	encryptedB64, err := EncryptBase64(msg, password)
	if err != nil {
		t.Fatalf("EncryptBase64 failed: %v", err)
	}

	// Should be valid base64
	if _, err := base64.StdEncoding.DecodeString(encryptedB64); err != nil {
		t.Fatalf("Invalid base64: %v", err)
	}

	decrypted, err := DecryptBase64(encryptedB64, password)
	if err != nil {
		t.Fatalf("DecryptBase64 failed: %v", err)
	}

	if decrypted.Body != msg.Body {
		t.Errorf("Body = %q, want %q", decrypted.Body, msg.Body)
	}
}

func TestWithAttachments(t *testing.T) {
	fileContent := base64.StdEncoding.EncodeToString([]byte("Hello, World!"))

	msg := NewMessage("Please see the attached file.").
		AddAttachment("hello.txt", fileContent, "text/plain").
		AddAttachment("data.json", base64.StdEncoding.EncodeToString([]byte(`{"key":"value"}`)), "application/json")

	password := "attachtest"

	encrypted, err := Encrypt(msg, password)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(encrypted, password)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if len(decrypted.Attachments) != 2 {
		t.Fatalf("Attachments count = %d, want 2", len(decrypted.Attachments))
	}

	att := decrypted.GetAttachment("hello.txt")
	if att == nil {
		t.Fatal("Attachment hello.txt not found")
	}
	if att.Content != fileContent {
		t.Error("Attachment content mismatch")
	}
	if att.MimeType != "text/plain" {
		t.Errorf("MimeType = %q, want %q", att.MimeType, "text/plain")
	}
}

func TestWithReplyKey(t *testing.T) {
	msg := NewMessage("Here's a public key for your reply.").
		WithReplyKey("dGVzdHB1YmxpY2tleWJhc2U2NA==")

	password := "pki-test"

	encrypted, err := Encrypt(msg, password)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(encrypted, password)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted.ReplyKey == nil {
		t.Fatal("ReplyKey is nil")
	}
	if decrypted.ReplyKey.PublicKey != "dGVzdHB1YmxpY2tleWJhc2U2NA==" {
		t.Error("ReplyKey.PublicKey mismatch")
	}
	if decrypted.ReplyKey.Algorithm != "x25519" {
		t.Errorf("Algorithm = %q, want %q", decrypted.ReplyKey.Algorithm, "x25519")
	}
}

func TestWithHint(t *testing.T) {
	msg := NewMessage("Password hint test")
	password := "birthday1990"
	hint := "Your birthday year"

	encrypted, err := EncryptWithHint(msg, password, hint)
	if err != nil {
		t.Fatalf("EncryptWithHint failed: %v", err)
	}

	// Get info should include hint
	info, err := GetInfo(encrypted)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if info.Hint != hint {
		t.Errorf("Hint = %q, want %q", info.Hint, hint)
	}

	// Should still decrypt
	decrypted, err := Decrypt(encrypted, password)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted.Body != msg.Body {
		t.Error("Body mismatch")
	}
}

func TestWrongPassword(t *testing.T) {
	msg := NewMessage("Secret message")
	password := "correct-password"

	encrypted, err := Encrypt(msg, password)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(encrypted, "wrong-password")
	if err == nil {
		t.Error("Decrypt with wrong password should have failed")
	}
}

func TestQuickFunctions(t *testing.T) {
	body := "Quick test message"
	password := "quickpass"

	encrypted, err := QuickEncrypt(body, password)
	if err != nil {
		t.Fatalf("QuickEncrypt failed: %v", err)
	}

	decrypted, err := QuickDecrypt(encrypted, password)
	if err != nil {
		t.Fatalf("QuickDecrypt failed: %v", err)
	}

	if decrypted != body {
		t.Errorf("Decrypted = %q, want %q", decrypted, body)
	}
}

func TestUnicodeContent(t *testing.T) {
	msg := NewMessage("日本語メッセージ 🔐 مرحبا").
		WithSubject("Unicode テスト").
		WithFrom("サポート")

	password := "unicode-test"

	encrypted, err := Encrypt(msg, password)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(encrypted, password)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted.Body != msg.Body {
		t.Errorf("Body = %q, want %q", decrypted.Body, msg.Body)
	}
	if decrypted.Subject != msg.Subject {
		t.Errorf("Subject = %q, want %q", decrypted.Subject, msg.Subject)
	}
}

func TestMetadata(t *testing.T) {
	msg := NewMessage("Message with metadata").
		SetMeta("ticket_id", "12345").
		SetMeta("priority", "high")

	password := "meta-test"

	encrypted, err := Encrypt(msg, password)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(encrypted, password)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted.Meta["ticket_id"] != "12345" {
		t.Error("ticket_id metadata mismatch")
	}
	if decrypted.Meta["priority"] != "high" {
		t.Error("priority metadata mismatch")
	}
}

func TestValidate(t *testing.T) {
	msg := NewMessage("Test")
	password := "test"

	encrypted, _ := Encrypt(msg, password)

	// Valid SMSG should pass
	if err := Validate(encrypted); err != nil {
		t.Errorf("Validate failed for valid SMSG: %v", err)
	}

	// Invalid data should fail
	if err := Validate([]byte("not an smsg")); err == nil {
		t.Error("Validate should fail for invalid data")
	}
}

func TestEmptyPasswordError(t *testing.T) {
	msg := NewMessage("Test")

	_, err := Encrypt(msg, "")
	if err != ErrPasswordRequired {
		t.Errorf("Expected ErrPasswordRequired, got %v", err)
	}
}

func TestEmptyMessageError(t *testing.T) {
	msg := &Message{}

	_, err := Encrypt(msg, "password")
	if err != ErrEmptyMessage {
		t.Errorf("Expected ErrEmptyMessage, got %v", err)
	}
}
