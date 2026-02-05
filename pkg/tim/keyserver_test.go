package tim

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Enchantrix/pkg/keyserver"
	"github.com/Snider/Enchantrix/pkg/keystore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestKeyServer(t *testing.T) *keyserver.Server {
	t.Helper()
	dir := t.TempDir()
	store, err := keystore.Create(filepath.Join(dir, "keys.trix"), "test-master")
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return keyserver.NewServer(store)
}

func TestToFromSigilKS(t *testing.T) {
	ctx := context.Background()
	ks := newTestKeyServer(t)

	keyID, err := ks.ImportPassword(ctx, "test-password", "tim-key")
	require.NoError(t, err)

	// Build a TIM
	m, err := New()
	require.NoError(t, err)
	m.RootFS.AddData("hello.txt", []byte("Hello from the keyserver!"))

	// Encrypt via keyserver
	stim, err := m.ToSigilKS(ctx, keyID, ks)
	require.NoError(t, err)
	assert.NotEmpty(t, stim)

	// Decrypt via keyserver
	restored, err := FromSigilKS(stim, keyID, ks)
	require.NoError(t, err)

	// Verify config round-trips
	assert.Equal(t, m.Config, restored.Config)

	// Verify rootfs round-trips
	f, err := restored.RootFS.Open("hello.txt")
	require.NoError(t, err)
	defer f.Close()
	info, _ := f.Stat()
	buf := make([]byte, info.Size())
	f.Read(buf)
	assert.Equal(t, "Hello from the keyserver!", string(buf))
}

func TestToSigilKSMatchesOldFormat(t *testing.T) {
	ctx := context.Background()
	ks := newTestKeyServer(t)

	password := "compat-test"
	keyID, err := ks.ImportPassword(ctx, password, "compat")
	require.NoError(t, err)

	m, err := New()
	require.NoError(t, err)
	m.RootFS.AddData("file.txt", []byte("content"))

	// Encrypt with keyserver
	stimKS, err := m.ToSigilKS(ctx, keyID, ks)
	require.NoError(t, err)

	// Decrypt with old password API (should work — same key derivation)
	restored, err := FromSigil(stimKS, password)
	require.NoError(t, err)
	assert.Equal(t, m.Config, restored.Config)
}

func TestOldFormatDecryptsWithKS(t *testing.T) {
	ctx := context.Background()
	ks := newTestKeyServer(t)

	password := "compat-test-2"
	keyID, err := ks.ImportPassword(ctx, password, "compat2")
	require.NoError(t, err)

	m, err := New()
	require.NoError(t, err)
	m.RootFS.AddData("data.bin", []byte{0xDE, 0xAD, 0xBE, 0xEF})

	// Encrypt with old API
	stim, err := m.ToSigil(password)
	require.NoError(t, err)

	// Decrypt with keyserver
	restored, err := FromSigilKS(stim, keyID, ks)
	require.NoError(t, err)
	assert.Equal(t, m.Config, restored.Config)
}

func TestCacheKS(t *testing.T) {
	ctx := context.Background()
	ks := newTestKeyServer(t)
	dir := t.TempDir()

	keyID, err := ks.ImportPassword(ctx, "cache-pass", "cache-key")
	require.NoError(t, err)

	cache, err := NewCacheKS(dir, keyID, ks)
	require.NoError(t, err)

	// Create TIM
	m, err := New()
	require.NoError(t, err)
	m.RootFS.AddData("cached.txt", []byte("cached content"))

	// Store
	err = cache.Store(ctx, "test-tim", m)
	require.NoError(t, err)

	// Load
	loaded, err := cache.Load(ctx, "test-tim")
	require.NoError(t, err)

	assert.Equal(t, m.Config, loaded.Config)

	f, err := loaded.RootFS.Open("cached.txt")
	require.NoError(t, err)
	defer f.Close()
	info, _ := f.Stat()
	buf := make([]byte, info.Size())
	f.Read(buf)
	assert.Equal(t, "cached content", string(buf))
}

func TestToSigilKSNilConfig(t *testing.T) {
	ctx := context.Background()
	ks := newTestKeyServer(t)

	keyID, _ := ks.ImportPassword(ctx, "pass", "key")

	m := &TerminalIsolationMatrix{
		Config: nil,
		RootFS: datanode.New(),
	}

	_, err := m.ToSigilKS(ctx, keyID, ks)
	assert.ErrorIs(t, err, ErrConfigIsNil)
}
