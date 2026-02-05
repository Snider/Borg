package stmf

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Snider/Enchantrix/pkg/keyserver"
	"github.com/Snider/Enchantrix/pkg/keystore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSTMFKeyServer(t *testing.T) *keyserver.Server {
	t.Helper()
	dir := t.TempDir()
	store, err := keystore.Create(filepath.Join(dir, "keys.trix"), "test-master")
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return keyserver.NewServer(store)
}

func TestGenerateKeyPairKS(t *testing.T) {
	ctx := context.Background()
	ks := newTestSTMFKeyServer(t)

	pubKey, keyID, err := GenerateKeyPairKS(ctx, "stmf-server", ks)
	require.NoError(t, err)
	assert.Len(t, pubKey, 32)
	assert.NotEmpty(t, keyID)

	// Public key should be retrievable
	pubKey2, err := ks.GetPublicKey(ctx, keyID)
	require.NoError(t, err)
	assert.Equal(t, pubKey, pubKey2)
}

func TestSTMFDecryptKS(t *testing.T) {
	ctx := context.Background()
	ks := newTestSTMFKeyServer(t)

	// Generate server keypair via keyserver
	pubKey, keyID, err := GenerateKeyPairKS(ctx, "stmf-server", ks)
	require.NoError(t, err)

	// Client encrypts with public key (no keyserver needed)
	formData := NewFormData().
		AddField("username", "alice").
		AddFieldWithType("password", "secret123", "password")

	encrypted, err := Encrypt(formData, pubKey)
	require.NoError(t, err)

	// Server decrypts via keyserver — private key never leaves
	decrypted, err := DecryptKS(ctx, encrypted, keyID, ks)
	require.NoError(t, err)

	assert.Equal(t, "alice", decrypted.Get("username"))
	assert.Equal(t, "secret123", decrypted.Get("password"))
}

func TestSTMFKeyRotation(t *testing.T) {
	ctx := context.Background()
	ks := newTestSTMFKeyServer(t)

	// Generate first keypair
	pubKey1, keyID1, err := GenerateKeyPairKS(ctx, "server-v1", ks)
	require.NoError(t, err)

	// Generate second keypair (rotation)
	pubKey2, keyID2, err := GenerateKeyPairKS(ctx, "server-v2", ks)
	require.NoError(t, err)

	assert.NotEqual(t, pubKey1, pubKey2)

	// Encrypt with old key
	form1 := NewFormData().AddField("version", "1")
	enc1, err := Encrypt(form1, pubKey1)
	require.NoError(t, err)

	// Encrypt with new key
	form2 := NewFormData().AddField("version", "2")
	enc2, err := Encrypt(form2, pubKey2)
	require.NoError(t, err)

	// Old key still decrypts old forms
	dec1, err := DecryptKS(ctx, enc1, keyID1, ks)
	require.NoError(t, err)
	assert.Equal(t, "1", dec1.Get("version"))

	// New key decrypts new forms
	dec2, err := DecryptKS(ctx, enc2, keyID2, ks)
	require.NoError(t, err)
	assert.Equal(t, "2", dec2.Get("version"))

	// New key cannot decrypt old forms (different ECDH shared secret)
	_, err = DecryptKS(ctx, enc1, keyID2, ks)
	assert.Error(t, err)
}

func TestSTMFPrivateKeyNeverLeaves(t *testing.T) {
	ctx := context.Background()
	ks := newTestSTMFKeyServer(t)

	_, keyID, err := GenerateKeyPairKS(ctx, "secure-key", ks)
	require.NoError(t, err)

	// ListKeys should not expose key data
	keys, err := ks.ListKeys(ctx)
	require.NoError(t, err)
	for _, k := range keys {
		assert.Nil(t, k.KeyData, "ListKeys must not expose key data")
	}

	// GetPublicKey only returns public component
	pubKey, err := ks.GetPublicKey(ctx, keyID)
	require.NoError(t, err)
	assert.Len(t, pubKey, 32) // public key only

	// There's no GetPrivateKey method on the interface — by design
}
