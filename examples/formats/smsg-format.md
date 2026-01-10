# SMSG Format Specification

## Overview
SMSG (Secure Message) is an encrypted container format using ChaCha20-Poly1305 authenticated encryption.

## File Structure

```
┌─────────────────────────────────────────┐
│ Magic Number: "SMSG" (4 bytes)          │
├─────────────────────────────────────────┤
│ Version: uint16 (2 bytes)               │
├─────────────────────────────────────────┤
│ Header Length: uint32 (4 bytes)         │
├─────────────────────────────────────────┤
│ Header (JSON, plaintext)                │
│ - algorithm: "chacha20poly1305"         │
│ - manifest: {title, artist, license...} │
│ - nonce: base64                         │
├─────────────────────────────────────────┤
│ Encrypted Payload                       │
│ - Nonce (24 bytes for XChaCha20)        │
│ - Ciphertext + Auth Tag                 │
└─────────────────────────────────────────┘
```

## Magic Number
- Binary: `0x53 0x4D 0x53 0x47`
- ASCII: "SMSG"
- Base64 (first 6 chars): "U01TRw"

## Header (JSON, unencrypted)
```json
{
  "algorithm": "chacha20poly1305",
  "manifest": {
    "title": "Track Title",
    "artist": "Artist Name",
    "license": "CC-BY-4.0",
    "expires": "2025-12-31T23:59:59Z",
    "tracks": [
      {"title": "Track 1", "start": 0, "trackNum": 1}
    ]
  }
}
```

The manifest is **readable without decryption** - this enables:
- License validation before decryption
- Metadata display in file browsers
- Expiration enforcement

## Encrypted Payload (JSON)
```json
{
  "from": "artist@example.com",
  "to": "fan@example.com",
  "subject": "Album Title",
  "body": "Thank you for your purchase!",
  "attachments": [
    {
      "name": "track.mp3",
      "mime": "audio/mpeg",
      "content": "<base64-encoded-data>"
    }
  ]
}
```

## Key Derivation
```
password → SHA-256 → 32-byte key
```

Simple but effective - the password IS the license key.

## Storage Formats

### Binary (.smsg)
Raw bytes. Canonical format for distribution.
```
53 4D 53 47 02 00 00 00 33 00 00 00 7B 22 61 6C ...
S  M  S  G  [ver]  [hdr len]  {"al...
```

### Base64 Text (.smsg or .smsg.b64)
For embedding in JSON, URLs, or text-based transport.
```
U01TRwIAAAEzeyJhbGdvcml0aG0iOiJjaGFjaGEyMHBvbHkxMzA1Ii...
```

## WASM API

```javascript
// Initialize
const go = new Go();
await WebAssembly.instantiateStreaming(fetch('stmf.wasm'), go.importObject);
go.run(result.instance);

// Get metadata (no password needed)
const info = await BorgSMSG.getInfo(base64Content);
// info.manifest.title, info.manifest.expires, etc.

// Decrypt (requires password)
const msg = await BorgSMSG.decryptStream(base64Content, password);
// msg.attachments[0].data is Uint8Array (binary)
// msg.attachments[0].mime is MIME type
```

## Security Properties

1. **Authenticated Encryption**: ChaCha20-Poly1305 provides both confidentiality and integrity
2. **No Key Escrow**: Password never transmitted, derived locally
3. **Metadata Privacy**: Only manifest is public; actual content encrypted
4. **Browser-Safe**: WASM runs in sandbox, keys never leave client

## Use Cases

| Use Case | Format | Notes |
|----------|--------|-------|
| Direct download | Binary | Most efficient |
| Email attachment | Base64 | Safe for text transport |
| IPFS/CDN | Binary | Content-addressed |
| Embedded in JSON | Base64 | API responses |
| Browser demo | Either | Must detect format |
