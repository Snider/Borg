package tim

import (
	"context"
	"encoding/binary"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Enchantrix/pkg/keyserver"
	"github.com/Snider/Enchantrix/pkg/trix"
)

// ToSigilKS encrypts the TIM using the keyserver. The key material never
// leaves the keyserver — only the key ID is passed.
// The output format matches ToSigil: a Trix container with "STIM" magic.
func (m *TerminalIsolationMatrix) ToSigilKS(ctx context.Context, keyID string, ks keyserver.KeyServer) ([]byte, error) {
	if m.Config == nil {
		return nil, ErrConfigIsNil
	}

	rootfsTar, err := m.RootFS.ToTar()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize rootfs: %w", err)
	}

	// Keyserver encrypts config+rootfs atomically — key never leaves
	payload, err := ks.EncryptTIM(ctx, keyID, m.Config, rootfsTar)
	if err != nil {
		return nil, fmt.Errorf("keyserver encrypt TIM: %w", err)
	}

	// Parse encrypted config size from payload for header metadata
	configSize := binary.BigEndian.Uint32(payload[:4])
	rootfsSize := len(payload) - 4 - int(configSize)

	// Wrap in Trix container with same header format as ToSigil
	t := &trix.Trix{
		Header: map[string]interface{}{
			"encryption_algorithm": "chacha20poly1305",
			"tim":                  true,
			"config_size":          configSize,
			"rootfs_size":          rootfsSize,
			"version":              "1.0",
		},
		Payload: payload,
	}

	return trix.Encode(t, "STIM", nil)
}

// FromSigilKS decrypts and deserializes a .stim file using the keyserver.
// The key material never leaves the keyserver.
func FromSigilKS(data []byte, keyID string, ks keyserver.KeyServer) (*TerminalIsolationMatrix, error) {
	return FromSigilKSCtx(context.Background(), data, keyID, ks)
}

// FromSigilKSCtx decrypts and deserializes a .stim file using the keyserver
// with a context.
func FromSigilKSCtx(ctx context.Context, data []byte, keyID string, ks keyserver.KeyServer) (*TerminalIsolationMatrix, error) {
	t, err := trix.Decode(data, "STIM", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decode stim: %w", err)
	}

	// Keyserver decrypts config+rootfs atomically — key never leaves
	config, rootfsTar, err := ks.DecryptTIM(ctx, keyID, t.Payload)
	if err != nil {
		return nil, fmt.Errorf("keyserver decrypt TIM: %w", err)
	}

	// Reconstruct DataNode from decrypted rootfs tar
	rootfs, err := datanode.FromTar(rootfsTar)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rootfs: %w", err)
	}

	return &TerminalIsolationMatrix{
		Config: config,
		RootFS: rootfs,
	}, nil
}

// CacheKS provides encrypted TIM storage using a keyserver.
// The key material never leaves the keyserver — only the key ID is referenced.
type CacheKS struct {
	Dir   string
	KeyID string
	KS    keyserver.KeyServer
}

// NewCacheKS creates a keyserver-backed TIM cache.
func NewCacheKS(dir string, keyID string, ks keyserver.KeyServer) (*CacheKS, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &CacheKS{Dir: dir, KeyID: keyID, KS: ks}, nil
}

// Store encrypts and saves a TIM to the cache via keyserver.
func (c *CacheKS) Store(ctx context.Context, name string, m *TerminalIsolationMatrix) error {
	data, err := m.ToSigilKS(ctx, c.KeyID, c.KS)
	if err != nil {
		return err
	}
	path := filepath.Join(c.Dir, name+".stim")
	return os.WriteFile(path, data, 0600)
}

// Load retrieves and decrypts a TIM from the cache via keyserver.
func (c *CacheKS) Load(ctx context.Context, name string) (*TerminalIsolationMatrix, error) {
	path := filepath.Join(c.Dir, name+".stim")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return FromSigilKSCtx(ctx, data, c.KeyID, c.KS)
}

// RunEncryptedKS runs an encrypted .stim file, decrypting via keyserver.
// The key material never leaves the keyserver.
func RunEncryptedKS(ctx context.Context, stimPath, keyID string, ks keyserver.KeyServer) error {
	data, err := os.ReadFile(stimPath)
	if err != nil {
		return fmt.Errorf("failed to read stim file: %w", err)
	}

	m, err := FromSigilKSCtx(ctx, data, keyID, ks)
	if err != nil {
		return fmt.Errorf("failed to decrypt stim: %w", err)
	}

	tempDir, err := os.MkdirTemp("", "borg-run-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := os.WriteFile(filepath.Join(tempDir, "config.json"), m.Config, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	rootfsPath := filepath.Join(tempDir, "rootfs")
	if err := os.MkdirAll(rootfsPath, 0755); err != nil {
		return fmt.Errorf("failed to create rootfs dir: %w", err)
	}

	err = m.RootFS.Walk(".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		target := filepath.Join(rootfsPath, path)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return m.RootFS.CopyFile(path, target, 0600)
	})
	if err != nil {
		return fmt.Errorf("failed to extract rootfs: %w", err)
	}

	cmd := ExecCommand("runc", "run", "-b", tempDir, "borg-container")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
