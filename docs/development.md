# Development

Prerequisites:
- Go 1.25 or newer
- Task (optional) — https://taskfile.dev
- MkDocs Material (optional for docs) — `pip install mkdocs-material`

## Workspace

This repo includes a `go.work` file configured for Go 1.25 to align with common workflows.

## Build

- `go build ./...`
- `task build`

## Test

- `go test ./...`
- `task test`

Note: Some tests may require network or git tooling depending on environment (e.g., pushing to a temporary repo).

## Run

- `task run`
- `./borg --help`

## Docs

Serve the documentation locally with MkDocs:

- `pip install mkdocs-material`
- `mkdocs serve`

The site configuration lives in `mkdocs.yml` and content in `docs/`.
