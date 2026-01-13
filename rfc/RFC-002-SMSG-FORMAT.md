# RFC-002: SMSG Container Format

**Status**: Draft
**Author**: [Snider](https://github.com/Snider/)
**Created**: 2026-01-13
**License**: EUPL-1.2
**Depends On**: RFC-001, RFC-007

---

## Abstract

SMSG (Secure Message) is an encrypted container format using ChaCha20-Poly1305 authenticated encryption. This RFC specifies the binary wire format, versioning, and encoding rules for SMSG files.

## 1. Overview

SMSG provides:
- Authenticated encryption (ChaCha20-Poly1305)
- Public metadata (manifest) readable without decryption
- Multiple format versions (v1 legacy, v2 binary, v3 streaming)
- Optional chunking for large files and seeking

## 2. File Structure

### 2.1 Binary Layout

```
Offset  Size    Field
------  -----   ------------------------------------
0       4       Magic: "SMSG" (ASCII)
4       2       Version: uint16 little-endian
6       3       Header Length: 3-byte big-endian
9       N       Header JSON (plaintext)
9+N     M       Encrypted Payload
```

### 2.2 Magic Number

| Format | Value |
|--------|-------|
| Binary | `0x53 0x4D 0x53 0x47` |
| ASCII | `SMSG` |
| Base64 (first 6 chars) | `U01TRw` |

### 2.3 Version Field

Current version: `0x0001` (1)

Decoders MUST reject versions they don't understand.

### 2.4 Header Length

3 bytes, big-endian unsigned integer. Supports headers up to 16 MB.

## 3. Header Format (JSON)

Header is always plaintext (never encrypted), enabling metadata inspection without decryption.

### 3.1 Base Header

```json
{
  "version": "1.0",
  "algorithm": "chacha20poly1305",
  "format": "v2",
  "compression": "zstd",
  "manifest": { ... }
}
```

### 3.2 V3 Header Extensions

```json
{
  "version": "1.0",
  "algorithm": "chacha20poly1305",
  "format": "v3",
  "compression": "zstd",
  "keyMethod": "lthn-rolling",
  "cadence": "daily",
  "manifest": { ... },
  "wrappedKeys": [
    {"date": "2026-01-13", "wrapped": "<base64>"},
    {"date": "2026-01-14", "wrapped": "<base64>"}
  ],
  "chunked": {
    "chunkSize": 1048576,
    "totalChunks": 42,
    "totalSize": 44040192,
    "index": [
      {"offset": 0, "size": 1048600},
      {"offset": 1048600, "size": 1048600}
    ]
  }
}
```

### 3.3 Header Field Reference

| Field | Type | Values | Description |
|-------|------|--------|-------------|
| version | string | "1.0" | Format version string |
| algorithm | string | "chacha20poly1305" | Always ChaCha20-Poly1305 |
| format | string | "", "v2", "v3" | Payload format version |
| compression | string | "", "gzip", "zstd" | Compression algorithm |
| keyMethod | string | "", "lthn-rolling" | Key derivation method |
| cadence | string | "daily", "12h", "6h", "1h" | Rolling key period (v3) |
| manifest | object | - | Content metadata |
| wrappedKeys | array | - | CEK wrapped for each period (v3) |
| chunked | object | - | Chunk index for seeking (v3) |

## 4. Manifest Structure

### 4.1 Complete Manifest

```go
type Manifest struct {
    Title        string            `json:"title,omitempty"`
    Artist       string            `json:"artist,omitempty"`
    Album        string            `json:"album,omitempty"`
    Genre        string            `json:"genre,omitempty"`
    Year         int               `json:"year,omitempty"`
    ReleaseType  string            `json:"release_type,omitempty"`
    Duration     int               `json:"duration,omitempty"`
    Format       string            `json:"format,omitempty"`
    ExpiresAt    int64             `json:"expires_at,omitempty"`
    IssuedAt     int64             `json:"issued_at,omitempty"`
    LicenseType  string            `json:"license_type,omitempty"`
    Tracks       []Track           `json:"tracks,omitempty"`
    Links        map[string]string `json:"links,omitempty"`
    Tags         []string          `json:"tags,omitempty"`
    Extra        map[string]string `json:"extra,omitempty"`
}

type Track struct {
    Title    string  `json:"title"`
    Start    float64 `json:"start"`
    End      float64 `json:"end,omitempty"`
    Type     string  `json:"type,omitempty"`
    TrackNum int     `json:"track_num,omitempty"`
}
```

### 4.2 Manifest Field Reference

| Field | Type | Range | Description |
|-------|------|-------|-------------|
| title | string | 0-255 chars | Display name (required for discovery) |
| artist | string | 0-255 chars | Creator name |
| album | string | 0-255 chars | Album/collection name |
| genre | string | 0-255 chars | Genre classification |
| year | int | 0-9999 | Release year (0 = unset) |
| releaseType | string | enum | "single", "album", "ep", "mix" |
| duration | int | 0+ | Total duration in seconds |
| format | string | any | Platform format string (e.g., "dapp.fm/v1") |
| expiresAt | int64 | 0+ | Unix timestamp (0 = never expires) |
| issuedAt | int64 | 0+ | Unix timestamp of license issue |
| licenseType | string | enum | "perpetual", "rental", "stream", "preview" |
| tracks | []Track | - | Track boundaries for multi-track releases |
| links | map | - | Platform name → URL (e.g., "bandcamp" → URL) |
| tags | []string | - | Arbitrary string tags |
| extra | map | - | Free-form key-value extension data |

## 5. Format Versions

### 5.1 Version Comparison

| Aspect | v1 (Legacy) | v2 (Binary) | v3 (Streaming) |
|--------|-------------|-------------|----------------|
| Payload Structure | JSON only | Length-prefixed JSON + binary | Same as v2 |
| Attachment Encoding | Base64 in JSON | Size field + raw binary | Size field + raw binary |
| Compression | None | zstd (default) | zstd (default) |
| Key Derivation | SHA256(password) | SHA256(password) | LTHN rolling keys |
| Chunked Support | No | No | Yes (optional) |
| Size Overhead | ~33% | ~25% | ~15% |
| Use Case | Legacy | General purpose | Time-limited streaming |

### 5.2 V1 Format (Legacy)

**Payload (after decryption):**

```json
{
  "body": "Message content",
  "subject": "Optional subject",
  "from": "sender@example.com",
  "to": "recipient@example.com",
  "timestamp": 1673644800,
  "attachments": [
    {
      "name": "file.bin",
      "content": "base64encodeddata==",
      "mime": "application/octet-stream",
      "size": 1024
    }
  ],
  "reply_key": {
    "public_key": "base64x25519key==",
    "algorithm": "x25519"
  },
  "meta": {
    "custom_field": "custom_value"
  }
}
```

- Attachments base64-encoded inline in JSON (~33% overhead)
- Simple but inefficient for large files

### 5.3 V2 Format (Binary)

**Payload structure (after decryption and decompression):**

```
Offset  Size    Field
------  -----   ------------------------------------
0       4       Message JSON Length (big-endian uint32)
4       N       Message JSON (attachments have size only, no content)
4+N     B1      Attachment 1 raw binary
4+N+B1  B2      Attachment 2 raw binary
...
```

**Message JSON (within payload):**

```json
{
  "body": "Message text",
  "subject": "Subject",
  "from": "sender",
  "attachments": [
    {"name": "file1.bin", "mime": "application/octet-stream", "size": 4096},
    {"name": "file2.bin", "mime": "image/png", "size": 65536}
  ],
  "timestamp": 1673644800
}
```

- Attachment `content` field omitted; binary data follows JSON
- Compressed before encryption
- 3-10x faster than v1, ~25% smaller

### 5.4 V3 Format (Streaming)

Same payload structure as v2, but with:
- LTHN-derived rolling keys instead of password
- CEK (Content Encryption Key) wrapped for each time period
- Optional chunking for seek support

**CEK Wrapping:**

```
For each rolling period:
  streamKey = SHA256(LTHN(period:license:fingerprint))
  wrappedKey = ChaCha20-Poly1305(CEK, streamKey)
```

**Rolling Periods (cadence):**

| Cadence | Period Format | Example |
|---------|---------------|---------|
| daily | YYYY-MM-DD | "2026-01-13" |
| 12h | YYYY-MM-DD-AM/PM | "2026-01-13-AM" |
| 6h | YYYY-MM-DD-HH | "2026-01-13-00", "2026-01-13-06" |
| 1h | YYYY-MM-DD-HH | "2026-01-13-15" |

### 5.5 V3 Chunked Format

**Payload (independently decryptable chunks):**

```
Offset      Size      Content
------      -----     ----------------------------------
0           1048600   Chunk 0: [24-byte nonce][ciphertext][16-byte tag]
1048600     1048600   Chunk 1: [24-byte nonce][ciphertext][16-byte tag]
...
```

- Each chunk encrypted separately with same CEK, unique nonce
- Enables seeking, HTTP Range requests
- Chunk size typically 1MB (configurable)

## 6. Encryption

### 6.1 Algorithm

XChaCha20-Poly1305 (extended nonce variant)

| Parameter | Value |
|-----------|-------|
| Key size | 32 bytes |
| Nonce size | 24 bytes (XChaCha) |
| Tag size | 16 bytes |

### 6.2 Ciphertext Structure

```
[24-byte XChaCha20 nonce][encrypted data][16-byte Poly1305 tag]
```

**Critical**: Nonces are embedded IN the ciphertext by the Enchantrix library, NOT transmitted separately in headers.

### 6.3 Key Derivation

**V1/V2 (Password-based):**

```go
key := sha256.Sum256([]byte(password))  // 32 bytes
```

**V3 (LTHN Rolling):**

```go
// For each period in rolling window:
streamKey := sha256.Sum256([]byte(
    crypt.NewService().Hash(crypt.LTHN, period + ":" + license + ":" + fingerprint)
))
```

## 7. Compression

| Value | Algorithm | Notes |
|-------|-----------|-------|
| "" (empty) | None | Raw bytes, default for v1 |
| "gzip" | RFC 1952 | Stdlib, WASM compatible |
| "zstd" | Zstandard | Default for v2/v3, better ratio |

**Order**: Compress → Encrypt (on write), Decrypt → Decompress (on read)

## 8. Message Structure

### 8.1 Go Types

```go
type Message struct {
    From        string            `json:"from,omitempty"`
    To          string            `json:"to,omitempty"`
    Subject     string            `json:"subject,omitempty"`
    Body        string            `json:"body"`
    Timestamp   int64             `json:"timestamp,omitempty"`
    Attachments []Attachment      `json:"attachments,omitempty"`
    ReplyKey    *KeyInfo          `json:"reply_key,omitempty"`
    Meta        map[string]string `json:"meta,omitempty"`
}

type Attachment struct {
    Name    string `json:"name"`
    Mime    string `json:"mime"`
    Size    int    `json:"size"`
    Content string `json:"content,omitempty"`  // Base64, v1 only
    Data    []byte `json:"-"`                  // Binary, v2/v3
}

type KeyInfo struct {
    PublicKey string `json:"public_key"`
    Algorithm string `json:"algorithm"`
}
```

### 8.2 Stream Parameters (V3)

```go
type StreamParams struct {
    License     string `json:"license"`      // User's license identifier
    Fingerprint string `json:"fingerprint"`  // Device fingerprint (optional)
    Cadence     string `json:"cadence"`      // Rolling period: daily, 12h, 6h, 1h
    ChunkSize   int    `json:"chunk_size"`   // Bytes per chunk (default 1MB)
}
```

## 9. Error Handling

### 9.1 Error Types

```go
var (
    ErrInvalidMagic     = errors.New("invalid SMSG magic")
    ErrInvalidPayload   = errors.New("invalid SMSG payload")
    ErrDecryptionFailed = errors.New("decryption failed (wrong password?)")
    ErrPasswordRequired = errors.New("password is required")
    ErrEmptyMessage     = errors.New("message cannot be empty")
    ErrStreamKeyExpired = errors.New("stream key expired (outside rolling window)")
    ErrNoValidKey       = errors.New("no valid wrapped key found for current date")
    ErrLicenseRequired  = errors.New("license is required for stream decryption")
)
```

### 9.2 Error Conditions

| Error | Cause | Recovery |
|-------|-------|----------|
| ErrInvalidMagic | File magic is not "SMSG" | Verify file format |
| ErrInvalidPayload | Corrupted payload structure | Re-download or restore |
| ErrDecryptionFailed | Wrong password or corrupted | Try correct password |
| ErrPasswordRequired | Empty password provided | Provide password |
| ErrStreamKeyExpired | Time outside rolling window | Wait for valid period or update file |
| ErrNoValidKey | No wrapped key for current period | License/fingerprint mismatch |
| ErrLicenseRequired | Empty StreamParams.License | Provide license identifier |

## 10. Constants

```go
const Magic = "SMSG"                      // 4 ASCII bytes
const Version = "1.0"                     // String version identifier
const DefaultChunkSize = 1024 * 1024      // 1 MB

const FormatV1 = ""                       // Legacy JSON format
const FormatV2 = "v2"                     // Binary format
const FormatV3 = "v3"                     // Streaming with rolling keys

const KeyMethodDirect = ""                // Password-direct (v1/v2)
const KeyMethodLTHNRolling = "lthn-rolling" // LTHN rolling (v3)

const CompressionNone = ""
const CompressionGzip = "gzip"
const CompressionZstd = "zstd"

const CadenceDaily = "daily"
const CadenceHalfDay = "12h"
const CadenceQuarter = "6h"
const CadenceHourly = "1h"
```

## 11. API Usage

### 11.1 V1 (Legacy)

```go
msg := NewMessage("Hello").WithSubject("Test")
encrypted, _ := Encrypt(msg, "password")
decrypted, _ := Decrypt(encrypted, "password")
```

### 11.2 V2 (Binary)

```go
msg := NewMessage("Hello").AddBinaryAttachment("file.bin", data, "application/octet-stream")
manifest := NewManifest("My Content")
encrypted, _ := EncryptV2WithManifest(msg, "password", manifest)
decrypted, _ := Decrypt(encrypted, "password")
```

### 11.3 V3 (Streaming)

```go
msg := NewMessage("Stream content")
params := &StreamParams{
    License:     "user-license",
    Fingerprint: "device-fingerprint",
    Cadence:     CadenceDaily,
    ChunkSize:   1048576,
}
manifest := NewManifest("Stream Track")
manifest.LicenseType = "stream"
encrypted, _ := EncryptV3(msg, params, manifest)
decrypted, header, _ := DecryptV3(encrypted, params)
```

## 12. Implementation Reference

- Types: `pkg/smsg/types.go`
- Encryption: `pkg/smsg/smsg.go`
- Streaming: `pkg/smsg/stream.go`
- WASM: `pkg/wasm/stmf/main.go`
- Tests: `pkg/smsg/*_test.go`

## 13. Security Considerations

1. **Nonce uniqueness**: Enchantrix generates random 24-byte nonces automatically
2. **Key entropy**: Passwords should have 64+ bits entropy (no key stretching)
3. **Manifest exposure**: Manifest is public; never include sensitive data
4. **Constant-time crypto**: Enchantrix uses constant-time comparison for auth tags
5. **Rolling window**: V3 keys valid for current + next period only

## 14. Future Work

- [ ] Key stretching (Argon2 option)
- [ ] Multi-recipient encryption
- [ ] Streaming API with ReadableStream
- [ ] Hardware key support (WebAuthn)
