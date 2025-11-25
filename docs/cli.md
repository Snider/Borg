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
- `borg collect github repo <repo-url> [--output <file>] [--format datanode|tim|trix] [--compression none|gz|xz]`
- `borg collect github release <release-url> [--output <file>]`
- `borg collect github repos <org-or-user> [--output <file>] [--format ...] [--compression ...]`
- `borg collect website <url> [--depth N] [--output <file>] [--format ...] [--compression ...]`
- `borg collect pwa --uri <url> [--output <file>] [--format ...] [--compression ...]`

Examples:
- `borg collect github repo https://github.com/Snider/Borg --output borg.dat`
- `borg collect website https://example.com --depth 1 --output site.dat`
- `borg collect pwa --uri https://squoosh.app --output squoosh.dat`

### all

Collect all public repositories from a GitHub user or organization.

- `borg all <url> [--output <file>]`

Example:
- `borg all https://github.com/Snider --output snider.dat`

### compile

Compile a Borgfile into a Terminal Isolation Matrix (TIM).

- `borg compile [--file <Borgfile>] [--output <file>]`

Example:
- `borg compile --file Borgfile --output a.tim`

### run

Execute a Terminal Isolation Matrix (TIM).

- `borg run <tim-file>`

Example:
- `borg run a.tim`

### serve

Serve a packaged DataNode or TIM via a static file server.

- `borg serve <file> [--port 8080]`

Examples:
- `borg serve squoosh.dat --port 8888`
- `borg serve borg.tim --port 9999`

### decode

Decode a `.trix` or `.tim` file back into a DataNode (`.dat`).

- `borg decode <file> [--output <file>] [--password <password>]`

Examples:
- `borg decode borg.trix --output borg.dat --password "secret"`
- `borg decode borg.tim --output borg.dat --i-am-in-isolation`

## Compression

All collect commands accept `--compression` with values:
- `none` (default)
- `gz`
- `xz`

Output filenames gain the appropriate extension automatically.

## Formats

Borg supports three output formats via the `--format` flag:

- `datanode`: A simple tarball containing the collected resources. (Default)
- `tim`: Terminal Isolation Matrix, a runc-compatible bundle.
- `trix`: Encrypted and structured file format.
