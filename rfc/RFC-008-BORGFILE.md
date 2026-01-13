# RFC-008: Borgfile Compilation

**Status**: Draft
**Author**: [Snider](https://github.com/Snider/)
**Created**: 2026-01-13
**License**: EUPL-1.2
**Depends On**: RFC-003, RFC-004

---

## Abstract

Borgfile is a declarative syntax for defining TIM container contents. It specifies how local files are mapped into the container filesystem, enabling reproducible container builds.

## 1. Overview

Borgfile provides:
- Dockerfile-like syntax for familiarity
- File mapping into containers
- Simple ADD directive
- Integration with TIM encryption

## 2. File Format

### 2.1 Location

- Default: `Borgfile` in current directory
- Override: `borg compile -f path/to/Borgfile`

### 2.2 Encoding

- UTF-8 text
- Unix line endings (LF)
- No BOM

## 3. Syntax

### 3.1 Parsing Implementation

```go
// cmd/compile.go:33-54
lines := strings.Split(content, "\n")
for _, line := range lines {
    parts := strings.Fields(line)  // Whitespace-separated tokens
    if len(parts) == 0 {
        continue  // Skip empty lines
    }
    switch parts[0] {
    case "ADD":
        // Process ADD directive
    default:
        return fmt.Errorf("unknown instruction: %s", parts[0])
    }
}
```

### 3.2 ADD Directive

```
ADD <source> <destination>
```

| Parameter | Description |
|-----------|-------------|
| source | Local path (relative to current working directory) |
| destination | Container path (leading slash stripped) |

### 3.3 Examples

```dockerfile
# Add single file
ADD ./app /usr/local/bin/app

# Add configuration
ADD ./config.yaml /etc/myapp/config.yaml

# Multiple files
ADD ./bin/server /app/server
ADD ./static /app/static
```

## 4. Path Resolution

### 4.1 Source Paths

- Resolved relative to **current working directory** (not Borgfile location)
- Must exist at compile time
- Read via `os.ReadFile(src)`

### 4.2 Destination Paths

- Leading slash stripped: `strings.TrimPrefix(dest, "/")`
- Added to DataNode as-is

```go
// cmd/compile.go:46-50
data, err := os.ReadFile(src)
if err != nil {
    return fmt.Errorf("invalid ADD instruction: %s", line)
}
name := strings.TrimPrefix(dest, "/")
m.RootFS.AddData(name, data)
```

## 5. File Handling

### 5.1 Permissions

**Current implementation**: Permissions are NOT preserved.

| Source | Container |
|--------|-----------|
| Any file | 0600 (hardcoded in DataNode.ToTar) |
| Any directory | 0755 (implicit) |

### 5.2 Timestamps

- Set to `time.Now()` when added to DataNode
- Original timestamps not preserved

### 5.3 File Types

- Regular files only
- No directory recursion (each file must be added explicitly)
- No symlink following

## 6. Error Handling

| Error | Cause |
|-------|-------|
| `invalid ADD instruction: {line}` | Wrong number of arguments |
| `os.ReadFile` error | Source file not found |
| `unknown instruction: {name}` | Unrecognized directive |
| `ErrPasswordRequired` | Encryption requested without password |

## 7. CLI Flags

```go
// cmd/compile.go:80-82
-f, --file     string  Path to Borgfile (default: "Borgfile")
-o, --output   string  Output path (default: "a.tim")
-e, --encrypt  string  Password for .stim encryption (optional)
```

## 8. Output Formats

### 8.1 Plain TIM

```bash
borg compile -f Borgfile -o container.tim
```

Output: Standard TIM tar archive with `config.json` + `rootfs/`

### 8.2 Encrypted STIM

```bash
borg compile -f Borgfile -e "password" -o container.stim
```

Output: ChaCha20-Poly1305 encrypted STIM container

**Auto-detection**: If `-e` flag provided, output automatically uses `.stim` format even if `-o` specifies `.tim`.

## 9. Default OCI Config

The current implementation creates a minimal config:

```go
// pkg/tim/config.go:6-10
func defaultConfig() (*trix.Trix, error) {
    return &trix.Trix{Header: make(map[string]interface{})}, nil
}
```

**Note**: This is a placeholder. For full OCI runtime execution, you'll need to provide a proper `config.json` in the container or modify the TIM after compilation.

## 10. Compilation Process

```
1. Read Borgfile content
2. Parse line-by-line
3. For each ADD directive:
   a. Read source file from filesystem
   b. Strip leading slash from destination
   c. Add to DataNode
4. Create TIM with default config + populated RootFS
5. If password provided:
   a. Encrypt to STIM via ToSigil()
   b. Adjust output extension to .stim
6. Write output file
```

## 11. Implementation Reference

- Parser/Compiler: `cmd/compile.go`
- TIM creation: `pkg/tim/tim.go`
- DataNode: `pkg/datanode/datanode.go`
- Tests: `cmd/compile_test.go`

## 12. Current Limitations

| Feature | Status |
|---------|--------|
| Comment support (`#`) | Not implemented |
| Quoted paths | Not implemented |
| Directory recursion | Not implemented |
| Permission preservation | Not implemented |
| Path resolution relative to Borgfile | Not implemented (uses CWD) |
| Full OCI config generation | Not implemented (empty header) |
| Symlink following | Not implemented |

## 13. Examples

### 13.1 Simple Application

```dockerfile
ADD ./myapp /usr/local/bin/myapp
ADD ./config.yaml /etc/myapp/config.yaml
```

### 13.2 Web Application

```dockerfile
ADD ./server /app/server
ADD ./index.html /app/static/index.html
ADD ./style.css /app/static/style.css
ADD ./app.js /app/static/app.js
```

### 13.3 With Encryption

```bash
# Create Borgfile
cat > Borgfile << 'EOF'
ADD ./secret-app /app/secret-app
ADD ./credentials.json /etc/app/credentials.json
EOF

# Compile with encryption
borg compile -f Borgfile -e "MySecretPassword123" -o secret.stim
```

## 14. Future Work

- [ ] Comment support (`#`)
- [ ] Quoted path support for spaces
- [ ] Directory recursion in ADD
- [ ] Permission preservation
- [ ] Path resolution relative to Borgfile location
- [ ] Full OCI config generation
- [ ] Variable substitution (`${VAR}`)
- [ ] Include directive
- [ ] Glob patterns in source
- [ ] COPY directive (alias for ADD)
