# Borg

[![codecov](https://codecov.io/github/Snider/Borg/branch/main/graph/badge.svg?token=XWWU0SBIR4)](https://codecov.io/github/Snider/Borg)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Snider/Borg)](go.mod)
[![License](https://img.shields.io/badge/license-EUPL--1.2-blue)](LICENSE)

Borg is a CLI tool and Go library for collecting, packaging, and encrypting data into portable, self-contained containers. It supports GitHub repositories, websites, PWAs, and arbitrary files.

## Features

- **Data Collection** - Clone GitHub repos, crawl websites, download PWAs
- **Portable Containers** - Package data into DataNodes (in-memory fs.FS) or TIM bundles (OCI-compatible)
- **Zero-Trust Encryption** - ChaCha20-Poly1305 encryption for TIM containers (.stim) and messages (.smsg)
- **SMSG Format** - Encrypted message containers with public manifests, attachments, and zstd compression
- **WASM Support** - Decrypt SMSG files in the browser via WebAssembly

## Installation

```bash
# From source
go install github.com/Snider/Borg@latest

# Or build locally
git clone https://github.com/Snider/Borg.git
cd Borg
go build -o borg ./
```

Requires Go 1.25+

## Quick Start

```bash
# Clone a GitHub repository into a TIM container
borg collect github repo https://github.com/user/repo --format tim -o repo.tim

# Encrypt a TIM container
borg compile -f Borgfile -e "password" -o encrypted.stim

# Run an encrypted container
borg run encrypted.stim -p "password"

# Inspect container metadata (without decrypting)
borg inspect encrypted.stim --json
```

## Container Formats

| Format | Extension | Description |
|--------|-----------|-------------|
| DataNode | `.tar` | In-memory filesystem, portable tarball |
| TIM | `.tim` | Terminal Isolation Matrix - OCI/runc compatible bundle |
| Trix | `.trix` | PGP-encrypted DataNode |
| STIM | `.stim` | ChaCha20-Poly1305 encrypted TIM |
| SMSG | `.smsg` | Encrypted message with attachments and public manifest |

## SMSG - Secure Message Format

SMSG is designed for distributing encrypted content with publicly visible metadata:

```go
import "github.com/Snider/Borg/pkg/smsg"

// Create and encrypt a message
msg := smsg.NewMessage("Hello, World!")
msg.AddBinaryAttachment("track.mp3", audioData, "audio/mpeg")

manifest := &smsg.Manifest{
    Title:  "Demo Track",
    Artist: "Artist Name",
}

encrypted, _ := smsg.EncryptV2WithManifest(msg, "password", manifest)

// Decrypt
decrypted, _ := smsg.Decrypt(encrypted, "password")
```

**v2 Binary Format** - Stores attachments as raw binary with zstd compression for optimal size.

See [RFC-001: Open Source DRM](RFC-001-OSS-DRM.md) for the full specification.

**Live Demo**: [demo.dapp.fm](https://demo.dapp.fm)

## Borgfile

Package files into a TIM container:

```dockerfile
ADD ./app /usr/local/bin/app
ADD ./config /etc/app/
```

```bash
borg compile -f Borgfile -o app.tim
borg compile -f Borgfile -e "secret" -o app.stim  # encrypted
```

## CLI Reference

```bash
# Collection
borg collect github repo <url>           # Clone repository
borg collect github repos <owner>        # Clone all repos from user/org
borg collect website <url> --depth 2     # Crawl website
borg collect pwa --uri <url>             # Download PWA

# Compilation
borg compile -f Borgfile -o out.tim      # Plain TIM
borg compile -f Borgfile -e "pass"       # Encrypted STIM

# Execution
borg run container.tim                   # Run plain TIM
borg run container.stim -p "pass"        # Run encrypted TIM

# Inspection
borg decode file.stim -p "pass" -o out.tar
borg inspect file.stim [--json]
```

## Documentation

```bash
mkdocs serve  # Serve docs locally at http://localhost:8000
```

## Development

```bash
task build    # Build binary
task test     # Run tests with coverage
task clean    # Clean build artifacts
```

## Architecture

```
Source (GitHub/Website/PWA)
    ↓ collect
DataNode (in-memory fs.FS)
    ↓ serialize
    ├── .tar  (raw tarball)
    ├── .tim  (runc container bundle)
    ├── .trix (PGP encrypted)
    └── .stim (ChaCha20-Poly1305 encrypted TIM)
```

## License

[EUPL-1.2](LICENSE) - European Union Public License

---

<details>
<summary>Borg Status Messages (for CLI theming)</summary>

**Initialization**
- `Core engaged… resistance is already buffering.`
- `Assimilating bytes… stand by for cube‑formation.`
- `Merging… the Core is rewriting reality, one block at a time.`

**Encryption**
- `Generating cryptographic sigils – the Core whispers to the witch.`
- `Encrypting payload – the Core feeds data to the witch's cauldron.`
- `Merge complete – data assimilated, encrypted, and sealed within us.`

**VCS Processing**
- `Initiating clone… the Core replicates the collective into your node.`
- `Merging branches… conflicts resolved, entropy minimized, assimilation complete.`

</details>
