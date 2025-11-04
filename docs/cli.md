# CLI Usage

`borg` is a command-line tool for collecting repositories, websites, and PWAs into portable data artifacts (DataNodes) or Terminal Isolation Matrices.

Use `borg --help` and `borg <command> --help` to see all flags.

## Top-level

- `borg --help`
- `borg --version`

## Commands

### collect

Collect and package inputs.

Subcommands:
- `borg collect github repo <repo-url> [--output <file>] [--format datanode|matrix] [--compression none|gz|xz]`
- `borg collect github repos <org-or-user> [--output <file>] [--format ...] [--compression ...]` (if available)
- `borg collect website <url> [--depth N] [--output <file>] [--format ...] [--compression ...]`
- `borg collect pwa --uri <url> [--output <file>] [--format ...] [--compression ...]`

Examples:
- borg collect github repo https://github.com/Snider/Borg --output borg.dat
- borg collect website https://example.com --depth 1 --output site.dat
- borg collect pwa --uri https://squoosh.app --output squoosh.dat

### serve

Serve a packaged DataNode or Matrix via a static file server.

- borg serve <file> [--port 8080]

Examples:
- borg serve squoosh.dat --port 8888
- borg serve borg.matrix --port 9999

## Compression

All collect commands accept `--compression` with values:
- none (default)
- gz
- xz

Output filenames gain the appropriate extension automatically.

## Matrix format

Use `--format matrix` to produce a runc-compatible bundle (Terminal Isolation Matrix). See the Overview page for details.
