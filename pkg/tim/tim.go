package tim

import (
	"archive/tar"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/Snider/Borg/pkg/datanode"
	borgtrix "github.com/Snider/Borg/pkg/trix"
	"github.com/Snider/Enchantrix/pkg/enchantrix"
	"github.com/Snider/Enchantrix/pkg/trix"
)

var (
	ErrDataNodeRequired   = errors.New("datanode is required")
	ErrConfigIsNil        = errors.New("config is nil")
	ErrPasswordRequired   = errors.New("password is required for encryption")
	ErrInvalidStimPayload = errors.New("invalid stim payload")
	ErrDecryptionFailed   = errors.New("decryption failed (wrong password?)")
)

// TerminalIsolationMatrix represents a runc bundle.
type TerminalIsolationMatrix struct {
	Config []byte
	RootFS *datanode.DataNode
}

// New creates a new, empty TerminalIsolationMatrix.
func New() (*TerminalIsolationMatrix, error) {
	// Use the default runc spec as a starting point.
	// This can be customized later.
	spec, err := defaultConfig()
	if err != nil {
		return nil, err
	}

	specBytes, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}

	return &TerminalIsolationMatrix{
		Config: specBytes,
		RootFS: datanode.New(),
	}, nil
}

// FromDataNode creates a new TerminalIsolationMatrix from a DataNode.
func FromDataNode(dn *datanode.DataNode) (*TerminalIsolationMatrix, error) {
	if dn == nil {
		return nil, ErrDataNodeRequired
	}
	m, err := New()
	if err != nil {
		return nil, err
	}
	m.RootFS = dn
	return m, nil
}

// FromTar creates a TerminalIsolationMatrix from a tarball.
// The tarball must contain config.json and a rootfs/ directory.
func FromTar(data []byte) (*TerminalIsolationMatrix, error) {
	tr := tar.NewReader(bytes.NewReader(data))

	var config []byte
	rootfs := datanode.New()

	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}

		if hdr.Name == "config.json" {
			config, err = io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("failed to read config.json: %w", err)
			}
		} else if strings.HasPrefix(hdr.Name, "rootfs/") && hdr.Typeflag == tar.TypeReg {
			// Strip "rootfs/" prefix
			name := strings.TrimPrefix(hdr.Name, "rootfs/")
			if name == "" {
				continue
			}
			content, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("failed to read %s: %w", hdr.Name, err)
			}
			rootfs.AddData(name, content)
		}
	}

	if config == nil {
		return nil, ErrConfigIsNil
	}

	return &TerminalIsolationMatrix{
		Config: config,
		RootFS: rootfs,
	}, nil
}

// ToTar serializes the TerminalIsolationMatrix to a tarball.
func (m *TerminalIsolationMatrix) ToTar() ([]byte, error) {
	if m.Config == nil {
		return nil, ErrConfigIsNil
	}
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	// Add the config.json file.
	hdr := &tar.Header{
		Name: "config.json",
		Mode: 0600,
		Size: int64(len(m.Config)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write(m.Config); err != nil {
		return nil, err
	}

	// Add the rootfs directory.
	hdr = &tar.Header{
		Name:     "rootfs/",
		Mode:     0755,
		Typeflag: tar.TypeDir,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}

	// Add the rootfs files.
	err := m.RootFS.Walk(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// If the root directory doesn't exist (i.e. empty datanode), it's not an error.
			if path == "." && errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}

		if d.IsDir() {
			return nil
		}

		file, err := m.RootFS.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		info, err := file.Stat()
		if err != nil {
			return err
		}

		hdr := &tar.Header{
			Name: "rootfs/" + path,
			Mode: 0600,
			Size: info.Size(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(file); err != nil {
			return err
		}

		if _, err := tw.Write(buf.Bytes()); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ToSigil serializes and encrypts the TIM to .stim format using ChaChaPolySigil.
// Config and RootFS are encrypted separately.
// The output format is a Trix container with "STIM" magic containing:
// - Header: {"encryption_algorithm": "chacha20poly1305", "tim": true}
// - Payload: [config_size(4 bytes)][encrypted_config][encrypted_rootfs]
func (m *TerminalIsolationMatrix) ToSigil(password string) ([]byte, error) {
	if password == "" {
		return nil, ErrPasswordRequired
	}
	if m.Config == nil {
		return nil, ErrConfigIsNil
	}

	key := borgtrix.DeriveKey(password)
	sigil, err := enchantrix.NewChaChaPolySigil(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create sigil: %w", err)
	}

	// Encrypt config
	encConfig, err := sigil.In(m.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt config: %w", err)
	}

	// Get rootfs as tar
	rootfsTar, err := m.RootFS.ToTar()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize rootfs: %w", err)
	}

	// Encrypt rootfs
	encRootFS, err := sigil.In(rootfsTar)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt rootfs: %w", err)
	}

	// Build payload: [config_size(4 bytes)][encrypted_config][encrypted_rootfs]
	payload := make([]byte, 4+len(encConfig)+len(encRootFS))
	binary.BigEndian.PutUint32(payload[:4], uint32(len(encConfig)))
	copy(payload[4:4+len(encConfig)], encConfig)
	copy(payload[4+len(encConfig):], encRootFS)

	// Create trix container
	t := &trix.Trix{
		Header: map[string]interface{}{
			"encryption_algorithm": "chacha20poly1305",
			"tim":                  true,
			"config_size":          len(encConfig),
			"rootfs_size":          len(encRootFS),
			"version":              "1.0",
		},
		Payload: payload,
	}

	return trix.Encode(t, "STIM", nil)
}

// FromSigil decrypts and deserializes a .stim file into a TerminalIsolationMatrix.
func FromSigil(data []byte, password string) (*TerminalIsolationMatrix, error) {
	if password == "" {
		return nil, ErrPasswordRequired
	}

	// Decode the trix container
	t, err := trix.Decode(data, "STIM", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decode stim: %w", err)
	}

	key := borgtrix.DeriveKey(password)
	sigil, err := enchantrix.NewChaChaPolySigil(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create sigil: %w", err)
	}

	// Parse payload structure
	if len(t.Payload) < 4 {
		return nil, ErrInvalidStimPayload
	}
	configSize := binary.BigEndian.Uint32(t.Payload[:4])

	if len(t.Payload) < int(4+configSize) {
		return nil, ErrInvalidStimPayload
	}

	encConfig := t.Payload[4 : 4+configSize]
	encRootFS := t.Payload[4+configSize:]

	// Decrypt config
	config, err := sigil.Out(encConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	// Decrypt rootfs
	rootfsTar, err := sigil.Out(encRootFS)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	// Reconstruct DataNode from tar
	rootfs, err := datanode.FromTar(rootfsTar)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rootfs: %w", err)
	}

	return &TerminalIsolationMatrix{
		Config: config,
		RootFS: rootfs,
	}, nil
}
