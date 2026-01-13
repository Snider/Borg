# RFC-010: WASM Decryption API

**Status**: Draft
**Author**: [Snider](https://github.com/Snider/)
**Created**: 2026-01-13
**License**: EUPL-1.2
**Depends On**: RFC-002, RFC-007, RFC-009

---

## Abstract

This RFC specifies the WebAssembly (WASM) API for browser-based decryption of SMSG content and STMF form encryption. The API is exposed through two JavaScript namespaces: `BorgSMSG` for content decryption and `BorgSTMF` for form encryption.

## 1. Overview

The WASM module provides:
- SMSG decryption (v1, v2, v3, chunked, ABR)
- SMSG encryption
- STMF form encryption/decryption
- Metadata extraction without decryption

## 2. Module Loading

### 2.1 Files Required

```
stmf.wasm       (~5.9MB)  Compiled Go WASM module
wasm_exec.js    (~20KB)   Go WASM runtime
```

### 2.2 Initialization

```html
<script src="wasm_exec.js"></script>
<script>
const go = new Go();
WebAssembly.instantiateStreaming(fetch('stmf.wasm'), go.importObject)
    .then(result => {
        go.run(result.instance);
        // BorgSMSG and BorgSTMF now available globally
    });
</script>
```

### 2.3 Ready Event

```javascript
document.addEventListener('borgstmf:ready', (event) => {
    console.log('WASM ready, version:', event.detail.version);
});
```

## 3. BorgSMSG Namespace

### 3.1 Version

```javascript
BorgSMSG.version  // "1.6.0"
BorgSMSG.ready    // true when loaded
```

### 3.2 Metadata Functions

#### getInfo(base64) → Promise<ManifestInfo>

Get manifest without decryption.

```javascript
const info = await BorgSMSG.getInfo(base64Content);
// info.version, info.algorithm, info.format
// info.manifest.title, info.manifest.artist
// info.isV3Streaming, info.isChunked
// info.wrappedKeys (for v3)
```

#### getInfoBinary(uint8Array) → Promise<ManifestInfo>

Binary input variant (no base64 decode needed).

```javascript
const bytes = new Uint8Array(await response.arrayBuffer());
const info = await BorgSMSG.getInfoBinary(bytes);
```

### 3.3 Decryption Functions

#### decrypt(base64, password) → Promise<Message>

Full decryption (v1 format, base64 attachments).

```javascript
const msg = await BorgSMSG.decrypt(base64Content, password);
// msg.body, msg.subject, msg.from
// msg.attachments[0].name, .content (base64), .mime
```

#### decryptStream(base64, password) → Promise<StreamMessage>

Streaming decryption (v2 format, binary attachments).

```javascript
const msg = await BorgSMSG.decryptStream(base64Content, password);
// msg.attachments[0].data (Uint8Array)
// msg.attachments[0].mime
```

#### decryptBinary(uint8Array, password) → Promise<StreamMessage>

Binary input, binary output.

```javascript
const bytes = new Uint8Array(await fetch(url).then(r => r.arrayBuffer()));
const msg = await BorgSMSG.decryptBinary(bytes, password);
```

#### quickDecrypt(base64, password) → Promise<string>

Returns body text only (fast path).

```javascript
const body = await BorgSMSG.quickDecrypt(base64Content, password);
```

### 3.4 V3 Streaming Functions

#### decryptV3(base64, params) → Promise<StreamMessage>

Decrypt v3 streaming content with LTHN rolling keys.

```javascript
const msg = await BorgSMSG.decryptV3(base64Content, {
    license: "user-license-key",
    fingerprint: "device-fingerprint"  // optional
});
```

#### getV3ChunkInfo(base64) → Promise<ChunkInfo>

Get chunk index for seeking without full decrypt.

```javascript
const chunkInfo = await BorgSMSG.getV3ChunkInfo(base64Content);
// chunkInfo.chunkSize (default 1MB)
// chunkInfo.totalChunks
// chunkInfo.totalSize
// chunkInfo.index[i].offset, .size
```

#### unwrapV3CEK(base64, params) → Promise<string>

Unwrap CEK for manual chunk decryption. Returns base64 CEK.

```javascript
const cekBase64 = await BorgSMSG.unwrapV3CEK(base64Content, {
    license: "license",
    fingerprint: "fp"
});
```

#### decryptV3Chunk(base64, cekBase64, chunkIndex) → Promise<Uint8Array>

Decrypt single chunk by index.

```javascript
const chunk = await BorgSMSG.decryptV3Chunk(base64Content, cekBase64, 5);
```

#### parseV3Header(uint8Array) → Promise<V3HeaderInfo>

Parse header from partial data (for streaming).

```javascript
const header = await BorgSMSG.parseV3Header(bytes);
// header.format, header.keyMethod, header.cadence
// header.payloadOffset (where chunks start)
// header.wrappedKeys, header.chunked, header.manifest
```

#### unwrapCEKFromHeader(wrappedKeys, params, cadence) → Promise<Uint8Array>

Unwrap CEK from parsed header.

```javascript
const cek = await BorgSMSG.unwrapCEKFromHeader(
    header.wrappedKeys,
    {license: "lic", fingerprint: "fp"},
    "daily"
);
```

#### decryptChunkDirect(chunkBytes, cek) → Promise<Uint8Array>

Low-level chunk decryption with pre-unwrapped CEK.

```javascript
const plaintext = await BorgSMSG.decryptChunkDirect(chunkBytes, cek);
```

### 3.5 Encryption Functions

#### encrypt(message, password, hint?) → Promise<string>

Encrypt message (v1 format). Returns base64.

```javascript
const encrypted = await BorgSMSG.encrypt({
    body: "Hello",
    attachments: [{
        name: "file.txt",
        content: btoa("data"),
        mime: "text/plain"
    }]
}, password, "optional hint");
```

#### encryptWithManifest(message, password, manifest) → Promise<string>

Encrypt with manifest (v2 format). Returns base64.

```javascript
const encrypted = await BorgSMSG.encryptWithManifest(message, password, {
    title: "My Track",
    artist: "Artist Name",
    licenseType: "perpetual"
});
```

### 3.6 ABR Functions

#### parseABRManifest(jsonString) → Promise<ABRManifest>

Parse HLS-style ABR manifest.

```javascript
const manifest = await BorgSMSG.parseABRManifest(manifestJson);
// manifest.version, manifest.title, manifest.duration
// manifest.variants[i].name, .bandwidth, .url
// manifest.defaultIdx
```

#### selectVariant(manifest, bandwidthBps) → Promise<number>

Select best variant for bandwidth (returns index).

```javascript
const idx = await BorgSMSG.selectVariant(manifest, measuredBandwidth);
// Uses 80% safety threshold
```

## 4. BorgSTMF Namespace

### 4.1 Key Generation

```javascript
const keypair = await BorgSTMF.generateKeyPair();
// keypair.publicKey (base64 X25519)
// keypair.privateKey (base64 X25519) - KEEP SECRET
```

### 4.2 Encryption

```javascript
// Encrypt JSON string
const encrypted = await BorgSTMF.encrypt(
    JSON.stringify(formData),
    serverPublicKeyBase64
);

// Encrypt with metadata
const encrypted = await BorgSTMF.encryptFields(
    {email: "user@example.com", password: "secret"},
    serverPublicKeyBase64,
    {timestamp: Date.now().toString()}  // optional metadata
);
```

## 5. Type Definitions

### 5.1 ManifestInfo

```typescript
interface ManifestInfo {
    version: string;
    algorithm: string;
    format?: string;
    compression?: string;
    hint?: string;
    keyMethod?: string;      // "LTHN" for v3
    cadence?: string;        // "daily", "12h", "6h", "1h"
    wrappedKeys?: WrappedKey[];
    isV3Streaming: boolean;
    chunked?: ChunkInfo;
    isChunked: boolean;
    manifest?: Manifest;
}
```

### 5.2 Message / StreamMessage

```typescript
interface Message {
    from?: string;
    to?: string;
    subject?: string;
    body: string;
    timestamp?: number;
    attachments: Attachment[];
    replyKey?: KeyInfo;
    meta?: Record<string, string>;
}

interface Attachment {
    name: string;
    mime: string;
    size: number;
    content?: string;      // base64 (v1)
    data?: Uint8Array;     // binary (v2/v3)
}
```

### 5.3 ChunkInfo

```typescript
interface ChunkInfo {
    chunkSize: number;      // default 1048576 (1MB)
    totalChunks: number;
    totalSize: number;
    index: ChunkEntry[];
}

interface ChunkEntry {
    offset: number;
    size: number;
}
```

### 5.4 Manifest

```typescript
interface Manifest {
    title: string;
    artist?: string;
    album?: string;
    genre?: string;
    year?: number;
    releaseType?: string;   // "single", "album", "ep", "mix"
    duration?: number;      // seconds
    format?: string;
    expiresAt?: number;     // Unix timestamp
    issuedAt?: number;      // Unix timestamp
    licenseType?: string;   // "perpetual", "rental", "stream", "preview"
    tracks?: Track[];
    tags?: string[];
    links?: Record<string, string>;
    extra?: Record<string, string>;
}
```

## 6. Error Handling

### 6.1 Pattern

All functions throw on error:

```javascript
try {
    const msg = await BorgSMSG.decrypt(content, password);
} catch (e) {
    console.error(e.message);
}
```

### 6.2 Common Errors

| Error | Cause |
|-------|-------|
| `decrypt requires 2 arguments` | Wrong argument count |
| `decryption failed: {reason}` | Wrong password or corrupted |
| `invalid format` | Not a valid SMSG file |
| `unsupported version` | Unknown format version |
| `key expired` | v3 rolling key outside window |
| `invalid base64: {reason}` | Base64 decode failed |
| `chunk out of range` | Invalid chunk index |

## 7. Performance

### 7.1 Binary vs Base64

- Binary functions (`*Binary`, `decryptStream`) are ~30% faster
- Avoid double base64 encoding

### 7.2 Large Files (>50MB)

Use chunked streaming:

```javascript
// Efficient: Cache CEK, stream chunks
const header = await BorgSMSG.parseV3Header(bytes);
const cek = await BorgSMSG.unwrapCEKFromHeader(header.wrappedKeys, params);

for (let i = 0; i < header.chunked.totalChunks; i++) {
    const chunk = await BorgSMSG.decryptChunkDirect(payload, cek);
    player.write(chunk);
    // chunk is GC'd after each iteration
}
```

### 7.3 Typical Execution Times

| Operation | Size | Time |
|-----------|------|------|
| getInfo | any | ~50-100ms |
| decrypt (small) | <1MB | ~200-500ms |
| decrypt (large) | 100MB | 2-5s |
| decryptV3Chunk | 1MB | ~200-400ms |
| generateKeyPair | - | ~50-200ms |

## 8. Browser Compatibility

| Browser | Support |
|---------|---------|
| Chrome 57+ | Full |
| Firefox 52+ | Full |
| Safari 11+ | Full |
| Edge 16+ | Full |
| IE | Not supported |

Requirements:
- WebAssembly support
- Async/await (ES2017)
- Uint8Array

## 9. Memory Management

- WASM module: ~5.9MB static
- Per-operation: Peak ~2-3x file size during decryption
- Go GC reclaims after Promise resolution
- Keys never leave WASM memory

## 10. Implementation Reference

- Source: `pkg/wasm/stmf/main.go` (1758 lines)
- Build: `GOOS=js GOARCH=wasm go build -o stmf.wasm ./pkg/wasm/stmf/`

## 11. Security Considerations

1. **Password handling**: Clear from memory after use
2. **Memory isolation**: WASM sandbox prevents JS access
3. **Constant-time crypto**: Go crypto uses safe operations
4. **Key protection**: Keys never exposed to JavaScript

## 12. Future Work

- [ ] WebWorker support for background decryption
- [ ] Streaming API with ReadableStream
- [ ] Smaller WASM size via TinyGo
- [ ] Native Web Crypto fallback for simple operations
