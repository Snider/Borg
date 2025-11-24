# Releasing

This project is configured for GoReleaser.

## Prerequisites
- Create a GitHub personal access token with `repo` scope and export as `GITHUB_TOKEN` in your shell.
- Ensure a clean working tree and a tagged commit.
- Install goreleaser: https://goreleaser.com/install/

## Snapshot builds

Generate local artifacts without publishing:

- `goreleaser release --snapshot --clean`

Artifacts appear under `dist/`.

## Full release

1. Tag a new version:
   - `git tag -a v0.1.0 -m "v0.1.0"`
   - `git push origin v0.1.0`
2. Run GoReleaser:
   - `GITHUB_TOKEN=... goreleaser release --clean`

This will:
- Build binaries for multiple OS/ARCH
- Produce checksums and archives
- Create/update a GitHub Release with changelog
- Optionally publish a Homebrew formula (if repository exists and permissions allow)

## Notes
- The Go toolchain version is 1.25 (see go.mod and go.work).
