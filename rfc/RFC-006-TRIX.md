# RFC-006: TRIX PGP Encryption Format

**Status**: Draft
**Author**: [Snider](https://github.com/Snider/)
**Created**: 2026-01-13
**License**: EUPL-1.2
**Depends On**: RFC-003

---

## Abstract

TRIX is a PGP-based encryption format for DataNode archives and account credentials. It provides symmetric and asymmetric encryption using OpenPGP standards and ChaCha20-Poly1305, enabling secure data exchange and identity management.

## 1. Overview

TRIX provides:
- PGP symmetric encryption for DataNode archives
- ChaCha20-Poly1305 modern encryption
- PGP armored keys for account/identity management
- Integration with Enchantrix library

## 2. Public API

### 2.1 Key Derivation

```go
// pkg/trix/trix.go:64-67
func DeriveKey(password string) []byte {
    hash := sha256.Sum256([]byte(password))
    return hash[:]  // 32 bytes
}
```

- Input: password string (any length)
- Output: 32-byte key (256 bits)
- Algorithm: SHA-256 hash of UTF-8 bytes
- Deterministic: identical passwords → identical keys

### 2.2 Legacy PGP Encryption

```go
// Encrypt DataNode to TRIX (PGP symmetric)
func ToTrix(dn *datanode.DataNode, password string) ([]byte, error)

// Decrypt TRIX to DataNode (DISABLED for encrypted payloads)
func FromTrix(data []byte, password string) (*datanode.DataNode, error)
```

**Note**: `FromTrix` with a non-empty password returns error `"decryption disabled: cannot accept encrypted payloads"`. This is intentional to prevent accidental password use.

### 2.3 Modern ChaCha20-Poly1305 Encryption

```go
// Encrypt with ChaCha20-Poly1305
func ToTrixChaCha(dn *datanode.DataNode, password string) ([]byte, error)

// Decrypt ChaCha20-Poly1305
func FromTrixChaCha(data []byte, password string) (*datanode.DataNode, error)
```

### 2.4 Error Variables

```go
var (
    ErrPasswordRequired = errors.New("password is required for encryption")
    ErrDecryptionFailed = errors.New("decryption failed (wrong password?)")
)
```

## 3. File Format

### 3.1 Container Structure

```
[4 bytes]   Magic: "TRIX" (ASCII)
[Variable]  Gob-encoded Header (map[string]interface{})
[Variable]  Payload (encrypted or unencrypted tarball)
```

### 3.2 Header Examples

**Unencrypted:**
```go
Header: map[string]interface{}{}  // Empty map
```

**ChaCha20-Poly1305:**
```go
Header: map[string]interface{}{
    "encryption_algorithm": "chacha20poly1305",
}
```

### 3.3 ChaCha20-Poly1305 Payload

```
[24 bytes]  XChaCha20 Nonce (embedded)
[N bytes]   Encrypted tar archive
[16 bytes]  Poly1305 authentication tag
```

**Note**: Nonces are embedded in the ciphertext by Enchantrix, not stored separately.

## 4. Encryption Workflows

### 4.1 ChaCha20-Poly1305 (Recommended)

```go
// Encryption
func ToTrixChaCha(dn *datanode.DataNode, password string) ([]byte, error) {
    // 1. Validate password is non-empty
    if password == "" {
        return nil, ErrPasswordRequired
    }

    // 2. Serialize DataNode to tar
    tarball, _ := dn.ToTar()

    // 3. Derive 32-byte key
    key := DeriveKey(password)

    // 4. Create sigil and encrypt
    sigil, _ := enchantrix.NewChaChaPolySigil(key)
    encrypted, _ := sigil.In(tarball)  // Generates nonce automatically

    // 5. Create Trix container
    t := &trix.Trix{
        Header:  map[string]interface{}{"encryption_algorithm": "chacha20poly1305"},
        Payload: encrypted,
    }

    // 6. Encode with TRIX magic
    return trix.Encode(t, "TRIX", nil)
}
```

### 4.2 Decryption

```go
func FromTrixChaCha(data []byte, password string) (*datanode.DataNode, error) {
    // 1. Validate password
    if password == "" {
        return nil, ErrPasswordRequired
    }

    // 2. Decode TRIX container
    t, _ := trix.Decode(data, "TRIX", nil)

    // 3. Derive key and decrypt
    key := DeriveKey(password)
    sigil, _ := enchantrix.NewChaChaPolySigil(key)
    tarball, err := sigil.Out(t.Payload)  // Extracts nonce, verifies MAC
    if err != nil {
        return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
    }

    // 4. Deserialize DataNode
    return datanode.FromTar(tarball)
}
```

### 4.3 Legacy PGP (Disabled Decryption)

```go
func ToTrix(dn *datanode.DataNode, password string) ([]byte, error) {
    tarball, _ := dn.ToTar()

    var payload []byte
    if password != "" {
        // PGP symmetric encryption
        cryptService := crypt.NewService()
        payload, _ = cryptService.SymmetricallyEncryptPGP([]byte(password), tarball)
    } else {
        payload = tarball
    }

    t := &trix.Trix{Header: map[string]interface{}{}, Payload: payload}
    return trix.Encode(t, "TRIX", nil)
}

func FromTrix(data []byte, password string) (*datanode.DataNode, error) {
    // Security: Reject encrypted payloads
    if password != "" {
        return nil, errors.New("decryption disabled: cannot accept encrypted payloads")
    }

    t, _ := trix.Decode(data, "TRIX", nil)
    return datanode.FromTar(t.Payload)
}
```

## 5. Enchantrix Library

### 5.1 Dependencies

```go
import (
    "github.com/Snider/Enchantrix/pkg/trix"      // Container format
    "github.com/Snider/Enchantrix/pkg/crypt"     // PGP operations
    "github.com/Snider/Enchantrix/pkg/enchantrix" // AEAD sigils
)
```

### 5.2 Trix Container

```go
type Trix struct {
    Header  map[string]interface{}
    Payload []byte
}

func Encode(t *Trix, magic string, extra interface{}) ([]byte, error)
func Decode(data []byte, magic string, extra interface{}) (*Trix, error)
```

### 5.3 ChaCha20-Poly1305 Sigil

```go
// Create sigil with 32-byte key
sigil, err := enchantrix.NewChaChaPolySigil(key)

// Encrypt (generates random 24-byte nonce)
ciphertext, err := sigil.In(plaintext)

// Decrypt (extracts nonce, verifies MAC)
plaintext, err := sigil.Out(ciphertext)
```

## 6. Account System Integration

### 6.1 PGP Armored Keys

```
-----BEGIN PGP PUBLIC KEY BLOCK-----

mQENBGX...base64...
-----END PGP PUBLIC KEY BLOCK-----
```

### 6.2 Key Storage

```
~/.borg/
├── identity.pub     # PGP public key (armored)
├── identity.key     # PGP private key (armored, encrypted)
└── keyring/         # Trusted public keys
```

## 7. CLI Usage

```bash
# Encrypt with TRIX (PGP symmetric)
borg collect github repo https://github.com/user/repo \
    --format trix \
    --password "password"

# Decrypt unencrypted TRIX
borg decode archive.trix -o decoded.tar

# Inspect without decrypting
borg inspect archive.trix
# Output:
#   Format: TRIX
#   encryption_algorithm: chacha20poly1305 (if present)
#   Payload Size: N bytes
```

## 8. Format Comparison

| Format | Extension | Algorithm | Use Case |
|--------|-----------|-----------|----------|
| `datanode` | `.tar` | None | Uncompressed archive |
| `tim` | `.tim` | None | Container bundle |
| `trix` | `.trix` | PGP/AES or ChaCha | Encrypted archives, accounts |
| `stim` | `.stim` | ChaCha20-Poly1305 | Encrypted containers |
| `smsg` | `.smsg` | ChaCha20-Poly1305 | Encrypted media |

## 9. Security Analysis

### 9.1 Key Derivation Limitations

**Current implementation: SHA-256 (single round)**

| Metric | Value |
|--------|-------|
| Algorithm | SHA-256 |
| Iterations | 1 |
| Salt | None |
| Key stretching | None |

**Implications:**
- GPU brute force: ~10 billion guesses/second
- 8-character password: ~10 seconds to break
- Recommendation: Use 15+ character passwords

### 9.2 ChaCha20-Poly1305 Properties

| Property | Status |
|----------|--------|
| Authentication | Poly1305 MAC (16 bytes) |
| Key size | 256 bits |
| Nonce size | 192 bits (XChaCha) |
| Standard | RFC 7539 compliant |

## 10. Test Coverage

| Test | Description |
|------|-------------|
| DeriveKey length | Output is exactly 32 bytes |
| DeriveKey determinism | Same password → same key |
| DeriveKey uniqueness | Different passwords → different keys |
| ToTrix without password | Valid TRIX with "TRIX" magic |
| ToTrix with password | PGP encryption applied |
| FromTrix unencrypted | Round-trip preserves files |
| FromTrix password rejection | Returns error |
| ToTrixChaCha success | Valid TRIX created |
| ToTrixChaCha empty password | Returns ErrPasswordRequired |
| FromTrixChaCha round-trip | Preserves nested directories |
| FromTrixChaCha wrong password | Returns ErrDecryptionFailed |
| FromTrixChaCha large data | 1MB file processed |

## 11. Implementation Reference

- Source: `pkg/trix/trix.go`
- Tests: `pkg/trix/trix_test.go`
- Enchantrix: `github.com/Snider/Enchantrix v0.0.2`

## 12. Security Considerations

1. **Use strong passwords**: 15+ characters due to no key stretching
2. **Prefer ChaCha**: Use `ToTrixChaCha` over legacy PGP
3. **Key backup**: Securely backup private keys
4. **Interoperability**: TRIX files with GPG require password

## 13. Future Work

- [ ] Key stretching (Argon2 option in DeriveKey)
- [ ] Public key encryption support
- [ ] Signature support
- [ ] Key expiration metadata
- [ ] Multi-recipient encryption
