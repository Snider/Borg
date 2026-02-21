# Borg Production Backup Upgrade — Design Document

**Date:** 2026-02-21
**Status:** Approved
**Approach:** Bottom-Up Refactor

## Problem Statement

Borg's `collect local` command fails on large directories because DataNode loads
everything into RAM. The UI spinner floods non-TTY output. Broken symlinks crash
the collection pipeline. Key derivation uses bare SHA-256. These issues prevent
Borg from being used for production backup workflows.

## Goals

1. Make `collect local` work reliably on large directories (10GB+)
2. Handle symlinks properly (skip broken, follow/store valid)
3. Add quiet/scripted mode for cron and pipeline use
4. Harden encryption key derivation (Argon2id)
5. Clean up the library for external consumers

## Non-Goals

- Full core/go-* package integration (deferred — circular dependency risk since
  core imports Borg)
- New CLI commands beyond fixing existing ones
- Network transport or remote sync features
- GUI or web interface

## Architecture

### Current Flow (Broken for Large Dirs)

```
Walk directory → Load ALL files into DataNode (RAM) → Compress → Encrypt → Write
```

### New Flow (Streaming)

```
Walk directory → tar.Writer stream → compress stream → chunked encrypt → output file
```

DataNode remains THE core abstraction — the I/O sandbox that keeps everything safe
and portable. The streaming path bypasses DataNode for the `collect local` pipeline
only, while DataNode continues to serve all other use cases (programmatic access,
format conversion, inspection).

## Design Sections

### 1. DataNode Refactor

DataNode gains a `ToTarWriter(w io.Writer)` method for streaming out its contents
without buffering the entire archive. This is the bridge between DataNode's sandbox
model and streaming I/O.

New symlink handling:

| Symlink State | Behaviour |
|---------------|-----------|
| Valid, points inside DataNode root | Store as symlink entry |
| Valid, points outside DataNode root | Follow and store target content |
| Broken (dangling) | Skip with warning (configurable via `SkipBrokenSymlinks`) |

The `AddPath` method gets an options struct:

```go
type AddPathOptions struct {
    SkipBrokenSymlinks bool   // default: true
    FollowSymlinks     bool   // default: false (store as symlinks)
    ExcludePatterns    []string
}
```

### 2. UI & Logger Cleanup

Replace direct spinner writes with a `Progress` interface:

```go
type Progress interface {
    Start(label string)
    Update(current, total int64)
    Finish(label string)
    Log(level, msg string, args ...any)
}
```

Two implementations:
- **InteractiveProgress** — spinner + progress bar (when `isatty(stdout)`)
- **QuietProgress** — structured log lines only (cron, pipes, `--quiet` flag)

TTY detection at startup selects the implementation. All existing `ui.Spinner` and
`fmt.Printf` calls in library code get replaced with `Progress` method calls.

New `--quiet` / `-q` flag on all commands suppresses non-error output.

### 3. TIM Streaming Encryption

ChaCha20-Poly1305 is AEAD — it needs the full plaintext to compute the auth tag.
For streaming, we use a chunked block format:

```
[magic: 4 bytes "STIM"]
[version: 1 byte]
[salt: 16 bytes]           ← Argon2id salt
[argon2 params: 12 bytes]  ← time, memory, threads (uint32 LE each)

Per block (repeated):
  [nonce: 12 bytes]
  [length: 4 bytes LE]     ← ciphertext length including 16-byte Poly1305 tag
  [ciphertext: N bytes]    ← encrypted chunk + tag

Final block:
  [nonce: 12 bytes]
  [length: 4 bytes LE = 0] ← zero length signals EOF
```

Block size: 1 MiB plaintext → ~1 MiB + 16 bytes ciphertext per block.

The `Sigil` (Enchantrix crypto handle) wraps this as `StreamEncrypt(r io.Reader,
w io.Writer)` and `StreamDecrypt(r io.Reader, w io.Writer)`.

### 4. Key Derivation Hardening

Replace bare `SHA-256(password)` with Argon2id:

```go
key := argon2.IDKey(password, salt, time=3, memory=64*1024, threads=4, keyLen=32)
```

Parameters stored in the STIM header (section 3 above) so they can be tuned
without breaking existing archives. Random 16-byte salt generated per archive.

Backward compatibility: detect old format by checking for "STIM" magic. Old files
(no magic header) use legacy SHA-256 derivation with a deprecation warning.

### 5. Collect Local Streaming Pipeline

The new `collect local` pipeline for large directories:

```
filepath.WalkDir
    → tar.NewWriter (streaming)
        → xz/gzip compressor (streaming)
            → chunked AEAD encryptor (streaming)
                → os.File output
```

Memory usage: ~2 MiB regardless of input size (1 MiB compress buffer + 1 MiB
encrypt block).

Error handling:
- Broken symlinks: skip with warning (not fatal)
- Permission denied: skip with warning, continue
- Disk full on output: fatal, clean up partial file
- Read errors mid-stream: fatal, clean up partial file

Compression selection: `--compress=xz` (default, best ratio) or `--compress=gzip`
(faster). Matches existing Borg compression support.

### 6. Core Package Integration (Deferred)

Core imports Borg, so Borg cannot import core packages without creating a circular
dependency. Integration points are marked with TODOs for when the dependency
direction is resolved (likely by extracting shared interfaces to a common module):

- `core/go` config system → Borg config loading
- `core/go` logging → Borg Progress interface backend
- `core/go-store` → DataNode persistence
- `core/go` io.Medium → DataNode filesystem abstraction

## File Impact Summary

| Area | Files | Change Type |
|------|-------|-------------|
| DataNode | `pkg/datanode/*.go` | Modify (ToTarWriter, symlinks, AddPathOptions) |
| UI | `pkg/ui/*.go` | Rewrite (Progress interface, TTY detection) |
| TIM/STIM | `pkg/tim/*.go` | Modify (streaming encrypt/decrypt, new header) |
| Crypto | `pkg/tim/crypto.go` (new) | Create (Argon2id, chunked AEAD) |
| Collect | `cmd/collect_local.go` | Rewrite (streaming pipeline) |
| CLI | `cmd/root.go`, `cmd/*.go` | Modify (--quiet flag) |

## Testing Strategy

- Unit tests for each component (DataNode, Progress, chunked AEAD, Argon2id)
- Round-trip tests: encrypt → decrypt → compare original
- Large file test: 100 MiB synthetic directory through full pipeline
- Symlink matrix: valid internal, valid external, broken, nested
- Backward compatibility: decrypt old-format STIM with new code
- Race detector: `go test -race ./...`

## Dependencies

New:
- `golang.org/x/crypto/argon2` (Argon2id key derivation)
- `golang.org/x/term` (TTY detection via `term.IsTerminal`)

Existing (unchanged):
- `github.com/snider/Enchantrix` (ChaCha20-Poly1305 via Sigil)
- `github.com/ulikunitz/xz` (XZ compression)

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| Breaking existing STIM format | Magic-byte detection for backward compat |
| Chunked AEAD security | Standard construction (each block independent nonce) |
| Circular dep with core | Deferred; TODO markers only |
| Large directory edge cases | Extensive symlink + permission test matrix |
