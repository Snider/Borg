package smsg

import (
	"testing"
	"time"
)

func TestDeriveStreamKey(t *testing.T) {
	// Test that same inputs produce same key
	key1 := DeriveStreamKey("2026-01-12", "license123", "fingerprint456")
	key2 := DeriveStreamKey("2026-01-12", "license123", "fingerprint456")

	if len(key1) != 32 {
		t.Errorf("Key length = %d, want 32", len(key1))
	}

	if string(key1) != string(key2) {
		t.Error("Same inputs should produce same key")
	}

	// Test that different dates produce different keys
	key3 := DeriveStreamKey("2026-01-13", "license123", "fingerprint456")
	if string(key1) == string(key3) {
		t.Error("Different dates should produce different keys")
	}

	// Test that different licenses produce different keys
	key4 := DeriveStreamKey("2026-01-12", "license789", "fingerprint456")
	if string(key1) == string(key4) {
		t.Error("Different licenses should produce different keys")
	}
}

func TestGetRollingDates(t *testing.T) {
	today, tomorrow := GetRollingDates()

	// Parse dates to verify format
	todayTime, err := time.Parse("2006-01-02", today)
	if err != nil {
		t.Fatalf("Invalid today format: %v", err)
	}

	tomorrowTime, err := time.Parse("2006-01-02", tomorrow)
	if err != nil {
		t.Fatalf("Invalid tomorrow format: %v", err)
	}

	// Tomorrow should be 1 day after today
	diff := tomorrowTime.Sub(todayTime)
	if diff != 24*time.Hour {
		t.Errorf("Tomorrow should be 24h after today, got %v", diff)
	}
}

func TestWrapUnwrapCEK(t *testing.T) {
	// Generate a test CEK
	cek, err := GenerateCEK()
	if err != nil {
		t.Fatalf("GenerateCEK failed: %v", err)
	}

	// Generate a stream key
	streamKey := DeriveStreamKey("2026-01-12", "test-license", "test-fp")

	// Wrap CEK
	wrapped, err := WrapCEK(cek, streamKey)
	if err != nil {
		t.Fatalf("WrapCEK failed: %v", err)
	}

	// Unwrap CEK
	unwrapped, err := UnwrapCEK(wrapped, streamKey)
	if err != nil {
		t.Fatalf("UnwrapCEK failed: %v", err)
	}

	// Verify CEK matches
	if string(cek) != string(unwrapped) {
		t.Error("Unwrapped CEK doesn't match original")
	}

	// Wrong key should fail
	wrongKey := DeriveStreamKey("2026-01-12", "wrong-license", "test-fp")
	_, err = UnwrapCEK(wrapped, wrongKey)
	if err == nil {
		t.Error("UnwrapCEK with wrong key should fail")
	}
}

func TestEncryptDecryptV3RoundTrip(t *testing.T) {
	msg := NewMessage("Hello, this is a v3 streaming message!").
		WithSubject("V3 Test").
		WithFrom("stream@dapp.fm")

	params := &StreamParams{
		License:     "test-license-123",
		Fingerprint: "device-fp-456",
	}

	manifest := NewManifest("Test Track")
	manifest.Artist = "Test Artist"
	manifest.LicenseType = "stream"

	// Encrypt
	encrypted, err := EncryptV3(msg, params, manifest)
	if err != nil {
		t.Fatalf("EncryptV3 failed: %v", err)
	}

	// Decrypt with same params
	decrypted, header, err := DecryptV3(encrypted, params)
	if err != nil {
		t.Fatalf("DecryptV3 failed: %v", err)
	}

	// Verify message content
	if decrypted.Body != msg.Body {
		t.Errorf("Body = %q, want %q", decrypted.Body, msg.Body)
	}
	if decrypted.Subject != msg.Subject {
		t.Errorf("Subject = %q, want %q", decrypted.Subject, msg.Subject)
	}

	// Verify header
	if header.Format != FormatV3 {
		t.Errorf("Format = %q, want %q", header.Format, FormatV3)
	}
	if header.KeyMethod != KeyMethodLTHNRolling {
		t.Errorf("KeyMethod = %q, want %q", header.KeyMethod, KeyMethodLTHNRolling)
	}
	if len(header.WrappedKeys) != 2 {
		t.Errorf("WrappedKeys count = %d, want 2", len(header.WrappedKeys))
	}

	// Verify manifest
	if header.Manifest == nil {
		t.Fatal("Manifest is nil")
	}
	if header.Manifest.Title != "Test Track" {
		t.Errorf("Manifest.Title = %q, want %q", header.Manifest.Title, "Test Track")
	}
}

