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

func TestEncryptWithManifest(t *testing.T) {
	msg := NewMessage("Licensed content")
	password := "license-token-123"

	// Create manifest with tracks
	manifest := NewManifest("Summer EP 2024").
		AddTrackFull("Intro", 0, 30, "intro").
		AddTrackFull("Main Track", 30, 180, "full").
		AddTrack("Outro", 180)
	manifest.Artist = "Test Artist"
	manifest.ReleaseType = "ep"
	manifest.Format = "dapp.fm/v1"

	encrypted, err := EncryptWithManifest(msg, password, manifest)
	if err != nil {
		t.Fatalf("EncryptWithManifest failed: %v", err)
	}

	// Get info without decryption - should have manifest
	header, err := GetInfo(encrypted)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if header.Manifest == nil {
		t.Fatal("Expected manifest in header")
	}

	if header.Manifest.Title != "Summer EP 2024" {
		t.Errorf("Title = %q, want %q", header.Manifest.Title, "Summer EP 2024")
	}

	if header.Manifest.Artist != "Test Artist" {
		t.Errorf("Artist = %q, want %q", header.Manifest.Artist, "Test Artist")
	}

	if header.Manifest.ReleaseType != "ep" {
		t.Errorf("ReleaseType = %q, want %q", header.Manifest.ReleaseType, "ep")
	}

	if len(header.Manifest.Tracks) != 3 {
		t.Errorf("Tracks count = %d, want 3", len(header.Manifest.Tracks))
	}

	// Verify tracks
	if header.Manifest.Tracks[0].Title != "Intro" {
		t.Errorf("Track 0 Title = %q, want %q", header.Manifest.Tracks[0].Title, "Intro")
	}
	if header.Manifest.Tracks[0].Start != 0 {
		t.Errorf("Track 0 Start = %v, want 0", header.Manifest.Tracks[0].Start)
	}
	if header.Manifest.Tracks[0].Type != "intro" {
		t.Errorf("Track 0 Type = %q, want %q", header.Manifest.Tracks[0].Type, "intro")
	}

	// Can still decrypt normally
	decrypted, err := Decrypt(encrypted, password)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted.Body != "Licensed content" {
		t.Errorf("Body = %q, want %q", decrypted.Body, "Licensed content")
	}
}

func TestManifestBuilder(t *testing.T) {
	manifest := NewManifest("Test Album")
	manifest.Artist = "Artist Name"
	manifest.Album = "Album Name"
	manifest.Year = 2024
	manifest.Genre = "Electronic"
	manifest.ReleaseType = "album"
	manifest.Tags = []string{"electronic", "ambient"}
	manifest.Extra["custom_field"] = "custom_value"

	// Add tracks
	manifest.AddTrack("Track 1", 0)
	manifest.AddTrack("Track 2", 120)
	manifest.AddTrackFull("Track 3", 240, 360, "outro")

	if manifest.Title != "Test Album" {
		t.Errorf("Title = %q, want %q", manifest.Title, "Test Album")
	}

	if len(manifest.Tracks) != 3 {
		t.Fatalf("Track count = %d, want 3", len(manifest.Tracks))
	}

	// First track should have TrackNum 1
	if manifest.Tracks[0].TrackNum != 1 {
		t.Errorf("Track 1 TrackNum = %d, want 1", manifest.Tracks[0].TrackNum)
	}

	// Third track should have end time
	if manifest.Tracks[2].End != 360 {
		t.Errorf("Track 3 End = %v, want 360", manifest.Tracks[2].End)
	}
}

func TestManifestExpiration(t *testing.T) {
	// Test perpetual license (no expiration)
	perpetual := NewManifest("Perpetual Album")
	if perpetual.IsExpired() {
		t.Error("Perpetual license should not be expired")
	}
	if perpetual.TimeRemaining() != 0 {
		t.Error("Perpetual license should have 0 time remaining (infinite)")
	}
	if perpetual.LicenseType != "perpetual" {
		t.Errorf("LicenseType = %q, want perpetual", perpetual.LicenseType)
	}

	// Test streaming access (24 hours)
	stream := NewManifest("Stream Album").WithStreamingAccess(24)
	if stream.IsExpired() {
		t.Error("Streaming license should not be expired immediately")
	}
	if stream.LicenseType != "stream" {
		t.Errorf("LicenseType = %q, want stream", stream.LicenseType)
	}
	remaining := stream.TimeRemaining()
	if remaining < 86000 || remaining > 86400 {
		t.Errorf("TimeRemaining = %d, expected ~86400", remaining)
	}

	// Test rental with duration
	rental := NewManifest("Rental Album").WithRentalDuration(3600) // 1 hour
	if rental.IsExpired() {
		t.Error("Rental license should not be expired immediately")
	}
	if rental.LicenseType != "rental" {
		t.Errorf("LicenseType = %q, want rental", rental.LicenseType)
	}

	// Test preview (30 seconds)
	preview := NewManifest("Preview Track").WithPreviewAccess(30)
	if preview.IsExpired() {
		t.Error("Preview license should not be expired immediately")
	}
	if preview.LicenseType != "preview" {
		t.Errorf("LicenseType = %q, want preview", preview.LicenseType)
	}

	// Test already expired license
	expired := NewManifest("Expired Album")
	expired.ExpiresAt = 1000 // Very old timestamp
	if !expired.IsExpired() {
		t.Error("License with old expiration should be expired")
	}
	if expired.TimeRemaining() >= 0 {
		t.Error("Expired license should have negative time remaining")
	}
}

