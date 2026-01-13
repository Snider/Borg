# RFC-007: LTHN Key Derivation

**Status**: Draft
**Author**: [Snider](https://github.com/Snider/)
**Created**: 2026-01-13
**License**: EUPL-1.2
**Depends On**: RFC-002

---

## Abstract

LTHN (Leet-Hash-Nonce) is a rainbow-table resistant key derivation function used for streaming DRM with time-limited access. It generates rolling keys that automatically expire without requiring revocation infrastructure.

## 1. Overview

LTHN provides:
- Rainbow-table resistant hashing
- Time-based key rolling
- Zero-trust key derivation (no key server)
- Configurable cadence (daily to hourly)

## 2. Motivation

Traditional DRM requires:
- Central key server
- License validation
- Revocation lists
- Network connectivity

LTHN eliminates these by:
- Deriving keys from public information + secret
- Time-bounding keys automatically
- Making rainbow tables impractical
- Working completely offline

## 3. Algorithm

### 3.1 Core Function

The LTHN hash is implemented in the Enchantrix library:

```go
import "github.com/Snider/Enchantrix/pkg/crypt"

cryptService := crypt.NewService()
lthnHash := cryptService.Hash(crypt.LTHN, input)
```

**LTHN formula**:
```
LTHN(input) = SHA256(input || reverse_leet(input))
```

Where `reverse_leet` performs bidirectional character substitution.

### 3.2 Reverse Leet Mapping

| Original | Leet | Bidirectional |
|----------|------|---------------|
| o | 0 | o ↔ 0 |
| l | 1 | l ↔ 1 |
| e | 3 | e ↔ 3 |
| a | 4 | a ↔ 4 |
| s | z | s ↔ z |
| t | 7 | t ↔ 7 |

### 3.3 Example

```
Input: "2026-01-13:license:fp"
reverse_leet: "pf:3zn3ci1:31-10-6202"
Combined: "2026-01-13:license:fppf:3zn3ci1:31-10-6202"
Result: SHA256(combined) → 32-byte hash
```

## 4. Stream Key Derivation

### 4.1 Implementation

```go
// pkg/smsg/stream.go:49-60
func DeriveStreamKey(date, license, fingerprint string) []byte {
    input := fmt.Sprintf("%s:%s:%s", date, license, fingerprint)
    cryptService := crypt.NewService()
    lthnHash := cryptService.Hash(crypt.LTHN, input)
    key := sha256.Sum256([]byte(lthnHash))
    return key[:]
}
```

### 4.2 Input Format

```
period:license:fingerprint

Where:
- period: Time period identifier (see Cadence)
- license: User's license key (password)
- fingerprint: Device/browser fingerprint
```

### 4.3 Output

32-byte key suitable for ChaCha20-Poly1305.

## 5. Cadence

### 5.1 Options

| Cadence | Constant | Period Format | Example | Duration |
|---------|----------|---------------|---------|----------|
| Daily | `CadenceDaily` | `2006-01-02` | `2026-01-13` | 24h |
| 12-hour | `CadenceHalfDay` | `2006-01-02-AM/PM` | `2026-01-13-PM` | 12h |
| 6-hour | `CadenceQuarter` | `2006-01-02-HH` | `2026-01-13-12` | 6h |
| Hourly | `CadenceHourly` | `2006-01-02-HH` | `2026-01-13-15` | 1h |

### 5.2 Period Calculation

```go
// pkg/smsg/stream.go:73-119
func GetCurrentPeriod(cadence Cadence) string {
    return GetPeriodAt(time.Now(), cadence)
}

func GetPeriodAt(t time.Time, cadence Cadence) string {
    switch cadence {
    case CadenceDaily:
        return t.Format("2006-01-02")
    case CadenceHalfDay:
        suffix := "AM"
        if t.Hour() >= 12 {
            suffix = "PM"
        }
        return t.Format("2006-01-02") + "-" + suffix
    case CadenceQuarter:
        bucket := (t.Hour() / 6) * 6
        return fmt.Sprintf("%s-%02d", t.Format("2006-01-02"), bucket)
    case CadenceHourly:
        return fmt.Sprintf("%s-%02d", t.Format("2006-01-02"), t.Hour())
    }
    return t.Format("2006-01-02")
}

func GetNextPeriod(cadence Cadence) string {
    return GetPeriodAt(time.Now().Add(GetCadenceDuration(cadence)), cadence)
}
```

### 5.3 Duration Mapping

```go
func GetCadenceDuration(cadence Cadence) time.Duration {
    switch cadence {
    case CadenceDaily:
        return 24 * time.Hour
    case CadenceHalfDay:
        return 12 * time.Hour
    case CadenceQuarter:
        return 6 * time.Hour
    case CadenceHourly:
        return 1 * time.Hour
    }
    return 24 * time.Hour
}
```

## 6. Rolling Windows

### 6.1 Dual-Key Strategy

At encryption time, CEK is wrapped with **two** keys:
1. Current period key
2. Next period key

This creates a rolling validity window:

```
Time: 2026-01-13 23:30 (daily cadence)

Valid keys:
- "2026-01-13:license:fp" (current period)
- "2026-01-14:license:fp" (next period)

Window: 24-48 hours of validity
```

### 6.2 Key Wrapping

```go
// pkg/smsg/stream.go:135-155
func WrapCEK(cek []byte, streamKey []byte) (string, error) {
    sigil := enchantrix.NewChaChaPolySigil()
    wrapped, err := sigil.Seal(cek, streamKey)
    if err != nil {
        return "", err
    }
    return base64.StdEncoding.EncodeToString(wrapped), nil
}
```

**Wrapped format**:
```
[24-byte nonce][encrypted CEK][16-byte auth tag]
→ base64 encoded for header storage
```

### 6.3 Key Unwrapping

```go
// pkg/smsg/stream.go:157-170
func UnwrapCEK(wrapped string, streamKey []byte) ([]byte, error) {
    data, err := base64.StdEncoding.DecodeString(wrapped)
    if err != nil {
        return nil, err
    }
    sigil := enchantrix.NewChaChaPolySigil()
    return sigil.Open(data, streamKey)
}
```

### 6.4 Decryption Flow

```go
// pkg/smsg/stream.go:606-633
func UnwrapCEKFromHeader(header *V3Header, params *StreamParams) ([]byte, error) {
    // Try current period first
    currentPeriod := GetCurrentPeriod(params.Cadence)
    currentKey := DeriveStreamKey(currentPeriod, params.License, params.Fingerprint)

    for _, wk := range header.WrappedKeys {
        cek, err := UnwrapCEK(wk.Key, currentKey)
        if err == nil {
            return cek, nil
        }
    }

    // Try next period (for clock skew)
    nextPeriod := GetNextPeriod(params.Cadence)
    nextKey := DeriveStreamKey(nextPeriod, params.License, params.Fingerprint)

    for _, wk := range header.WrappedKeys {
        cek, err := UnwrapCEK(wk.Key, nextKey)
        if err == nil {
            return cek, nil
        }
    }

    return nil, ErrKeyExpired
}
```

## 7. V3 Header Format

```go
type V3Header struct {
    Format      string       `json:"format"`      // "v3"
    Manifest    *Manifest    `json:"manifest"`
    WrappedKeys []WrappedKey `json:"wrappedKeys"`
    Chunked     *ChunkInfo   `json:"chunked,omitempty"`
}

type WrappedKey struct {
    Period string `json:"period"`  // e.g., "2026-01-13"
    Key    string `json:"key"`     // base64-encoded wrapped CEK
}
```

## 8. Rainbow Table Resistance

### 8.1 Why It Works

Standard hash:
```
SHA256("2026-01-13:license:fp") → predictable, precomputable
```

LTHN hash:
```
LTHN("2026-01-13:license:fp")
= SHA256("2026-01-13:license:fp" + reverse_leet("2026-01-13:license:fp"))
= SHA256("2026-01-13:license:fp" + "pf:3zn3ci1:31-10-6202")
```

The salt is **derived from the input itself**, making precomputation impractical:
- Each unique input has a unique salt
- Cannot build rainbow tables without knowing all possible inputs
- Input space includes license keys (high entropy)

### 8.2 Security Analysis

| Attack | Mitigation |
|--------|------------|
| Rainbow tables | Input-derived salt makes precomputation infeasible |
| Brute force | License key entropy (64+ bits recommended) |
| Time oracle | Rolling window prevents precise timing attacks |
| Key sharing | Keys expire within cadence window |

## 9. Zero-Trust Properties

| Property | Implementation |
|----------|----------------|
| No key server | Keys derived locally from LTHN |
| Auto-expiration | Rolling periods invalidate old keys |
| No revocation | Keys naturally expire within cadence window |
| Device binding | Fingerprint in derivation input |
| User binding | License key in derivation input |

## 10. Test Vectors

From `pkg/smsg/stream_test.go`:

```go
// Stream key generation
date := "2026-01-12"
license := "test-license"
fingerprint := "test-fp"
key := DeriveStreamKey(date, license, fingerprint)
// key is 32 bytes, deterministic

// Period calculation at 2026-01-12 15:30:00 UTC
t := time.Date(2026, 1, 12, 15, 30, 0, 0, time.UTC)

GetPeriodAt(t, CadenceDaily)   // "2026-01-12"
GetPeriodAt(t, CadenceHalfDay) // "2026-01-12-PM"
GetPeriodAt(t, CadenceQuarter) // "2026-01-12-12"
GetPeriodAt(t, CadenceHourly)  // "2026-01-12-15"

// Next periods
// Daily: "2026-01-12" → "2026-01-13"
// 12h:   "2026-01-12-PM" → "2026-01-13-AM"
// 6h:    "2026-01-12-12" → "2026-01-12-18"
// 1h:    "2026-01-12-15" → "2026-01-12-16"
```

## 11. Implementation Reference

- Stream key derivation: `pkg/smsg/stream.go`
- LTHN hash: `github.com/Snider/Enchantrix/pkg/crypt`
- WASM bindings: `pkg/wasm/stmf/main.go` (decryptV3, unwrapCEK)
- Tests: `pkg/smsg/stream_test.go`

## 12. Security Considerations

1. **License entropy**: Recommend 64+ bits (12+ alphanumeric chars)
2. **Fingerprint stability**: Should be stable but not user-controllable
3. **Clock skew**: Rolling windows handle ±1 period drift
4. **Key exposure**: Derived keys valid only for one period

## 13. References

- RFC-002: SMSG Format (v3 streaming)
- RFC-001: OSS DRM (Section 3.4)
- RFC 8439: ChaCha20-Poly1305
- Enchantrix: github.com/Snider/Enchantrix