func TestDecryptV3WrongLicense(t *testing.T) {
	msg := NewMessage("Secret content")

	params := &StreamParams{
		License:     "correct-license",
		Fingerprint: "device-fp",
	}

	encrypted, err := EncryptV3(msg, params, nil)
	if err != nil {
		t.Fatalf("EncryptV3 failed: %v", err)
	}

	// Try to decrypt with wrong license
	wrongParams := &StreamParams{
		License:     "wrong-license",
		Fingerprint: "device-fp",
	}

	_, _, err = DecryptV3(encrypted, wrongParams)
	if err == nil {
		t.Error("DecryptV3 with wrong license should fail")
	}
	if err != ErrNoValidKey {
		t.Errorf("Error = %v, want ErrNoValidKey", err)
	}
}

func TestDecryptV3WrongFingerprint(t *testing.T) {
	msg := NewMessage("Secret content")

	params := &StreamParams{
		License:     "test-license",
		Fingerprint: "correct-fingerprint",
	}

	encrypted, err := EncryptV3(msg, params, nil)
	if err != nil {
		t.Fatalf("EncryptV3 failed: %v", err)
	}

	// Try to decrypt with wrong fingerprint
	wrongParams := &StreamParams{
		License:     "test-license",
		Fingerprint: "wrong-fingerprint",
	}

	_, _, err = DecryptV3(encrypted, wrongParams)
	if err == nil {
		t.Error("DecryptV3 with wrong fingerprint should fail")
	}
}

func TestEncryptV3WithAttachment(t *testing.T) {
	msg := NewMessage("Message with attachment")
	msg.AddBinaryAttachment("test.mp3", []byte("fake audio data here"), "audio/mpeg")

	params := &StreamParams{
		License:     "test-license",
		Fingerprint: "test-fp",
	}

	encrypted, err := EncryptV3(msg, params, nil)
	if err != nil {
		t.Fatalf("EncryptV3 failed: %v", err)
	}

	decrypted, _, err := DecryptV3(encrypted, params)
	if err != nil {
		t.Fatalf("DecryptV3 failed: %v", err)
	}

	// Verify attachment
	if len(decrypted.Attachments) != 1 {
		t.Fatalf("Attachment count = %d, want 1", len(decrypted.Attachments))
	}

	att := decrypted.GetAttachment("test.mp3")
	if att == nil {
		t.Fatal("Attachment not found")
	}
	if att.MimeType != "audio/mpeg" {
		t.Errorf("MimeType = %q, want %q", att.MimeType, "audio/mpeg")
	}
}

func TestEncryptV3RequiresLicense(t *testing.T) {
	msg := NewMessage("Test")

	// Nil params
	_, err := EncryptV3(msg, nil, nil)
	if err != ErrLicenseRequired {
		t.Errorf("Error = %v, want ErrLicenseRequired", err)
	}

	// Empty license
	_, err = EncryptV3(msg, &StreamParams{}, nil)
	if err != ErrLicenseRequired {
		t.Errorf("Error = %v, want ErrLicenseRequired", err)
	}
}

