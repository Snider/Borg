# RFC-004: Terminal Isolation Matrix (TIM)

**Status**: Draft
**Author**: [Snider](https://github.com/Snider/)
**Created**: 2026-01-13
**License**: EUPL-1.2
**Depends On**: RFC-003

---

## Abstract

TIM (Terminal Isolation Matrix) is an OCI-compatible container bundle format. It packages a runtime configuration with a root filesystem (DataNode) for execution via runc or compatible runtimes.

## 1. Overview

TIM provides:
- OCI runtime-spec compatible bundles
- Portable container packaging
- Integration with DataNode filesystem
- Encryption via STIM (RFC-005)

## 2. Implementation

### 2.1 Core Type

```go
// pkg/tim/tim.go:28-32
type TerminalIsolationMatrix struct {
    Config []byte              // Raw OCI runtime specification (JSON)
    RootFS *datanode.DataNode  // In-memory filesystem
}
```

### 2.2 Error Variables

```go
var (
    ErrDataNodeRequired   = errors.New("datanode is required")
    ErrConfigIsNil        = errors.New("config is nil")
    ErrPasswordRequired   = errors.New("password is required for encryption")
    ErrInvalidStimPayload = errors.New("invalid stim payload")
    ErrDecryptionFailed   = errors.New("decryption failed (wrong password?)")
)
```

## 3. Public API

### 3.1 Constructors

```go
// Create empty TIM with default config
func New() (*TerminalIsolationMatrix, error)

// Wrap existing DataNode into TIM
func FromDataNode(dn *DataNode) (*TerminalIsolationMatrix, error)

// Deserialize from tar archive
func FromTar(data []byte) (*TerminalIsolationMatrix, error)
```

### 3.2 Serialization

```go
// Serialize to tar archive
func (m *TerminalIsolationMatrix) ToTar() ([]byte, error)

// Encrypt to STIM format (ChaCha20-Poly1305)
func (m *TerminalIsolationMatrix) ToSigil(password string) ([]byte, error)
```

### 3.3 Decryption

```go
// Decrypt from STIM format
func FromSigil(data []byte, password string) (*TerminalIsolationMatrix, error)
```

### 3.4 Execution

```go
// Run plain .tim file with runc
func Run(timPath string) error

// Decrypt and run .stim file
func RunEncrypted(stimPath, password string) error
```

## 4. Tar Archive Structure

### 4.1 Layout

```
config.json             (root level, mode 0600)
rootfs/                 (directory, mode 0755)
rootfs/bin/app          (files within rootfs/)
rootfs/etc/config
...
```

### 4.2 Serialization (ToTar)

```go
// pkg/tim/tim.go:111-195
func (m *TerminalIsolationMatrix) ToTar() ([]byte, error) {
    // 1. Write config.json header (size = len(m.Config), mode 0600)
    // 2. Write config.json content
    // 3. Write rootfs/ directory entry (TypeDir, mode 0755)
    // 4. Walk m.RootFS depth-first
    // 5. For each file: tar entry with name "rootfs/" + path, mode 0600
}
```

### 4.3 Deserialization (FromTar)

```go
func FromTar(data []byte) (*TerminalIsolationMatrix, error) {
    // 1. Parse tar entries
    // 2. "config.json" → stored as raw bytes in Config
    // 3. "rootfs/*" prefix → stripped and added to DataNode
    // 4. Error if config.json missing (ErrConfigIsNil)
}
```

## 5. OCI Config

### 5.1 Default Config

The `New()` function creates a TIM with a default config from `pkg/tim/config.go`:

```go
func defaultConfig() (*trix.Trix, error) {
    return &trix.Trix{Header: make(map[string]interface{})}, nil
}
```

**Note**: The default config is minimal. Applications should populate the Config field with a proper OCI runtime spec.

### 5.2 OCI Runtime Spec Example

```json
{
  "ociVersion": "1.0.2",
  "process": {
    "terminal": false,
    "user": {"uid": 0, "gid": 0},
    "args": ["/bin/app"],
    "env": ["PATH=/usr/bin:/bin"],
    "cwd": "/"
  },
  "root": {
    "path": "rootfs",
    "readonly": true
  },
  "mounts": [],
  "linux": {
    "namespaces": [
      {"type": "pid"},
      {"type": "network"},
      {"type": "mount"}
    ]
  }
}
```

## 6. Execution Flow

### 6.1 Plain TIM (Run)

```go
// pkg/tim/run.go:18-74
func Run(timPath string) error {
    // 1. Create temporary directory (borg-run-*)
    // 2. Extract tar entry-by-entry
    //    - Security: Path traversal check (prevents ../)
    //    - Validates: target = Clean(target) within tempDir
    // 3. Create directories as needed (0755)
    // 4. Write files with 0600 permissions
    // 5. Execute: runc run -b <tempDir> borg-container
    // 6. Stream stdout/stderr directly
    // 7. Return exit code
}
```

### 6.2 Encrypted TIM (RunEncrypted)

```go
// pkg/tim/run.go:79-134
func RunEncrypted(stimPath, password string) error {
    // 1. Read encrypted .stim file
    // 2. Decrypt using FromSigil() with password
    // 3. Create temporary directory (borg-run-*)
    // 4. Write config.json to tempDir
    // 5. Create rootfs/ subdirectory
    // 6. Walk DataNode and extract all files to rootfs/
    //    - Uses CopyFile() with 0600 permissions
    // 7. Execute: runc run -b <tempDir> borg-container
    // 8. Stream stdout/stderr
    // 9. Clean up temp directory (defer os.RemoveAll)
    // 10. Return exit code
}
```

### 6.3 Security Controls

| Control | Implementation |
|---------|----------------|
| Path traversal | `filepath.Clean()` + prefix validation |
| Temp cleanup | `defer os.RemoveAll(tempDir)` |
| File permissions | Hardcoded 0600 (files), 0755 (dirs) |
| Test injection | `ExecCommand` variable for mocking runc |

## 7. Cache API

### 7.1 Cache Structure

```go
// pkg/tim/cache.go
type Cache struct {
    Dir      string  // Directory path for storage
    Password string  // Shared password for all TIMs
}
```

### 7.2 Cache Operations

```go
// Create cache with master password
func NewCache(dir, password string) (*Cache, error)

// Store TIM (encrypted automatically as .stim)
func (c *Cache) Store(name string, m *TerminalIsolationMatrix) error

// Load TIM (decrypted automatically)
func (c *Cache) Load(name string) (*TerminalIsolationMatrix, error)

// Delete cached TIM
func (c *Cache) Delete(name string) error

// Check if TIM exists
func (c *Cache) Exists(name string) bool

// List all cached TIM names
func (c *Cache) List() ([]string, error)

// Load and execute cached TIM
func (c *Cache) Run(name string) error

// Get file size of cached .stim
func (c *Cache) Size(name string) (int64, error)
```

### 7.3 Cache Directory Structure

```
cache/
├── mycontainer.stim     (encrypted)
├── another.stim         (encrypted)
└── ...
```

- All TIMs stored as `.stim` files (encrypted)
- Single password protects entire cache
- Directory created with 0700 permissions
- Files stored with 0600 permissions

## 8. CLI Usage

```bash
# Compile Borgfile to TIM
borg compile -f Borgfile -o container.tim

# Compile with encryption
borg compile -f Borgfile -e "password" -o container.stim

# Run plain TIM
borg run container.tim

# Run encrypted TIM
borg run container.stim -p "password"

# Decode (extract) to tar
borg decode container.stim -p "password" --i-am-in-isolation -o container.tar

# Inspect metadata without decrypting
borg inspect container.stim
```

## 9. Implementation Reference

- TIM core: `pkg/tim/tim.go`
- Execution: `pkg/tim/run.go`
- Cache: `pkg/tim/cache.go`
- Config: `pkg/tim/config.go`
- Tests: `pkg/tim/tim_test.go`, `pkg/tim/run_test.go`, `pkg/tim/cache_test.go`

## 10. Security Considerations

1. **Path traversal prevention**: `filepath.Clean()` + prefix validation
2. **Permission hardcoding**: 0600 files, 0755 directories
3. **Secure cleanup**: `defer os.RemoveAll()` on temp directories
4. **Command injection prevention**: `ExecCommand` variable (no shell)
5. **Config validation**: Validate OCI spec before execution

## 11. OCI Compatibility

TIM bundles are compatible with:
- runc
- crun
- youki
- Any OCI runtime-spec 1.0.2 compliant runtime

## 12. Test Coverage

| Area | Tests |
|------|-------|
| TIM creation | DataNode wrapping, default config |
| Serialization | Tar round-trips, large files (1MB+) |
| Encryption | ToSigil/FromSigil, wrong password detection |
| Caching | Store/Load/Delete, List, Size |
| Execution | ZIP slip prevention, temp cleanup |
| Error handling | Nil DataNode, nil config, invalid tar |

## 13. Future Work

- [ ] Image layer support
- [ ] Registry push/pull
- [ ] Multi-platform bundles
- [ ] Signature verification
- [ ] Full OCI config generation
