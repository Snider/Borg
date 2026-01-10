# Failure Case 001: Double Base64 Encoding

## Error Message
```
Failed: decryption failed: invalid SMSG magic: trix: invalid magic number: expected SMSG, got U01T
```

## Environment
- Demo page: `demo/index.html`
- File: `demo/demo-track.smsg`
- WASM version: 1.2.0

## Root Cause Analysis

### The Problem
The demo file `demo-track.smsg` is stored as **base64-encoded text**, but the JavaScript code treats it as binary and re-encodes it to base64 before passing to WASM.

### Evidence

File inspection:
```bash
$ file demo/demo-track.smsg
ASCII text, with very long lines (65536), with no line terminators

$ head -c 64 demo/demo-track.smsg | xxd
00000000: 5530 3154 5277 4941 4141 457a 6579 4a68  U01TRwIAAAEzeyJh
```

The file starts with `U01TRwIA...` which is **base64-encoded SMSG**:
- `U01TRw` decodes to bytes `0x53 0x4D 0x53 0x47` = "SMSG" (the magic number)

### The Double-Encoding Chain

```
Original SMSG binary:
  SMSG.... (starts with 0x534D5347)
       ↓ base64 encode (file storage)
  U01TRwIA... (stored in demo-track.smsg)
       ↓ fetch() as binary
  [0x55, 0x30, 0x31, 0x54, ...] (bytes of ASCII "U01T...")
       ↓ btoa() in JavaScript
  VTAxVFJ3SUFBQUUzZXlK... (base64 of base64!)
       ↓ WASM base64 decode
  U01TRwIA... (back to first base64)
       ↓ WASM tries to parse as SMSG
  ERROR: expected "SMSG", got "U01T" (first 4 chars of base64)
```

### Why "U01T"?
The error shows "U01T" because when WASM decodes the double-base64, it gets back the original base64 string, and the first 4 ASCII characters "U01T" are interpreted as the magic number instead of the actual bytes 0x534D5347.

## Solution Options

### Option A: Store as binary (recommended)
Convert the demo file to raw binary format:
```bash
base64 -d demo/demo-track.smsg > demo/demo-track-binary.smsg
mv demo/demo-track-binary.smsg demo/demo-track.smsg
```

### Option B: Detect format in JavaScript
Check if content is already base64 and skip re-encoding:
```javascript
// Check if content looks like base64 (ASCII text starting with valid base64 chars)
const isBase64 = /^[A-Za-z0-9+/=]+$/.test(text.trim());
if (!isBase64) {
    // Binary content - encode to base64
    base64 = btoa(binaryToString(bytes));
} else {
    // Already base64 - use as-is
    base64 = text;
}
```

### Option C: Use text fetch for base64 files
```javascript
// For base64-encoded .smsg files
const response = await fetch(DEMO_URL);
const base64 = await response.text(); // Don't re-encode
```

## Lesson Learned
SMSG files can exist in two formats:
1. **Binary** (.smsg) - raw bytes, magic number is `0x534D5347`
2. **Base64** (.smsg.b64 or .smsg with text content) - ASCII text, starts with `U01T`

The loader must detect which format it's receiving and handle accordingly.

## Recommended Fix
Implement Option A (binary storage) for the demo, as it's the canonical format and avoids ambiguity. Reserve Option B for the License Manager where users might drag-drop either format.

## Related
- `pkg/smsg/smsg.go` - SMSG format definition
- `pkg/wasm/stmf/main.go` - WASM decryption API
- `demo/index.html` - Demo page loader