func TestCadencePeriods(t *testing.T) {
	// Test at a known time: 2026-01-12 15:30:00 UTC
	testTime := time.Date(2026, 1, 12, 15, 30, 0, 0, time.UTC)

	tests := []struct {
		cadence         Cadence
		expectedCurrent string
		expectedNext    string
	}{
		{CadenceDaily, "2026-01-12", "2026-01-13"},
		{CadenceHalfDay, "2026-01-12-PM", "2026-01-13-AM"},
		{CadenceQuarter, "2026-01-12-12", "2026-01-12-18"},
		{CadenceHourly, "2026-01-12-15", "2026-01-12-16"},
	}

	for _, tc := range tests {
		t.Run(string(tc.cadence), func(t *testing.T) {
			current, next := GetRollingPeriods(tc.cadence, testTime)
			if current != tc.expectedCurrent {
				t.Errorf("current = %q, want %q", current, tc.expectedCurrent)
			}
			if next != tc.expectedNext {
				t.Errorf("next = %q, want %q", next, tc.expectedNext)
			}
		})
	}
}

func TestCadenceHalfDayAM(t *testing.T) {
	// Test in the morning
	testTime := time.Date(2026, 1, 12, 9, 0, 0, 0, time.UTC)
	current, next := GetRollingPeriods(CadenceHalfDay, testTime)

	if current != "2026-01-12-AM" {
		t.Errorf("current = %q, want %q", current, "2026-01-12-AM")
	}
	if next != "2026-01-12-PM" {
		t.Errorf("next = %q, want %q", next, "2026-01-12-PM")
	}
}

func TestCadenceQuarterBoundary(t *testing.T) {
	// Test at 23:00 - should wrap to next day
	testTime := time.Date(2026, 1, 12, 23, 0, 0, 0, time.UTC)
	current, next := GetRollingPeriods(CadenceQuarter, testTime)

	if current != "2026-01-12-18" {
		t.Errorf("current = %q, want %q", current, "2026-01-12-18")
	}
	if next != "2026-01-13-00" {
		t.Errorf("next = %q, want %q", next, "2026-01-13-00")
	}
}

func TestEncryptDecryptV3WithCadence(t *testing.T) {
	cadences := []Cadence{CadenceDaily, CadenceHalfDay, CadenceQuarter, CadenceHourly}

	for _, cadence := range cadences {
		t.Run(string(cadence), func(t *testing.T) {
			msg := NewMessage("Testing " + string(cadence) + " cadence")

			params := &StreamParams{
				License:     "cadence-test-license",
				Fingerprint: "cadence-test-fp",
				Cadence:     cadence,
			}

			// Encrypt
			encrypted, err := EncryptV3(msg, params, nil)
			if err != nil {
				t.Fatalf("EncryptV3 failed: %v", err)
			}

			// Decrypt with same params
			decrypted, header, err := DecryptV3(encrypted, params)
			if err != nil {
				t.Fatalf("DecryptV3 failed: %v", err)
			}

			if decrypted.Body != msg.Body {
				t.Errorf("Body = %q, want %q", decrypted.Body, msg.Body)
			}

			// Verify cadence in header
			if header.Cadence != cadence {
				t.Errorf("Cadence = %q, want %q", header.Cadence, cadence)
			}
		})
	}
}

func TestRollingKeyWindow(t *testing.T) {
	// This test verifies that both today's and tomorrow's keys work
	msg := NewMessage("Rolling window test")

	// Create params
	params := &StreamParams{
		License:     "rolling-test-license",
		Fingerprint: "rolling-test-fp",
	}

	// Encrypt with current time
	encrypted, err := EncryptV3(msg, params, nil)
	if err != nil {
		t.Fatalf("EncryptV3 failed: %v", err)
	}

	// Should decrypt successfully (within rolling window)
	decrypted, header, err := DecryptV3(encrypted, params)
	if err != nil {
		t.Fatalf("DecryptV3 failed: %v", err)
	}

	if decrypted.Body != msg.Body {
		t.Errorf("Body = %q, want %q", decrypted.Body, msg.Body)
	}

	// Verify we have both today and tomorrow keys
	today, tomorrow := GetRollingDates()
	hasToday := false
	hasTomorrow := false
	for _, wk := range header.WrappedKeys {
		if wk.Date == today {
			hasToday = true
		}
		if wk.Date == tomorrow {
			hasTomorrow = true
		}
	}
	if !hasToday {
		t.Error("Missing today's wrapped key")
	}
	if !hasTomorrow {
		t.Error("Missing tomorrow's wrapped key")
	}
}