func TestExpirationInHeader(t *testing.T) {
	msg := NewMessage("Licensed content")
	password := "stream-token-123"

	// Create streaming license (24 hours)
	manifest := NewManifest("Streaming EP").WithStreamingAccess(24)

	encrypted, err := EncryptWithManifest(msg, password, manifest)
	if err != nil {
		t.Fatalf("EncryptWithManifest failed: %v", err)
	}

	// Get info should show expiration
	header, err := GetInfo(encrypted)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if header.Manifest == nil {
		t.Fatal("Expected manifest in header")
	}

	if header.Manifest.LicenseType != "stream" {
		t.Errorf("LicenseType = %q, want stream", header.Manifest.LicenseType)
	}

	if header.Manifest.ExpiresAt == 0 {
		t.Error("ExpiresAt should not be 0 for streaming license")
	}

	if header.Manifest.IssuedAt == 0 {
		t.Error("IssuedAt should not be 0")
	}

	if header.Manifest.IsExpired() {
		t.Error("New streaming license should not be expired")
	}
}

func TestManifestLinks(t *testing.T) {
	manifest := NewManifest("Test Track").
		AddLink("home", "https://example.com/artist").
		AddLink("beatport", "https://beatport.com/artist/test").
		AddLink("soundcloud", "https://soundcloud.com/test")

	if len(manifest.Links) != 3 {
		t.Fatalf("Links count = %d, want 3", len(manifest.Links))
	}

	if manifest.Links["home"] != "https://example.com/artist" {
		t.Errorf("Links[home] = %q, want %q", manifest.Links["home"], "https://example.com/artist")
	}

	if manifest.Links["beatport"] != "https://beatport.com/artist/test" {
		t.Errorf("Links[beatport] = %q, want %q", manifest.Links["beatport"], "https://beatport.com/artist/test")
	}

	// Test manifest with links in encrypted message
	msg := NewMessage("Track content")
	password := "link-test"

	encrypted, err := EncryptWithManifest(msg, password, manifest)
	if err != nil {
		t.Fatalf("EncryptWithManifest failed: %v", err)
	}

	header, err := GetInfo(encrypted)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if header.Manifest == nil {
		t.Fatal("Expected manifest in header")
	}

	if len(header.Manifest.Links) != 3 {
		t.Fatalf("Header Links count = %d, want 3", len(header.Manifest.Links))
	}

	if header.Manifest.Links["home"] != "https://example.com/artist" {
		t.Errorf("Header Links[home] = %q, want %q", header.Manifest.Links["home"], "https://example.com/artist")
	}
}

func TestV2BinaryFormat(t *testing.T) {
	// Create message with binary attachment
	binaryData := []byte("Hello, this is binary content! \x00\x01\x02\x03")
	msg := NewMessage("V2 format test").
		AddBinaryAttachment("test.bin", binaryData, "application/octet-stream")

	password := "v2-test"

	// Encrypt with v2 format
	encrypted, err := EncryptV2(msg, password)
	if err != nil {
		t.Fatalf("EncryptV2 failed: %v", err)
	}

	// Check header
	header, err := GetInfo(encrypted)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if header.Format != FormatV2 {
		t.Errorf("Format = %q, want %q", header.Format, FormatV2)
	}

	if header.Compression != CompressionZstd {
		t.Errorf("Compression = %q, want %q", header.Compression, CompressionZstd)
	}

	// Decrypt
	decrypted, err := Decrypt(encrypted, password)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted.Body != "V2 format test" {
		t.Errorf("Body = %q, want %q", decrypted.Body, "V2 format test")
	}

	if len(decrypted.Attachments) != 1 {
		t.Fatalf("Attachments count = %d, want 1", len(decrypted.Attachments))
	}

	att := decrypted.Attachments[0]
	if att.Name != "test.bin" {
		t.Errorf("Attachment name = %q, want %q", att.Name, "test.bin")
	}

	// Decode attachment and verify content
	decoded, err := base64.StdEncoding.DecodeString(att.Content)
	if err != nil {
		t.Fatalf("Failed to decode attachment: %v", err)
	}

	if string(decoded) != string(binaryData) {
		t.Errorf("Attachment content mismatch")
	}
}

