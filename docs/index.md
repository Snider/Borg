# Borg

Borg is a command-line tool for collecting resources from various URIs (like Git repositories and websites) into a unified format.

## Installation

You can install Borg using `go install`:

```bash
go install github.com/Snider/Borg@latest
```

## Usage

Borg provides several subcommands for collecting different types of resources.

### `borg collect`

The `collect` command is the main entry point for collecting resources. It has several subcommands for different resource types.

#### `borg collect github repo`

This command collects a single Git repository and stores it in a DataNode.

```bash
./borg collect github repo https://github.com/Snider/Borg --output borg.dat
```

#### `borg collect github release`

This command downloads and packages the assets from a GitHub release.

```bash
./borg collect github release https://github.com/Snider/Borg/releases/latest --output borg-release.dat
```

#### `borg collect pwa`

This command collects a Progressive Web App (PWA) from a given URI.

```bash
./borg collect pwa --uri https://squoosh.app --output squoosh.dat
```

#### `borg collect website`

This command collects a single website and stores it in a DataNode.

```bash
./borg collect website https://example.com --output example.dat
```

### `borg all`

The `borg all` command collects all public repositories from a GitHub user or organization.

```bash
./borg all https://github.com/Snider --output snider.dat
```

### `borg compile`

The `borg compile` command compiles a `Borgfile` into a Terminal Isolation Matrix.

```bash
./borg compile --file Borgfile --output a.tim
```

### `borg run`

The `borg run` command executes a Terminal Isolation Matrix.

```bash
./borg run a.tim
```

### `borg serve`

The `borg serve` command serves a DataNode or Terminal Isolation Matrix using a static file server.

```bash
./borg serve my-collected-data.dat --port 8080
```

### `borg decode`

The `borg decode` command decodes a `.trix` or `.tim` file.

```bash
./borg decode my-collected-data.trix --output my-collected-data.dat
```

## Formats

Borg supports three output formats: `datanode`, `tim`, and `trix`.

### DataNode

The `datanode` format is a simple tarball containing the collected resources. This is the default format.

### Terminal Isolation Matrix (TIM)

The Terminal Isolation Matrix (`tim`) is a `runc` bundle that can be executed in an isolated environment. This is useful for analyzing potentially malicious code without affecting the host system. A `.tim` file is a specialized `.trix` file with the `tim` flag set in its header.

To create a TIM, use the `--format tim` flag with any of the `collect` subcommands.

```bash
./borg collect github repo https://github.com/Snider/Borg --output borg.tim --format tim
```

### Trix

The `trix` format is an encrypted and structured file format. It is used as the underlying format for `.tim` files, but can also be used on its own for encrypting any `DataNode`.

To create a `.trix` file, use the `--format trix` flag with any of the `collect` subcommands.

```bash
./borg collect github repo https://github.com/Snider/Borg --output borg.trix --format trix --password "my-secret-password"
```

## Encryption

Both the `tim` and `trix` formats can be encrypted with a password by using the `--password` flag.

## Decoding

To decode a `.trix` or `.tim` file, use the `decode` command. If the file is encrypted, you must provide the `--password` flag.

```bash
./borg decode borg.trix --output borg.dat --password "my-secret-password"
```

If you are decoding a `.tim` file, you must also provide the `--i-am-in-isolation` flag. This is a safety measure to prevent you from accidentally executing potentially malicious code on your host system.

```bash
./borg decode borg.tim --output borg.dat --i-am-in-isolation
```
