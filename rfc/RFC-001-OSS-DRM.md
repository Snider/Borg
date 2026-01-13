# RFC-001: Open Source DRM for Independent Artists

**Status**: Proposed
**Author**: [Snider](https://github.com/Snider/)
**Created**: 2026-01-10
**License**: EUPL-1.2

---

**Revision History**

| Date | Status | Notes |
|------|--------|-------|
| 2026-01-13 | Proposed | **Adaptive Bitrate (ABR)**: HLS-style multi-quality streaming with encrypted variants. New Section 3.7. All Future Work items complete. |
| 2026-01-12 | Proposed | **Chunked streaming**: v3 now supports optional ChunkSize for independently decryptable chunks - enables seek, HTTP Range, and decrypt-while-downloading. |
| 2026-01-12 | Proposed | **v3 Streaming**: LTHN rolling keys with configurable cadence (daily/12h/6h/1h). CEK wrapping for zero-trust streaming. WASM v1.3.0 with decryptV3(). |
| 2026-01-10 | Proposed | Technical review passed. Fixed section numbering (7.x, 8.x, 9.x, 11.x). Updated WASM size to 5.9MB. Implementation verified complete for stated scope. |

---

## Abstract

This RFC describes an open-source Digital Rights Management (DRM) system designed for independent artists to distribute encrypted media directly to fans without platform intermediaries. The system uses ChaCha20-Poly1305 authenticated encryption with a "password-as-license" model, enabling zero-trust distribution where the encryption key serves as both the license and the decryption mechanism.

## 1. Motivation

### 1.1 The Problem

Traditional music distribution forces artists into platforms that:
- Take 30-70% of revenue (Spotify, Apple Music, Bandcamp)
- Control the relationship between artist and fan
- Require ongoing subscription for access
- Can delist content unilaterally

Existing DRM systems (Widevine, FairPlay) require:
- Platform integration and licensing fees
- Centralized key servers
- Proprietary implementations
- Trust in third parties

### 1.2 The Solution

A DRM system where:
- **The password IS the license** - no key servers, no escrow
- **Artists keep 100%** - sell direct, any payment processor
- **Host anywhere** - CDN, IPFS, S3, personal server
- **Browser or native** - same encryption, same content
- **Open source** - auditable, forkable, community-owned

## 2. Design Philosophy

### 2.1 "Honest DRM"

Traditional DRM operates on a flawed premise: that sufficiently complex technology can prevent copying. History proves otherwise—every DRM system has been broken. The result is systems that:
- Punish paying customers with restrictions
- Get cracked within days/weeks anyway
- Require massive infrastructure (key servers, license servers)
- Create single points of failure

This system embraces a different philosophy: **DRM for honest people**.

The goal isn't to stop determined pirates (impossible). The goal is:
1. Make the legitimate path easy and pleasant
2. Make casual sharing slightly inconvenient
3. Create a social/economic deterrent (sharing = giving away money)
4. Remove all friction for paying customers

### 2.2 Password-as-License

The password IS the license. This is not a limitation—it's the core innovation.

```
Traditional DRM:
  Purchase → License Server → Device Registration → Key Exchange → Playback
  (5 steps, 3 network calls, 2 points of failure)

dapp.fm:
  Purchase → Password → Playback
  (2 steps, 0 network calls, 0 points of failure)
```

Benefits:
- **No accounts** - No email harvesting, no password resets, no data breaches
- **No servers** - Artist can disappear; content still works forever
- **No revocation anxiety** - You bought it, you own it
- **Transferable** - Give your password to a friend (like lending a CD)
- **Archival** - Works in 50 years if you have the password

### 2.3 Encryption as Access Control

We use military-grade encryption (ChaCha20-Poly1305) not because we need military-grade security, but because:
1. It's fast (important for real-time media)
2. It's auditable (open standard, RFC 8439)
3. It's already implemented everywhere (Go stdlib, browser crypto)
4. It provides authenticity (Poly1305 MAC prevents tampering)

The threat model isn't nation-states—it's casual piracy. The encryption just needs to be "not worth the effort to crack for a $10 album."

## 3. Architecture

### 3.1 System Components

```
┌─────────────────────────────────────────────────────────────┐
│                    DISTRIBUTION LAYER                        │
│  CDN / IPFS / S3 / GitHub / Personal Server                 │
│  (Encrypted .smsg files - safe to host anywhere)            │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    PLAYBACK LAYER                            │
│  ┌─────────────────┐     ┌─────────────────────────────┐   │
│  │   Browser Demo   │     │     Native Desktop App      │   │
│  │   (WASM)        │     │     (Wails + Go)            │   │
│  │                 │     │                             │   │
│  │  ┌───────────┐  │     │  ┌───────────────────────┐  │   │
│  │  │ stmf.wasm │  │     │  │  Go SMSG Library      │  │   │
│  │  │           │  │     │  │  (pkg/smsg)           │  │   │
│  │  │ ChaCha20  │  │     │  │                       │  │   │
│  │  │ Poly1305  │  │     │  │  ChaCha20-Poly1305    │  │   │
│  │  └───────────┘  │     │  └───────────────────────┘  │   │
│  └─────────────────┘     └─────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    LICENSE LAYER                             │
│  Password = License Key = Decryption Key                     │
│  (Sold via Gumroad, Stripe, PayPal, Crypto, etc.)           │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 SMSG Container Format

See: `examples/formats/smsg-format.md`

Key properties:
- **Magic number**: "SMSG" (0x534D5347)
- **Algorithm**: ChaCha20-Poly1305 (authenticated encryption)
- **Format**: v1 (JSON+base64) or v2 (binary, 25% smaller)
- **Compression**: zstd (default), gzip, or none
- **Manifest**: Unencrypted metadata (title, artist, license, expiry, links)
- **Payload**: Encrypted media with attachments

#### Format Versions

| Format | Payload Structure | Size | Speed | Use Case |
|--------|------------------|------|-------|----------|
| **v1** | JSON with base64-encoded attachments | +33% overhead | Baseline | Legacy |
| **v2** | Binary header + raw attachments + zstd | ~Original size | 3-10x faster | Download-to-own |
| **v3** | CEK + wrapped keys + rolling LTHN | ~Original size | 3-10x faster | **Streaming** |
| **v3+chunked** | v3 with independently decryptable chunks | ~Original size | Seekable | **Chunked streaming** |

v2 is recommended for download-to-own (perpetual license). v3 is recommended for streaming (time-limited access). v3 with chunking is recommended for large files requiring seek capability or decrypt-while-downloading.

### 3.3 Key Derivation (v1/v2)

```
License Key (password)
        │
        ▼
   SHA-256 Hash
        │
        ▼
32-byte Symmetric Key
        │
        ▼
ChaCha20-Poly1305 Decryption
```

Simple, auditable, no key escrow.

**Note on password hashing**: SHA-256 is used for simplicity and speed. For high-value content, artists may choose to use stronger KDFs (Argon2, scrypt) in custom implementations. The format supports algorithm negotiation via the header.

### 3.4 Streaming Key Derivation (v3)

v3 format uses **LTHN rolling keys** for zero-trust streaming. The platform controls key refresh cadence.

```
┌──────────────────────────────────────────────────────────────────┐
│                    v3 STREAMING KEY FLOW                          │
├──────────────────────────────────────────────────────────────────┤
│                                                                    │
│  SERVER (encryption time):                                        │
│  ─────────────────────────                                        │
│  1. Generate random CEK (Content Encryption Key)                  │
│  2. Encrypt content with CEK (one-time)                          │
│  3. For current period AND next period:                          │
│     streamKey = SHA256(LTHN(period:license:fingerprint))         │
│     wrappedKey = ChaCha(CEK, streamKey)                          │
│  4. Store wrapped keys in header (CEK never transmitted)         │
│                                                                    │
│  CLIENT (decryption time):                                        │
│  ────────────────────────                                        │
│  1. Derive streamKey = SHA256(LTHN(period:license:fingerprint))  │
│  2. Try to unwrap CEK from current period key                    │
│  3. If fails, try next period key                                │
│  4. Decrypt content with unwrapped CEK                           │
│                                                                    │
└──────────────────────────────────────────────────────────────────┘
```

#### LTHN Hash Function

LTHN is rainbow-table resistant because the salt is derived from the input itself:

```
LTHN(input) = SHA256(input + reverse_leet(input))

where reverse_leet swaps: o↔0, l↔1, e↔3, a↔4, s↔z, t↔7

Example:
  LTHN("2026-01-12:license:fp")
  = SHA256("2026-01-12:license:fp" + "pf:3zn3ci1:21-10-6202")
```

You cannot compute the hash without knowing the original input.

#### Cadence Options

The platform chooses the key refresh rate. Faster cadence = tighter access control.

| Cadence | Period Format | Rolling Window | Use Case |
|---------|---------------|----------------|----------|
| `daily` | `2026-01-12` | 24-48 hours | Standard streaming |
| `12h` | `2026-01-12-AM/PM` | 12-24 hours | Premium content |
| `6h` | `2026-01-12-00/06/12/18` | 6-12 hours | High-value content |
| `1h` | `2026-01-12-15` | 1-2 hours | Live events |

The rolling window ensures smooth key transitions. At any time, both the current period key AND the next period key are valid.

#### Zero-Trust Properties

- **Server never stores keys** - Derived on-demand from LTHN
- **Keys auto-expire** - No revocation mechanism needed
- **Sharing keys is pointless** - They expire within the cadence window
- **Fingerprint binds to device** - Different device = different key
- **License ties to user** - Different user = different key

### 3.5 Chunked Streaming (v3 with ChunkSize)

When `StreamParams.ChunkSize > 0`, v3 format splits content into independently decryptable chunks, enabling:

- **Decrypt-while-downloading** - Play media as chunks arrive
- **HTTP Range requests** - Fetch specific chunks by byte offset
- **Seekable playback** - Jump to any position without decrypting previous chunks

```
┌──────────────────────────────────────────────────────────────────┐
│                    V3 CHUNKED FORMAT                              │
├──────────────────────────────────────────────────────────────────┤
│                                                                    │
│  Header (cleartext):                                              │
│    format: "v3"                                                   │
│    chunked: {                                                     │
│      chunkSize: 1048576,     // 1MB default                       │
│      totalChunks: N,                                              │
│      totalSize: X,           // unencrypted total                 │
│      index: [                // for HTTP Range / seeking          │
│        { offset: 0, size: Y },                                    │
│        { offset: Y, size: Z },                                    │
│        ...                                                        │
│      ]                                                            │
│    }                                                              │
│    wrappedKeys: [...]        // same as non-chunked v3           │
│                                                                    │
│  Payload:                                                         │
│    [chunk 0: nonce + encrypted + tag]                            │
│    [chunk 1: nonce + encrypted + tag]                            │
│    ...                                                            │
│    [chunk N: nonce + encrypted + tag]                            │
│                                                                    │
└──────────────────────────────────────────────────────────────────┘
```

**Key insight**: Each chunk is encrypted with the same CEK but gets its own random nonce, making chunks independently decryptable. The chunk index in the header enables:

1. **Seeking**: Calculate which chunk contains byte offset X, fetch just that chunk
2. **Range requests**: Use HTTP Range headers to fetch specific encrypted chunks
3. **Streaming**: Decrypt chunk 0 for metadata, then stream chunks 1-N as they arrive

**Usage example**:
```go
params := &StreamParams{
    License:     "user-license",
    Fingerprint: "device-fp",
    ChunkSize:   1024 * 1024,  // 1MB chunks
}

// Encrypt with chunking
encrypted, _ := EncryptV3(msg, params, manifest)

// For streaming playback:
header, _ := GetV3Header(encrypted)
cek, _ := UnwrapCEKFromHeader(header, params)
payload, _ := GetV3Payload(encrypted)

for i := 0; i < header.Chunked.TotalChunks; i++ {
    chunk, _ := DecryptV3Chunk(payload, cek, i, header.Chunked)
    player.Write(chunk)  // Stream to audio/video player
}
```

### 3.6 Supported Content Types

SMSG is content-agnostic. Any file can be an attachment:

| Type | MIME | Use Case |
|------|------|----------|
| Audio | audio/mpeg, audio/flac, audio/wav | Music, podcasts |
| Video | video/mp4, video/webm | Music videos, films |
| Images | image/png, image/jpeg | Album art, photos |
| Documents | application/pdf | Liner notes, lyrics |
| Archives | application/zip | Multi-file releases |
| Any | application/octet-stream | Anything else |

Multiple attachments per SMSG are supported (e.g., album + cover art + PDF booklet).

### 3.7 Adaptive Bitrate Streaming (ABR)

For large video content, ABR enables automatic quality switching based on network conditions—like HLS/DASH but with ChaCha20-Poly1305 encryption.

**Architecture:**
```
ABR Manifest (manifest.json)
├── Title: "My Video"
├── Version: "abr-v1"
├── Variants: [1080p, 720p, 480p, 360p]
└── DefaultIdx: 1 (720p)

track-1080p.smsg ──┐
track-720p.smsg  ──┼── Each is standard v3 chunked SMSG
track-480p.smsg  ──┤   Same password decrypts ALL variants
track-360p.smsg  ──┘
```

**ABR Manifest Format:**
```json
{
  "version": "abr-v1",
  "title": "Content Title",
  "duration": 300,
  "variants": [
    {
      "name": "360p",
      "bandwidth": 500000,
      "width": 640,
      "height": 360,
      "codecs": "avc1.640028,mp4a.40.2",
      "url": "track-360p.smsg",
      "chunkCount": 12,
      "fileSize": 18750000
    },
    {
      "name": "720p",
      "bandwidth": 2500000,
      "width": 1280,
      "height": 720,
      "codecs": "avc1.640028,mp4a.40.2",
      "url": "track-720p.smsg",
      "chunkCount": 48,
      "fileSize": 93750000
    }
  ],
  "defaultIdx": 1
}
```

**Bandwidth Estimation Algorithm:**
1. Measure download time for each chunk
2. Calculate bits per second: `(bytes × 8 × 1000) / timeMs`
3. Average last 3 samples for stability
4. Apply 80% safety factor to prevent buffering

**Variant Selection:**
```
Selected = highest quality where (bandwidth × 0.8) >= variant.bandwidth
```

**Key Properties:**
- **Same password for all variants**: CEK unwrapped once, works everywhere
- **Chunk-boundary switching**: Clean cuts, no partial chunk issues
- **Independent variants**: No cross-file dependencies
- **CDN-friendly**: Each variant is a standard file, cacheable separately

**Creating ABR Content:**
```bash
# Use mkdemo-abr to create variant set from source video
go run ./cmd/mkdemo-abr input.mp4 output-dir/ [password]

# Output:
#   output-dir/manifest.json     (ABR manifest)
#   output-dir/track-1080p.smsg  (v3 chunked, 5 Mbps)
#   output-dir/track-720p.smsg   (v3 chunked, 2.5 Mbps)
#   output-dir/track-480p.smsg   (v3 chunked, 1 Mbps)
#   output-dir/track-360p.smsg   (v3 chunked, 500 Kbps)
```

**Standard Presets:**

| Name | Resolution | Bitrate | Use Case |
|------|------------|---------|----------|
| 1080p | 1920×1080 | 5 Mbps | High quality, fast connections |
| 720p | 1280×720 | 2.5 Mbps | Default, most connections |
| 480p | 854×480 | 1 Mbps | Mobile, medium connections |
| 360p | 640×360 | 500 Kbps | Slow connections, previews |

## 4. Demo Page Architecture

**Live Demo**: https://demo.dapp.fm

### 4.1 Components

```
demo/
├── index.html          # Single-page application
├── stmf.wasm           # Go WASM decryption module (~5.9MB)
├── wasm_exec.js        # Go WASM runtime
├── demo-track.smsg     # Sample encrypted content (v2/zstd)
└── profile-avatar.jpg  # Artist avatar
```

### 4.2 UI Modes

The demo has three modes, accessible via tabs:

| Mode | Purpose | Default |
|------|---------|---------|
| **Profile** | Artist landing page with auto-playing content | Yes |
| **Fan** | Upload and decrypt purchased .smsg files | No |
| **Artist** | Re-key content, create new packages | No |

### 4.3 Profile Mode (Default)

```
┌─────────────────────────────────────────────────────────────┐
│  dapp.fm                    [Profile] [Fan] [Artist]        │
├─────────────────────────────────────────────────────────────┤
│  Zero-Trust DRM        ⚠️ Demo pre-seeded with keys         │
├─────────────────────────────────────────────────────────────┤
│  [No Middlemen] [No Fees] [Host Anywhere] [Browser/Native]  │
├─────────────────┬───────────────────────────────────────────┤
│   SIDEBAR       │              MAIN CONTENT                  │
│  ┌───────────┐  │  ┌─────────────────────────────────────┐  │
│  │  Avatar   │  │  │  🛒 Buy This Track on Beatport      │  │
│  │           │  │  │  95%-100%* goes to the artist       │  │
│  │  Artist   │  │  ├─────────────────────────────────────┤  │
│  │  Name     │  │  │                                     │  │
│  │           │  │  │     VIDEO PLAYER                    │  │
│  │  Links:   │  │  │     (auto-starts at 1:08)          │  │
│  │  Beatport │  │  │     with native controls           │  │
│  │  Spotify  │  │  │                                     │  │
│  │  YouTube  │  │  ├─────────────────────────────────────┤  │
│  │  etc.     │  │  │  About the Artist                   │  │
│  └───────────┘  │  │  (Bio text)                         │  │
│                 │  └─────────────────────────────────────┘  │
├─────────────────┴───────────────────────────────────────────┤
│  GitHub · EUPL-1.2 · Viva La OpenSource 💜                  │
└─────────────────────────────────────────────────────────────┘
```

### 4.4 Decryption Flow

```
User clicks "Play Demo Track"
        │
        ▼
fetch(demo-track.smsg)
        │
        ▼
Convert to base64 ◄─── CRITICAL: Must handle binary vs text format
        │                See: examples/failures/001-double-base64-encoding.md
        ▼
BorgSMSG.getInfo(base64)
        │
        ▼
Display manifest (title, artist, license)
        │
        ▼
BorgSMSG.decryptStream(base64, password)
        │
        ▼
Create Blob from Uint8Array
        │
        ▼
URL.createObjectURL(blob)
        │
        ▼
<audio> or <video> element plays content
```

### 4.5 Fan Unlock Tab

Allows fans to:
1. Upload any `.smsg` file they purchased
2. Enter their license key (password)
3. Decrypt and play locally

No server communication - everything in browser.

## 5. Artist Portal (License Manager)

The License Manager (`js/borg-stmf/artist-portal.html`) is the artist-facing tool for creating and issuing licenses.

### 5.1 Workflow

```
┌─────────────────────────────────────────────────────────────┐
│                    ARTIST PORTAL                             │
├─────────────────────────────────────────────────────────────┤
│  1. Upload Content                                           │
│     - Drag/drop audio or video file                         │
│     - Or use demo content for testing                       │
├─────────────────────────────────────────────────────────────┤
│  2. Define Track List (CD Mastering)                        │
│     - Track titles                                           │
│     - Start/end timestamps → chapter markers                │
│     - Mix types (full, intro, chorus, drop, etc.)           │
├─────────────────────────────────────────────────────────────┤
│  3. Configure License                                        │
│     - Perpetual (own forever)                               │
│     - Rental (time-limited)                                 │
│     - Streaming (24h access)                                │
│     - Preview (30 seconds)                                  │
├─────────────────────────────────────────────────────────────┤
│  4. Generate License                                         │
│     - Auto-generate token or set custom                     │
│     - Token encrypts content with manifest                  │
│     - Download .smsg file                                   │
├─────────────────────────────────────────────────────────────┤
│  5. Distribute                                               │
│     - Upload .smsg to CDN/IPFS/S3                           │
│     - Sell license token via payment processor              │
│     - Fan receives token, downloads .smsg, plays            │
└─────────────────────────────────────────────────────────────┘
```

### 5.2 License Types

| Type | Duration | Use Case |
|------|----------|----------|
| **Perpetual** | Forever | Album purchase, own forever |
| **Rental** | 7-90 days | Limited edition, seasonal content |
| **Streaming** | 24 hours | On-demand streaming model |
| **Preview** | 30 seconds | Free samples, try-before-buy |

### 5.3 Track List as Manifest

The artist defines tracks like mastering a CD:

```json
{
  "tracks": [
    {"title": "Intro", "start": 0, "end": 45, "type": "intro"},
    {"title": "Main Track", "start": 45, "end": 240, "type": "full"},
    {"title": "The Drop", "start": 120, "end": 180, "type": "drop"},
    {"title": "Outro", "start": 240, "end": 300, "type": "outro"}
  ]
}
```

Same master file, different licensed "cuts":
- **Full Album**: All tracks, perpetual
- **Radio Edit**: Tracks 2-3 only, rental
- **DJ Extended**: Loop points enabled, perpetual
- **Preview**: First 30 seconds, expires immediately

### 5.4 Stats Dashboard

The Artist Portal tracks:
- Total licenses issued
- Potential revenue (based on entered prices)
- 100% cut (reminder: no platform fees)

## 6. Economic Model

### 6.1 The Offer

**Self-host for 0%. Let us host for 5%.**

That's it. No hidden fees, no per-stream calculations, no "recoupable advances."

| Option | Cut | What You Get |
|--------|-----|--------------|
| **Self-host** | 0% | Tools, format, documentation. Host on your own CDN/IPFS/server |
| **dapp.fm hosted** | 5% | CDN, player embed, analytics, payment integration |

Compare to:
- Spotify: ~30% of $0.003/stream (you need 300k streams to earn $1000)
- Apple Music: ~30%
- Bandcamp: ~15-20%
- DistroKid: Flat fee but still platform-dependent

### 6.2 License Key Strategies

Artists can choose their pricing model:

**Per-Album License**
```
Album: "My Greatest Hits"
Price: $10
License: "MGH-2024-XKCD-7829"
→ One password unlocks entire album
```

**Per-Track License**
```
Track: "Single Release"
Price: $1
License: "SINGLE-A7B3-C9D2"
→ Individual track, individual price
```

**Tiered Licenses**
```
Standard: $10 → MP3 version
Premium: $25 → FLAC + stems + bonus content
→ Different passwords, different content
```

**Time-Limited Previews**
```
Preview license expires in 7 days
Full license: permanent
→ Manifest contains expiry date
```

### 6.3 License Key Best Practices

For artists generating license keys:

```bash
# Good: Memorable but unique
MGH-2024-XKCD-7829
ALBUM-[year]-[random]-[checksum]

# Good: UUID for automation
550e8400-e29b-41d4-a716-446655440000

# Avoid: Dictionary words (bruteforceable)
password123
mysecretalbum
```

Recommended entropy: 64+ bits (e.g., 4 random words, or 12+ random alphanumeric)

### 6.4 No Revocation (By Design)

**Q: What if someone leaks the password?**

A: Then they leak it. Same as if someone photocopies a book or rips a CD.

This is a feature, not a bug:
- **No revocation server** = No single point of failure
- **No phone home** = Works offline, forever
- **Leaked keys** = Social problem, not technical problem

Mitigation strategies for artists:
1. Personalized keys per buyer (track who leaked)
2. Watermarked content (forensic tracking)
3. Time-limited keys for subscription models
4. Social pressure (small community = reputation matters)

The system optimizes for **happy paying customers**, not **punishing pirates**.

## 7. Security Model

### 7.1 Threat Model

| Threat | Mitigation |
|--------|------------|
| Man-in-the-middle | Content encrypted at rest; HTTPS for transport |
| Key server compromise | No key server - password-derived keys |
| Platform deplatforming | Self-hostable, decentralized distribution |
| Unauthorized sharing | Economic/social deterrent (password = paid license) |
| Memory extraction | Accepted risk - same as any DRM |

### 7.2 What This System Does NOT Prevent

- Users sharing their password (same as sharing any license)
- Screen recording of playback
- Memory dumping of decrypted content

This is **intentional**. The goal is not unbreakable DRM (which is impossible) but:
1. Making casual piracy inconvenient
2. Giving artists control of their distribution
3. Enabling direct artist-to-fan sales
4. Removing platform dependency

### 7.3 Trust Boundaries

```
TRUSTED                           UNTRUSTED
────────                          ─────────
User's browser/device             Distribution CDN
Decryption code (auditable)       Payment processor
License key (in user's head)      Internet transport
Local playback                    Third-party hosting
```

## 8. Implementation Status

### 8.1 Completed
- [x] SMSG format specification (v1, v2, v3)
- [x] Go encryption/decryption library (pkg/smsg)
- [x] WASM build for browser (pkg/wasm/stmf)
- [x] Native desktop app (Wails, cmd/dapp-fm-app)
- [x] Demo page with Profile/Fan/Artist modes
- [x] License Manager component
- [x] Streaming decryption API (v1.2.0)
- [x] **v2 binary format** - 25% smaller files
- [x] **zstd compression** - 3-10x faster than gzip
- [x] **Manifest links** - Artist platform links in metadata
- [x] **Live demo** - https://demo.dapp.fm
- [x] RFC-quality demo file with cryptographically secure password
- [x] **v3 streaming format** - LTHN rolling keys with CEK wrapping
- [x] **Configurable cadence** - daily/12h/6h/1h key rotation
- [x] **WASM v1.3.0** - `BorgSMSG.decryptV3()` for streaming
- [x] **Chunked streaming** - Independently decryptable chunks for seek/streaming
- [x] **Adaptive Bitrate (ABR)** - HLS-style multi-quality streaming with encrypted variants

### 8.2 Fixed Issues
- [x] ~~Double base64 encoding bug~~ - Fixed by using binary format
- [x] ~~Demo file format detection~~ - v2 format auto-detected via header
- [x] ~~Key wrapping for streaming~~ - Implemented in v3 format

### 8.3 Future Work
- [x] Multi-bitrate adaptive streaming (see Section 3.7 ABR)
- [x] Payment integration examples (see `docs/payment-integration.md`)
- [x] IPFS distribution guide (see `docs/ipfs-distribution.md`)
- [x] Demo page "Streaming" tab for v3 showcase

## 9. Usage Examples

### 9.1 Artist Workflow

```bash
# 1. Package your media (uses v2 binary format + zstd by default)
go run ./cmd/mkdemo my-track.mp4 my-track.smsg
# Output:
#   Created: my-track.smsg (29220077 bytes)
#   Master Password: PMVXogAJNVe_DDABfTmLYztaJAzsD0R7
#   Store this password securely - it cannot be recovered!

# Or programmatically:
msg := smsg.NewMessage("Welcome to my album")
msg.AddBinaryAttachment("track.mp4", mediaBytes, "video/mp4")
manifest := smsg.NewManifest("Track Title")
manifest.Artist = "Artist Name"
manifest.AddLink("home", "https://linktr.ee/artist")
encrypted, _ := smsg.EncryptV2WithManifest(msg, password, manifest)

# 2. Upload to any hosting
aws s3 cp my-track.smsg s3://my-bucket/releases/
# or: ipfs add my-track.smsg
# or: scp my-track.smsg myserver:/var/www/

# 3. Sell license keys
# Use Gumroad, Stripe, PayPal - any payment method
# Deliver the master password on purchase
```

### 9.2 Fan Workflow

```
1. Purchase from artist's website → receive license key
2. Download .smsg file from CDN/IPFS/wherever
3. Open demo page or native app
4. Enter license key
5. Content decrypts and plays locally
```

### 9.3 Browser Integration

```html
<script src="wasm_exec.js"></script>
<script src="stmf.wasm.js"></script>
<script>
async function playContent(smsgUrl, licenseKey) {
    const response = await fetch(smsgUrl);
    const bytes = new Uint8Array(await response.arrayBuffer());
    const base64 = arrayToBase64(bytes);  // Must be binary→base64

    const msg = await BorgSMSG.decryptStream(base64, licenseKey);

    const blob = new Blob([msg.attachments[0].data], {
        type: msg.attachments[0].mime
    });
    document.querySelector('audio').src = URL.createObjectURL(blob);
}
</script>
```

## 10. Comparison to Existing Solutions

| Feature | dapp.fm (self) | dapp.fm (hosted) | Spotify | Bandcamp | Widevine |
|---------|----------------|------------------|---------|----------|----------|
| Artist revenue | **100%** | **95%** | ~30% | ~80% | N/A |
| Platform cut | **0%** | **5%** | ~70% | ~15-20% | Varies |
| Self-hostable | Yes | Optional | No | No | No |
| Open source | Yes | Yes | No | No | No |
| Key escrow | None | None | Required | Required | Required |
| Browser support | WASM | WASM | Web | Web | CDM |
| Offline support | Yes | Yes | Premium | Download | Depends |
| Platform lock-in | **None** | **None** | High | Medium | High |
| Works if platform dies | **Yes** | **Yes** | No | No | No |

## 11. Interoperability & Versioning

### 11.1 Format Versioning

SMSG includes version and format fields for forward compatibility:

| Version | Format | Features |
|---------|--------|----------|
| 1.0 | v1 | ChaCha20-Poly1305, JSON+base64 attachments |
| 1.0 | **v2** | Binary attachments, zstd compression (25% smaller, 3-10x faster) |
| 1.0 | **v3** | LTHN rolling keys, CEK wrapping, chunked streaming |
| 1.0 | **v3+ABR** | Multi-quality variants with adaptive bitrate switching |
| 2 (future) | - | Algorithm negotiation, multiple KDFs |

Decoders MUST reject versions they don't understand. Use v2 for download-to-own, v3 for streaming, v3+ABR for video.

### 11.2 Third-Party Implementations

The format is intentionally simple to implement:

**Minimum Viable Player (any language)**:
1. Parse 4-byte magic ("SMSG")
2. Read version (2 bytes) and header length (4 bytes)
3. Parse JSON header
4. SHA-256 hash the password
5. ChaCha20-Poly1305 decrypt payload
6. Parse JSON payload, extract attachments

Reference implementations:
- Go: `pkg/smsg/` (canonical)
- WASM: `pkg/wasm/stmf/` (browser)
- (contributions welcome: Rust, Python, JS-native)

### 11.3 Embedding & Integration

SMSG files can be:
- **Embedded in HTML**: Base64 in data attributes
- **Served via API**: JSON wrapper with base64 content
- **Bundled in apps**: Compiled into native binaries
- **Stored on IPFS**: Content-addressed, immutable
- **Distributed via torrents**: Encrypted = safe to share publicly

The player is embeddable:
```html
<iframe src="https://dapp.fm/embed/HASH" width="400" height="200"></iframe>
```

## 12. References

- **Live Demo**: https://demo.dapp.fm
- ChaCha20-Poly1305: RFC 8439
- zstd compression: https://github.com/klauspost/compress/tree/master/zstd
- SMSG Format: `examples/formats/smsg-format.md`
- Demo Page Source: `demo/index.html`
- WASM Module: `pkg/wasm/stmf/`
- Native App: `cmd/dapp-fm-app/`
- Demo Creator Tool: `cmd/mkdemo/`
- ABR Creator Tool: `cmd/mkdemo-abr/`
- ABR Package: `pkg/smsg/abr.go`

## 13. License

This specification and implementation are licensed under EUPL-1.2.

**Viva La OpenSource** 💜
