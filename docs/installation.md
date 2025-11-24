# Installation

This project builds a single binary named `borg`.

Options to install:

- From source (requires Go 1.25 or newer):
  - Clone the repository and build:
    - `go build -o borg ./`
  - Or use the Taskfile:
    - `task build`

- From releases (recommended):
  - Download an archive for your OS/ARCH from GitHub Releases once you publish with GoReleaser.
  - Unpack and place `borg` on your PATH.

- Homebrew (if you publish to a tap):
  - `brew tap Snider/homebrew-tap`
  - `brew install borg`

Requirements:
- Go 1.25+ to build from source.
- macOS, Linux, Windows, or FreeBSD on amd64/arm64; Linux ARM v6/v7 supported.