func TestV2WithManifest(t *testing.T) {
	binaryData := make([]byte, 1024) // 1KB of zeros
	for i := range binaryData {
		binaryData[i] = byte(i % 256)
	}

	msg := NewMessage("V2 with manifest").
		AddBinaryAttachment("data.bin", binaryData, "application/octet-stream")

	manifest := NewManifest("Test Album").
		AddLink("home", "https://example.com")
	manifest.Artist = "Test Artist"

	password := "v2-manifest-test"

	encrypted, err := EncryptV2WithManifest(msg, password, manifest)
	if err != nil {
		t.Fatalf("EncryptV2WithManifest failed: %v", err)
	}

	// Verify header
	header, err := GetInfo(encrypted)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if header.Format != FormatV2 {
		t.Errorf("Format = %q, want %q", header.Format, FormatV2)
	}

	if header.Manifest == nil {
		t.Fatal("Expected manifest")
	}

	if header.Manifest.Title != "Test Album" {
		t.Errorf("Manifest Title = %q, want %q", header.Manifest.Title, "Test Album")
	}

	if header.Manifest.Artist != "Test Artist" {
		t.Errorf("Manifest Artist = %q, want %q", header.Manifest.Artist, "Test Artist")
	}

	// Decrypt and verify
	decrypted, err := Decrypt(encrypted, password)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if len(decrypted.Attachments) != 1 {
		t.Fatalf("Attachments count = %d, want 1", len(decrypted.Attachments))
	}

	decoded, _ := base64.StdEncoding.DecodeString(decrypted.Attachments[0].Content)
	if len(decoded) != 1024 {
		t.Errorf("Decoded length = %d, want 1024", len(decoded))
	}
}

func TestV2SizeSavings(t *testing.T) {
	// Create a message with binary data
	binaryData := make([]byte, 10000) // 10KB
	for i := range binaryData {
		binaryData[i] = byte(i % 256)
	}

	msg := NewMessage("Size comparison test")
	msg.AddBinaryAttachment("large.bin", binaryData, "application/octet-stream")

	password := "size-test"

	// Encrypt with v1 (base64)
	v1Encrypted, err := Encrypt(msg, password)
	if err != nil {
		t.Fatalf("Encrypt v1 failed: %v", err)
	}

	// Encrypt with v2 (binary + gzip)
	v2Encrypted, err := EncryptV2(msg, password)
	if err != nil {
		t.Fatalf("EncryptV2 failed: %v", err)
	}

	t.Logf("V1 size: %d bytes", len(v1Encrypted))
	t.Logf("V2 size: %d bytes", len(v2Encrypted))
	t.Logf("Savings: %.1f%%", (1.0-float64(len(v2Encrypted))/float64(len(v1Encrypted)))*100)

	// V2 should be smaller (at least 20% savings from base64 removal alone)
	if len(v2Encrypted) >= len(v1Encrypted) {
		t.Errorf("V2 should be smaller than V1: v2=%d, v1=%d", len(v2Encrypted), len(v1Encrypted))
	}

	// Both should decrypt to the same content
	d1, _ := Decrypt(v1Encrypted, password)
	d2, _ := Decrypt(v2Encrypted, password)

	if d1.Body != d2.Body {
		t.Error("Decrypted bodies don't match")
	}

	c1, _ := base64.StdEncoding.DecodeString(d1.Attachments[0].Content)
	c2, _ := base64.StdEncoding.DecodeString(d2.Attachments[0].Content)

	if string(c1) != string(c2) {
		t.Error("Decrypted attachment content doesn't match")
	}
}

func TestV2NoCompression(t *testing.T) {
	msg := NewMessage("No compression test").
		AddBinaryAttachment("test.txt", []byte("Hello World"), "text/plain")

	password := "no-compress"

	// Encrypt without compression
	encrypted, err := EncryptV2WithOptions(msg, password, nil, CompressionNone)
	if err != nil {
		t.Fatalf("EncryptV2WithOptions failed: %v", err)
	}

	header, err := GetInfo(encrypted)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if header.Format != FormatV2 {
		t.Errorf("Format = %q, want %q", header.Format, FormatV2)
	}

	if header.Compression != "" {
		t.Errorf("Compression = %q, want empty", header.Compression)
	}

	// Should still decrypt
	decrypted, err := Decrypt(encrypted, password)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted.Body != "No compression test" {
		t.Errorf("Body = %q, want %q", decrypted.Body, "No compression test")
	}
}
