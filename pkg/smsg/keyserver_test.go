package smsg

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Snider/Enchantrix/pkg/keyserver"
	"github.com/Snider/Enchantrix/pkg/keystore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSMSGKeyServer(t *testing.T) *keyserver.Server {
	t.Helper()
	dir := t.TempDir()
	store, err := keystore.Create(filepath.Join(dir, "keys.trix"), "test-master")
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return keyserver.NewServer(store)
}

func TestEncryptDecryptKSRoundTrip(t *testing.T) {
	ctx := context.Background()
	ks := newTestSMSGKeyServer(t)

	keyID, err := ks.ImportPassword(ctx, "smsg-password", "smsg-key")
	require.NoError(t, err)

	msg := NewMessage("Hello, keyserver SMSG!")
	msg.WithSubject("Test").WithFrom("alice")

	encrypted, err := EncryptKS(ctx, msg, keyID, ks)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	decrypted, err := DecryptKS(ctx, encrypted, keyID, ks)
	require.NoError(t, err)
	assert.Equal(t, "Hello, keyserver SMSG!", decrypted.Body)
	assert.Equal(t, "Test", decrypted.Subject)
	assert.Equal(t, "alice", decrypted.From)
}

func TestKSCompatWithOldAPI(t *testing.T) {
	ctx := context.Background()
	ks := newTestSMSGKeyServer(t)

	password := "compat-password"
	keyID, err := ks.ImportPassword(ctx, password, "compat")
	require.NoError(t, err)

	// Encrypt with old API
	msg := NewMessage("compatibility test")
	encrypted, err := Encrypt(msg, password)
	require.NoError(t, err)

	// Decrypt with keyserver
	decrypted, err := DecryptKS(ctx, encrypted, keyID, ks)
	require.NoError(t, err)
	assert.Equal(t, "compatibility test", decrypted.Body)
}

func TestKSEncryptOldDecrypt(t *testing.T) {
	ctx := context.Background()
	ks := newTestSMSGKeyServer(t)

	password := "compat-password-2"
	keyID, err := ks.ImportPassword(ctx, password, "compat2")
	require.NoError(t, err)

	// Encrypt with keyserver
	msg := NewMessage("keyserver encrypted")
	encrypted, err := EncryptKS(ctx, msg, keyID, ks)
	require.NoError(t, err)

	// Decrypt with old API
	decrypted, err := Decrypt(encrypted, password)
	require.NoError(t, err)
	assert.Equal(t, "keyserver encrypted", decrypted.Body)
}

func TestEncryptKSEmptyMessage(t *testing.T) {
	ctx := context.Background()
	ks := newTestSMSGKeyServer(t)

	keyID, _ := ks.ImportPassword(ctx, "pass", "key")

	msg := &Message{}
	_, err := EncryptKS(ctx, msg, keyID, ks)
	assert.ErrorIs(t, err, ErrEmptyMessage)
}

func TestV3KSRoundTrip(t *testing.T) {
	ctx := context.Background()
	ks := newTestSMSGKeyServer(t)

	params := V3ParamsKS{
		License:     "test-license-123",
		Fingerprint: "device-fp-456",
		Cadence:     CadenceDaily,
	}

	msg := NewMessage("V3 keyserver streaming test")

	encrypted, err := EncryptV3KS(ctx, msg, params, nil, ks)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	decrypted, header, err := DecryptV3KS(ctx, encrypted, params, ks)
	require.NoError(t, err)
	assert.Equal(t, "V3 keyserver streaming test", decrypted.Body)
	assert.Equal(t, FormatV3, header.Format)
	assert.Equal(t, KeyMethodLTHNRolling, header.KeyMethod)
}

func TestV3KSWithManifest(t *testing.T) {
	ctx := context.Background()
	ks := newTestSMSGKeyServer(t)

	params := V3ParamsKS{
		License:     "license-xyz",
		Fingerprint: "fp-abc",
	}

	manifest := NewManifest("Test Track")
	manifest.Artist = "Test Artist"

	msg := NewMessage("manifest test")

	encrypted, err := EncryptV3KS(ctx, msg, params, manifest, ks)
	require.NoError(t, err)

	_, header, err := DecryptV3KS(ctx, encrypted, params, ks)
	require.NoError(t, err)
	assert.NotNil(t, header.Manifest)
	assert.Equal(t, "Test Track", header.Manifest.Title)
}

func TestV3KSNoLicense(t *testing.T) {
	ctx := context.Background()
	ks := newTestSMSGKeyServer(t)

	params := V3ParamsKS{Fingerprint: "fp"}
	msg := NewMessage("test")

	_, err := EncryptV3KS(ctx, msg, params, nil, ks)
	assert.ErrorIs(t, err, ErrLicenseRequired)
}
