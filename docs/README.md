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

**Example:**
```
./borg collect website https://google.com --output website.dat --depth 1
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

**Example:**
```
./borg collect pwa --uri https://squoosh.app --output squoosh.dat
```

### `serve`

Serves the contents of a packaged DataNode file using a static file server.

**Usage:**
```
borg serve [file] [flags]
```

**Flags:**
- `--port string`: Port to serve the DataNode on (default "8080")

**Example:**
```
./borg serve squoosh.dat --port 8888
```

## Inspecting a DataNode

The `examples` directory contains a Go program that can be used to inspect the contents of a `.dat` file.

**Usage:**
```
go run examples/inspect_datanode.go <path to .dat file>
```

**Example:**
```
# First, create a .dat file
./borg collect github repo https://github.com/Snider/Borg --output borg.dat

# Then, inspect it
go run examples/inspect_datanode.go borg.dat
```
