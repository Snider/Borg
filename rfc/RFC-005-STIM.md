# RFC-005: STIM Encrypted Container Format

**Status**: Draft
**Author**: [Snider](https://github.com/Snider/)
**Created**: 2026-01-13
**License**: EUPL-1.2
**Depends On**: RFC-003, RFC-004

---

## Abstract

STIM (Secure TIM) is an encrypted container format that wraps TIM bundles using ChaCha20-Poly1305 authenticated encryption. It enables secure distribution and execution of containers without exposing the contents.

## 1. Overview

STIM provides:
- Encrypted TIM containers
- ChaCha20-Poly1305 authenticated encryption
- Separate encryption of config and rootfs
- Direct execution without persistent decryption

## 2. Format Name

**ChaChaPolySigil** - The internal name for the STIM format, using:
- ChaCha20-Poly1305 algorithm (via Enchantrix library)
- Trix container wrapper with "STIM" magic

## 3. File Structure

### 3.1 Container Format

STIM uses the **Trix container format** from Enchantrix library:

```
┌─────────────────────────────────────────┐
│ Magic: "STIM" (4 bytes ASCII)           │
├─────────────────────────────────────────┤
│ Trix Header (Gob-encoded JSON)          │
│  - encryption_algorithm: "chacha20poly1305"
│  - tim: true                            │
│  - config_size: uint32                  │
│  - rootfs_size: uint32                  │
│  - version: "1.0"                       │
├─────────────────────────────────────────┤
│ Trix Payload:                           │
│  [config_size: 4 bytes BE uint32]       │
│  [encrypted config]                     │
│  [encrypted rootfs tar]                 │
└─────────────────────────────────────────┘
```

### 3.2 Payload Structure

```
Offset  Size    Field
------  -----   ------------------------------------
0       4       Config size (big-endian uint32)
4       N       Encrypted config (includes nonce + tag)
4+N     M       Encrypted rootfs tar (includes nonce + tag)
```

### 3.3 Encrypted Component Format

Each encrypted component (config and rootfs) follows Enchantrix format:

```
[24-byte XChaCha20 nonce][ciphertext][16-byte Poly1305 tag]
```

**Critical**: Nonces are **embedded in the ciphertext**, not transmitted separately.

## 4. Encryption

### 4.1 Algorithm

XChaCha20-Poly1305 (extended nonce variant)

| Parameter | Value |
|-----------|-------|
| Key size | 32 bytes |
| Nonce size | 24 bytes (embedded) |
| Tag size | 16 bytes |

### 4.2 Key Derivation

```go
// pkg/trix/trix.go:64-67
func DeriveKey(password string) []byte {
    hash := sha256.Sum256([]byte(password))
    return hash[:]  // 32 bytes
}
```

### 4.3 Dual Encryption

Config and RootFS are encrypted **separately** with independent nonces:

```go
// pkg/tim/tim.go:217-232
func (m *TerminalIsolationMatrix) ToSigil(password string) ([]byte, error) {
    // 1. Derive key
    key := trix.DeriveKey(password)

    // 2. Create sigil
    sigil, _ := enchantrix.NewChaChaPolySigil(key)

    // 3. Encrypt config (generates fresh nonce automatically)
    encConfig, _ := sigil.In(m.Config)

    // 4. Serialize rootfs to tar
    rootfsTar, _ := m.RootFS.ToTar()

    // 5. Encrypt rootfs (generates different fresh nonce)
    encRootFS, _ := sigil.In(rootfsTar)

    // 6. Build payload
    payload := make([]byte, 4+len(encConfig)+len(encRootFS))
    binary.BigEndian.PutUint32(payload[:4], uint32(len(encConfig)))
    copy(payload[4:4+len(encConfig)], encConfig)
    copy(payload[4+len(encConfig):], encRootFS)

    // 7. Create Trix container with STIM magic
    // ...
}
```

**Rationale for dual encryption:**
- Config can be decrypted separately for inspection
- Allows streaming decryption of large rootfs
- Independent nonces prevent any nonce reuse

## 5. Decryption Flow

```go
// pkg/tim/tim.go:255-308
func FromSigil(data []byte, password string) (*TerminalIsolationMatrix, error) {
    // 1. Decode Trix container with magic "STIM"
    t, _ := trix.Decode(data, "STIM", nil)

    // 2. Derive key from password
    key := trix.DeriveKey(password)

    // 3. Create sigil
    sigil, _ := enchantrix.NewChaChaPolySigil(key)

    // 4. Parse payload: extract configSize from first 4 bytes
    configSize := binary.BigEndian.Uint32(t.Payload[:4])

    // 5. Validate bounds
    if int(configSize) > len(t.Payload)-4 {
        return nil, ErrInvalidStimPayload
    }

    // 6. Extract encrypted components
    encConfig := t.Payload[4 : 4+configSize]
    encRootFS := t.Payload[4+configSize:]

    // 7. Decrypt config (nonce auto-extracted by Enchantrix)
    config, err := sigil.Out(encConfig)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
    }

    // 8. Decrypt rootfs
    rootfsTar, err := sigil.Out(encRootFS)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
    }

    // 9. Reconstruct DataNode from tar
    rootfs, _ := datanode.FromTar(rootfsTar)

    return &TerminalIsolationMatrix{Config: config, RootFS: rootfs}, nil
}
```

## 6. Trix Header

```go
Header: map[string]interface{}{
    "encryption_algorithm": "chacha20poly1305",
    "tim":                  true,
    "config_size":          len(encConfig),
    "rootfs_size":          len(encRootFS),
    "version":              "1.0",
}
```

## 7. CLI Usage

```bash
# Create encrypted container
borg compile -f Borgfile -e "password" -o container.stim

# Run encrypted container
borg run container.stim -p "password"

# Decode (extract) encrypted container
borg decode container.stim -p "password" --i-am-in-isolation -o container.tar

# Inspect without decrypting (shows header metadata only)
borg inspect container.stim
# Output:
#   Format: STIM
#   encryption_algorithm: chacha20poly1305
#   config_size: 1234
#   rootfs_size: 567890
```

## 8. Cache API

```go
// Create cache with master password
cache, err := tim.NewCache("/path/to/cache", masterPassword)

// Store TIM (encrypted automatically as .stim)
err := cache.Store("name", tim)

// Load TIM (decrypted automatically)
tim, err := cache.Load("name")

// List cached containers
names, err := cache.List()
```

## 9. Execution Security

```go
// Secure execution flow
func RunEncrypted(path, password string) error {
    // 1. Create secure temp directory
    tmpDir, _ := os.MkdirTemp("", "borg-run-*")
    defer os.RemoveAll(tmpDir)  // Secure cleanup

    // 2. Read and decrypt
    data, _ := os.ReadFile(path)
    tim, _ := FromSigil(data, password)

    // 3. Extract to temp
    tim.ExtractTo(tmpDir)

    // 4. Execute with runc
    return runRunc(tmpDir)
}
```

## 10. Security Properties

### 10.1 Confidentiality

- Contents encrypted with ChaCha20-Poly1305
- Password-derived key never stored
- Nonces are random, never reused

### 10.2 Integrity

- Poly1305 MAC prevents tampering
- Decryption fails if modified
- Separate MACs for config and rootfs

### 10.3 Error Detection

| Error | Cause |
|-------|-------|
| `ErrPasswordRequired` | Empty password provided |
| `ErrInvalidStimPayload` | Payload < 4 bytes or invalid size |
| `ErrDecryptionFailed` | Wrong password or corrupted data |

## 11. Comparison to TRIX

| Feature | STIM | TRIX |
|---------|------|------|
| Algorithm | ChaCha20-Poly1305 | PGP/AES or ChaCha |
| Content | TIM bundles | DataNode (raw files) |
| Structure | Dual encryption | Single blob |
| Magic | "STIM" | "TRIX" |
| Use case | Container execution | General encryption, accounts |

STIM is for containers. TRIX is for general file encryption and accounts.

## 12. Implementation Reference

- Encryption: `pkg/tim/tim.go` (ToSigil, FromSigil)
- Key derivation: `pkg/trix/trix.go` (DeriveKey)
- Cache: `pkg/tim/cache.go`
- CLI: `cmd/run.go`, `cmd/decode.go`, `cmd/compile.go`
- Enchantrix: `github.com/Snider/Enchantrix`

## 13. Security Considerations

1. **Password strength**: Recommend 64+ bits entropy (12+ chars)
2. **Key derivation**: SHA-256 only (no stretching) - use strong passwords
3. **Memory handling**: Keys should be wiped after use
4. **Temp files**: Use tmpfs when available, secure wipe after
5. **Side channels**: Enchantrix uses constant-time crypto operations

## 14. Future Work

- [ ] Hardware key support (YubiKey, TPM)
- [ ] Key stretching (Argon2)
- [ ] Multi-recipient encryption
- [ ] Streaming decryption for large rootfs
