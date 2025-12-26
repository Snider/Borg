# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Development Commands

```bash
# Build
task build                              # or: go build -o borg main.go

# Test
task test                               # all tests with coverage
go test -run TestName ./pkg/tim         # single test
go test -v ./pkg/tim/...                # verbose package tests

# Clean and utilities
task clean                              # remove build artifacts
mkdocs serve                            # serve docs locally
```

## Architecture Overview

Borg collects data from various sources (GitHub, websites, PWAs) and packages it into portable, optionally encrypted containers.

### Core Abstractions

```
Source (GitHub/Website/PWA)
    ↓ collect
DataNode (in-memory fs.FS)
    ↓ serialize
    ├── .tar (raw tarball)
    ├── .tim (runc container bundle)
    ├── .trix (PGP encrypted)
    └── .stim (ChaCha20-Poly1305 encrypted TIM)
```

**DataNode** (`pkg/datanode/datanode.go`): In-memory filesystem implementing `fs.FS`. Core methods:
- `AddData(path, content)` - add file
- `ToTar()` / `FromTar()` - serialize/deserialize
- `Walk()`, `Open()`, `Stat()` - fs.FS interface

**TIM** (`pkg/tim/tim.go`): Terminal Isolation Matrix - runc-compatible container bundle with:
- `Config []byte` - OCI runtime spec (config.json)
- `RootFS *DataNode` - container filesystem
- `ToTar()` / `ToSigil(password)` - serialize plain or encrypted

### Encryption

Two encryption systems via Enchantrix library:

| Format | Algorithm | Use Case |
|--------|-----------|----------|
| `.trix` | PGP symmetric | Legacy DataNode encryption |
| `.stim` | ChaCha20-Poly1305 | TIM encryption (config + rootfs encrypted separately) |

**ChaChaPolySigil** (`pkg/tim/tim.go`):
```go
// Encrypt TIM
stim, _ := tim.ToSigil(password)

// Decrypt TIM
tim, _ := tim.FromSigil(data, password)

// Run encrypted TIM
tim.RunEncrypted(path, password)
```

**Key derivation**: `trix.DeriveKey(password)` - SHA-256(password) → 32-byte key

**Cache API** (`pkg/tim/cache.go`): Encrypted TIM storage
```go
cache, _ := tim.NewCache("/path/to/cache", password)
cache.Store("name", tim)
tim, _ := cache.Load("name")
```

### Package Structure

| Package | Purpose |
|---------|---------|
| `cmd/` | Cobra CLI commands |
| `pkg/datanode/` | In-memory fs.FS |
| `pkg/tim/` | Container bundles, encryption, execution |
| `pkg/trix/` | Trix format wrapper (PGP + ChaCha) |
| `pkg/compress/` | gzip/xz compression |
| `pkg/vcs/` | Git operations |
| `pkg/github/` | GitHub API client |
| `pkg/website/` | Website crawler |
| `pkg/pwa/` | PWA downloader |

### CLI Reference

```bash
# Collect
borg collect github repo <url>          # clone git repo
borg collect github repos <owner>       # clone all repos from user/org
borg collect website <url> --depth 2    # crawl website
borg collect pwa --uri <url>            # download PWA

# Common flags for collect commands:
#   --format datanode|tim|trix|stim
#   --compression none|gz|xz
#   --password <pass>                   # required for trix/stim

# Compile TIM from Borgfile
borg compile -f Borgfile -o out.tim
borg compile -f Borgfile -e "password"  # encrypted → .stim

# Run
borg run container.tim                  # plain TIM
borg run container.stim -p "password"   # encrypted TIM

# Decode
borg decode file.trix -o decoded.tar
borg decode file.stim -p "pass" --i-am-in-isolation -o decoded.tar

# Inspect (view metadata without decrypting)
borg inspect file.stim                  # human-readable
borg inspect file.stim --json           # JSON output
```

### Borgfile Format

```dockerfile
ADD local/path /container/path
```

### Testing Patterns

Tests use dependency injection for external services:
- `pkg/tim/run.go`: `ExecCommand` var for mocking runc
- `pkg/vcs/git.go`: `GitCloner` interface for mocking git
- `cmd/`: Commands expose `New*Cmd()` for testing

When adding encryption tests, use round-trip pattern:
```go
stim, _ := tim.ToSigil(password)
restored, _ := tim.FromSigil(stim, password)
// verify restored matches original
```
