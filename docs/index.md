# Borg Data Collector

Borg Data Collector is a command-line tool for collecting and managing data from various sources.

## Commands

### `collect`

This command is used to collect resources from different sources and store them in a DataNode.

#### `collect github repo`

Collects a single Git repository and stores it in a DataNode.

**Usage:**
```
borg collect github repo [repository-url] [flags]
```

**Flags:**
- `--output string`: Output file for the DataNode (default "repo.dat")
- `--format string`: Output format (datanode or matrix) (default "datanode")
- `--compression string`: Compression format (none, gz, or xz) (default "none")

**Example:**
```
./borg collect github repo https://github.com/Snider/Borg --output borg.dat
```

#### `collect website`

Collects a single website and stores it in a DataNode.

**Usage:**
```
borg collect website [url] [flags]
```

**Flags:**
- `--output string`: Output file for the DataNode (default "website.dat")
- `--depth int`: Recursion depth for downloading (default 2)
- `--format string`: Output format (datanode or matrix) (default "datanode")
- `--compression string`: Compression format (none, gz, or xz) (default "none")

**Example:**
```
./borg collect website https://google.com --output website.dat --depth 1
```

#### `collect github repos`

Collects all public repositories for a user or organization.

**Usage:**
```
borg collect github repos [user-or-org] [flags]
```

**Example:**
```
./borg collect github repos Snider
```

#### `collect github release`

Downloads the latest release of a file from GitHub releases.

**Usage:**
```
borg collect github release [repository-url] [flags]
```

**Flags:**
- `--output string`: Output directory for the downloaded file (default ".")
- `--pack`: Pack all assets into a DataNode
- `--file string`: The file to download from the release
- `--version string`: The version to check against

**Example:**
```
# Download the latest release of the 'borg' executable
./borg collect github release https://github.com/Snider/Borg --file borg

# Pack all assets from the latest release into a DataNode
./borg collect github release https://github.com/Snider/Borg --pack --output borg-release.dat
```

#### `collect pwa`

Collects a single PWA and stores it in a DataNode.

**Usage:**
```
borg collect pwa [flags]
```

**Flags:**
- `--uri string`: The URI of the PWA to collect
- `--output string`: Output file for the DataNode (default "pwa.dat")
- `--format string`: Output format (datanode or matrix) (default "datanode")
- `--compression string`: Compression format (none, gz, or xz) (default "none")

**Example:**
```
./borg collect pwa --uri https://squoosh.app --output squoosh.dat
```

### `compile`

Compiles a `Borgfile` into a Terminal Isolation Matrix.

**Usage:**
```
borg compile [flags]
```

**Flags:**
- `--file string`: Path to the Borgfile (default "Borgfile")
- `--output string`: Path to the output matrix file (default "a.matrix")

**Example:**
```
./borg compile -f my-borgfile -o my-app.matrix
```

### `serve`

Serves the contents of a packaged DataNode or Terminal Isolation Matrix file using a static file server.

**Usage:**
```
borg serve [file] [flags]
```

**Flags:**
- `--port string`: Port to serve the DataNode on (default "8080")

**Example:**
```
# Serve a DataNode
./borg serve squoosh.dat --port 8888

# Serve a Terminal Isolation Matrix
./borg serve borg.matrix --port 9999
```

## Compression

All `collect` commands support optional compression. The following compression formats are available:

- `none`: No compression (default)
- `gz`: Gzip compression
- `xz`: XZ compression

To use compression, specify the desired format with the `--compression` flag. The output filename will be automatically updated with the appropriate extension (e.g., `.gz`, `.xz`).

**Example:**
```
./borg collect github repo https://github.com/Snider/Borg --compression gz
```

The `serve` command can transparently serve compressed files.

## Terminal Isolation Matrix

The `matrix` format creates a `runc` compatible bundle. This bundle can be executed by `runc` to create a container with the collected files. This is useful for creating isolated environments for testing or analysis.

To create a Matrix, use the `--format matrix` flag with any of the `collect` subcommands.

**Example:**
```
./borg collect github repo https://github.com/Snider/Borg --output borg.matrix --format matrix
```

The `borg run` command is used to execute a Terminal Isolation Matrix. This command handles the unpacking and execution of the matrix in a secure, isolated environment using `runc`. This ensures that the payload can be safely analyzed without affecting the host system.

**Example:**
```
./borg run borg.matrix
```

## Programmatic Usage

The `examples` directory contains a number of Go programs that demonstrate how to use the `borg` package programmatically.

### Inspecting a DataNode

The `inspect_datanode` example demonstrates how to read, decompress, and walk a `.dat` file.

**Usage:**
```
go run examples/inspect_datanode/main.go <path to .dat file>
```

### Creating a Matrix Programmatically

The `create_matrix_programmatically` example demonstrates how to create a Terminal Isolation Matrix from scratch.

**Usage:**
```
go run examples/create_matrix_programmatically/main.go
```

### Running a Matrix Programmatically

The `run_matrix_programmatically` example demonstrates how to run a Terminal Isolation Matrix using the `borg` package.

**Usage:**
```
go run examples/run_matrix_programmatically/main.go
```

### Collecting a Website

The `collect_website` example demonstrates how to collect a website and package it into a `.dat` file.

**Usage:**
```
go run examples/collect_website/main.go
```

### Collecting a GitHub Release

The `collect_github_release` example demonstrates how to collect the latest release of a GitHub repository.

**Usage:**
```
go run examples/collect_github_release/main.go
```

### Collecting All Repositories for a User

The `all` example demonstrates how to collect all public repositories for a GitHub user.

**Usage:**
```
go run examples/all/main.go
```

### Collecting a PWA

The `collect_pwa` example demonstrates how to collect a Progressive Web App and package it into a `.dat` file.

**Usage:**
```
go run examples/collect_pwa/main.go
```

### Collecting a GitHub Repository

The `collect_github_repo` example demonstrates how to clone a GitHub repository and package it into a `.dat` file.

**Usage:**
```
go run examples/collect_github_repo/main.go
```

### Serving a DataNode

The `serve` example demonstrates how to serve the contents of a `.dat` file over HTTP.

**Usage:**
```
go run examples/serve/main.go
```
