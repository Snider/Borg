# Borg Production Backup Upgrade — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make Borg's `collect local` work reliably on large directories with streaming I/O, proper symlink handling, quiet mode, and hardened encryption.

**Architecture:** Bottom-up refactor — build new primitives (Progress interface, streaming DataNode output, chunked AEAD, Argon2id), then rewire `collect local` to use them as a streaming pipeline: `walk → tar → compress → encrypt → file`. DataNode remains the core sandbox abstraction; streaming bypasses it only for the large-directory collection path.

**Tech Stack:** Go, Enchantrix (ChaCha20-Poly1305), Argon2id (`golang.org/x/crypto/argon2`), `golang.org/x/term` (TTY detection), `archive/tar`, `compress/gzip`, `github.com/ulikunitz/xz`

---

## Background for Implementers

### Repository Layout

```
/home/claude/Code/Borg/
├── cmd/                    # Cobra CLI commands
│   ├── root.go             # Root command + --verbose flag
│   ├── collect_local.go    # collect local command (MAIN TARGET)
│   └── ...                 # Other collect commands
├── pkg/
│   ├── datanode/           # In-memory fs.FS (THE core sandbox)
│   │   └── datanode.go     # DataNode struct, AddData, ToTar, FromTar
│   ├── tim/                # TIM container + STIM encryption
│   │   ├── tim.go          # TerminalIsolationMatrix, ToSigil, FromSigil
│   │   ├── run.go          # Container execution via runc
│   │   └── cache.go        # Encrypted TIM cache
│   ├── trix/               # Trix format wrapper
│   │   └── trix.go         # DeriveKey (SHA-256), ToTrix, FromTrix
│   ├── compress/           # gzip/xz compression
│   │   └── compress.go     # Compress(data, format), Decompress(data)
│   └── ui/                 # Progress bars + prompter
│       ├── progressbar.go  # NewProgressBar wrapper
│       ├── progress_writer.go  # io.Writer → progressbar adapter
│       └── non_interactive_prompter.go  # Quote-rotating prompter
├── go.mod
└── main.go
```

### Key Patterns

- **Test naming:** `TestXxx_Good` (happy path), `TestXxx_Bad` (expected errors), `TestXxx_Ugly` (edge cases)
- **Testing framework:** `testing` stdlib + `assert`/`require` from testify where already used, but most tests use plain `t.Fatal`/`t.Errorf`
- **DataNode files:** Stored as `map[string]*dataFile` with normalized paths (no leading `/`)
- **Encryption chain:** `password → DeriveKey (SHA-256) → 32-byte key → NewChaChaPolySigil → sigil.In(data)/sigil.Out(data)`
- **STIM format:** `trix.Encode(Trix{Header, Payload}, "STIM", nil)` where Payload = `[4-byte config size][encrypted config][encrypted rootfs]`

### Build & Test Commands

```bash
go build -o borg ./           # Build binary
go test ./...                 # All tests
go test -run TestName ./...   # Single test
go test -v ./pkg/datanode/    # Verbose, one package
go test -race ./...           # Race detector
go test -cover ./pkg/tim/     # Coverage for package
```

---

## Task 1: Progress Interface

Create a `Progress` interface that replaces direct spinner/progressbar writes. Two implementations: interactive (TTY) and quiet (cron/pipes).

**Files:**
- Create: `pkg/ui/progress.go`
- Create: `pkg/ui/progress_test.go`

**Step 1: Write the failing test**

Create `pkg/ui/progress_test.go`:

```go
package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestQuietProgress_Log_Good(t *testing.T) {
	var buf bytes.Buffer
	p := NewQuietProgress(&buf)
	p.Log("info", "test message", "key", "val")
	out := buf.String()
	if !strings.Contains(out, "test message") {
		t.Fatalf("expected log output to contain 'test message', got: %s", out)
	}
}

func TestQuietProgress_StartFinish_Good(t *testing.T) {
	var buf bytes.Buffer
	p := NewQuietProgress(&buf)
	p.Start("collecting")
	p.Update(50, 100)
	p.Finish("done")
	out := buf.String()
	if !strings.Contains(out, "collecting") {
		t.Fatalf("expected 'collecting' in output, got: %s", out)
	}
	if !strings.Contains(out, "done") {
		t.Fatalf("expected 'done' in output, got: %s", out)
	}
}

func TestQuietProgress_Update_Ugly(t *testing.T) {
	var buf bytes.Buffer
	p := NewQuietProgress(&buf)
	// Should not panic with zero total
	p.Update(0, 0)
	p.Update(5, 0)
}
```

**Step 2: Run test to verify it fails**

Run: `go test -run TestQuietProgress -v ./pkg/ui/`
Expected: FAIL — `NewQuietProgress` undefined

**Step 3: Write the Progress interface and QuietProgress implementation**

Create `pkg/ui/progress.go`:

```go
package ui

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// Progress abstracts output for both interactive and scripted use.
type Progress interface {
	Start(label string)
	Update(current, total int64)
	Finish(label string)
	Log(level, msg string, args ...any)
}

// QuietProgress writes structured log lines. For cron, pipes, --quiet.
type QuietProgress struct {
	w io.Writer
}

func NewQuietProgress(w io.Writer) *QuietProgress {
	return &QuietProgress{w: w}
}

func (q *QuietProgress) Start(label string) {
	fmt.Fprintf(q.w, "[START] %s\n", label)
}

func (q *QuietProgress) Update(current, total int64) {
	if total > 0 {
		fmt.Fprintf(q.w, "[PROGRESS] %d/%d\n", current, total)
	}
}

func (q *QuietProgress) Finish(label string) {
	fmt.Fprintf(q.w, "[DONE] %s\n", label)
}

func (q *QuietProgress) Log(level, msg string, args ...any) {
	fmt.Fprintf(q.w, "[%s] %s", level, msg)
	for i := 0; i+1 < len(args); i += 2 {
		fmt.Fprintf(q.w, " %v=%v", args[i], args[i+1])
	}
	fmt.Fprintln(q.w)
}

// InteractiveProgress uses a progress bar for TTY output.
type InteractiveProgress struct {
	bar *ProgressBarWrapper
	w   io.Writer
}

// ProgressBarWrapper wraps schollz/progressbar for the Progress interface.
type ProgressBarWrapper struct {
	total   int
	current int
	desc    string
}

func NewInteractiveProgress(w io.Writer) *InteractiveProgress {
	return &InteractiveProgress{w: w}
}

func (p *InteractiveProgress) Start(label string) {
	fmt.Fprintf(p.w, "→ %s\n", label)
}

func (p *InteractiveProgress) Update(current, total int64) {
	// Interactive progress bar updates handled by existing progressbar lib
	// when integrated — for now, simple percentage
	if total > 0 {
		pct := current * 100 / total
		fmt.Fprintf(p.w, "\r  %d%%", pct)
	}
}

func (p *InteractiveProgress) Finish(label string) {
	fmt.Fprintf(p.w, "\r✓ %s\n", label)
}

func (p *InteractiveProgress) Log(level, msg string, args ...any) {
	fmt.Fprintf(p.w, "  %s", msg)
	for i := 0; i+1 < len(args); i += 2 {
		fmt.Fprintf(p.w, " %v=%v", args[i], args[i+1])
	}
	fmt.Fprintln(p.w)
}

// IsTTY returns true if the given file descriptor is a terminal.
func IsTTY(fd int) bool {
	return term.IsTerminal(fd)
}

// DefaultProgress returns InteractiveProgress for TTYs, QuietProgress otherwise.
func DefaultProgress() Progress {
	if IsTTY(int(os.Stdout.Fd())) {
		return NewInteractiveProgress(os.Stdout)
	}
	return NewQuietProgress(os.Stdout)
}
```

**Step 4: Add `golang.org/x/term` dependency**

Run: `go get golang.org/x/term`

**Step 5: Run tests to verify they pass**

Run: `go test -run TestQuietProgress -v ./pkg/ui/`
Expected: PASS (all 3 tests)

**Step 6: Commit**

```bash
git add pkg/ui/progress.go pkg/ui/progress_test.go go.mod go.sum
git commit -m "feat(ui): add Progress interface with Quiet and Interactive implementations"
```

---

## Task 2: Wire --quiet Flag into CLI

Add `--quiet/-q` global flag and pass Progress through command context.

**Files:**
- Modify: `cmd/root.go`
- Create: `cmd/context.go`
- Create: `cmd/context_test.go`

**Step 1: Write the failing test**

Create `cmd/context_test.go`:

```go
package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestProgressFromCmd_Good(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.PersistentFlags().BoolP("quiet", "q", false, "")

	p := ProgressFromCmd(cmd)
	if p == nil {
		t.Fatal("expected non-nil Progress")
	}
}

func TestProgressFromCmd_Quiet_Good(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.PersistentFlags().BoolP("quiet", "q", true, "")
	_ = cmd.PersistentFlags().Set("quiet", "true")

	p := ProgressFromCmd(cmd)
	if p == nil {
		t.Fatal("expected non-nil Progress")
	}
	// QuietProgress should be returned when --quiet is set
}
```

**Step 2: Run test to verify it fails**

Run: `go test -run TestProgressFromCmd -v ./cmd/`
Expected: FAIL — `ProgressFromCmd` undefined

**Step 3: Implement context helper**

Create `cmd/context.go`:

```go
package cmd

import (
	"os"

	"github.com/Snider/Borg/pkg/ui"
	"github.com/spf13/cobra"
)

// ProgressFromCmd returns a Progress based on --quiet flag and TTY detection.
func ProgressFromCmd(cmd *cobra.Command) ui.Progress {
	quiet, _ := cmd.Flags().GetBool("quiet")
	if quiet {
		return ui.NewQuietProgress(os.Stderr)
	}
	return ui.DefaultProgress()
}
```

**Step 4: Add --quiet flag to root command**

In `cmd/root.go`, add after the `--verbose` flag line:

```go
rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress non-error output")
```

**Step 5: Run tests**

Run: `go test -run TestProgressFromCmd -v ./cmd/`
Expected: PASS

**Step 6: Commit**

```bash
git add cmd/context.go cmd/context_test.go cmd/root.go
git commit -m "feat(cli): add --quiet flag and ProgressFromCmd helper"
```

---

## Task 3: DataNode Symlink Support

Add symlink storage to DataNode — store symlink entries in the files map with a target path, and handle them in ToTar/FromTar.

**Files:**
- Modify: `pkg/datanode/datanode.go`
- Modify: `pkg/datanode/datanode_test.go`

**Step 1: Write the failing tests**

Add to `pkg/datanode/datanode_test.go`:

```go
func TestAddSymlink_Good(t *testing.T) {
	dn := New()
	dn.AddSymlink("link.txt", "target.txt")

	info, err := dn.Stat("link.txt")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("expected symlink mode")
	}
}

func TestSymlinkTarRoundTrip_Good(t *testing.T) {
	dn := New()
	dn.AddData("real.txt", []byte("content"))
	dn.AddSymlink("link.txt", "real.txt")

	tarData, err := dn.ToTar()
	if err != nil {
		t.Fatalf("ToTar failed: %v", err)
	}

	restored, err := FromTar(tarData)
	if err != nil {
		t.Fatalf("FromTar failed: %v", err)
	}

	info, err := restored.Stat("link.txt")
	if err != nil {
		t.Fatalf("Stat link failed: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("expected symlink mode after round-trip")
	}
}

func TestAddSymlink_Bad(t *testing.T) {
	dn := New()
	// Empty name should be ignored (same as AddData behaviour)
	dn.AddSymlink("", "target")

	_, err := dn.Stat("")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test -run "TestAddSymlink|TestSymlinkTarRoundTrip" -v ./pkg/datanode/`
Expected: FAIL — `AddSymlink` undefined

**Step 3: Add symlink support to DataNode**

In `pkg/datanode/datanode.go`, modify the `dataFile` struct to add a symlink target field, add `AddSymlink` method, update `ToTar` to write symlink entries, and update `FromTar` to read symlink entries:

1. Add `symlink string` field to `dataFile` struct (around line 13-18)
2. Add `isSymlink()` method on `dataFile` that returns `d.symlink != ""`
3. Add `AddSymlink(name, target string)` method on `DataNode`
4. In `ToTar()` (around line 58-82): check `f.isSymlink()` — if true, write tar header with `Typeflag: tar.TypeSymlink` and `Linkname: f.symlink`
5. In `FromTar()` (around line 32-55): handle `tar.TypeSymlink` case — call `dn.AddSymlink(header.Name, header.Linkname)`
6. In `Stat()`: return mode with `os.ModeSymlink` for symlink files
7. Update `dataFileInfo.Mode()` to check if the underlying file is a symlink

**Step 4: Run all datanode tests**

Run: `go test -v ./pkg/datanode/`
Expected: ALL PASS (new + existing)

**Step 5: Commit**

```bash
git add pkg/datanode/datanode.go pkg/datanode/datanode_test.go
git commit -m "feat(datanode): add symlink support with tar round-trip"
```

---

## Task 4: DataNode AddPath — Add Files from Filesystem

Add `AddPath` method that walks a real directory and adds files to DataNode, with configurable symlink and exclusion behaviour.

**Files:**
- Modify: `pkg/datanode/datanode.go`
- Create: `pkg/datanode/addpath_test.go`

**Step 1: Write the failing tests**

Create `pkg/datanode/addpath_test.go`:

```go
package datanode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddPath_Good(t *testing.T) {
	// Create temp directory with files
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello"), 0644)
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "sub", "nested.txt"), []byte("nested"), 0644)

	dn := New()
	err := dn.AddPath(dir, AddPathOptions{})
	if err != nil {
		t.Fatalf("AddPath failed: %v", err)
	}

	// Check files exist
	if _, err := dn.Stat("hello.txt"); err != nil {
		t.Fatalf("hello.txt not found: %v", err)
	}
	if _, err := dn.Stat("sub/nested.txt"); err != nil {
		t.Fatalf("sub/nested.txt not found: %v", err)
	}
}

func TestAddPath_SkipBrokenSymlinks_Good(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "real.txt"), []byte("real"), 0644)
	os.Symlink("/nonexistent/target", filepath.Join(dir, "broken"))

	dn := New()
	err := dn.AddPath(dir, AddPathOptions{SkipBrokenSymlinks: true})
	if err != nil {
		t.Fatalf("AddPath failed: %v", err)
	}

	if _, err := dn.Stat("real.txt"); err != nil {
		t.Fatalf("real.txt not found: %v", err)
	}
	// Broken symlink should be skipped
	if _, err := dn.Stat("broken"); err == nil {
		t.Fatal("expected broken symlink to be skipped")
	}
}

func TestAddPath_ExcludePatterns_Good(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("keep"), 0644)
	os.WriteFile(filepath.Join(dir, "skip.log"), []byte("skip"), 0644)

	dn := New()
	err := dn.AddPath(dir, AddPathOptions{
		ExcludePatterns: []string{"*.log"},
	})
	if err != nil {
		t.Fatalf("AddPath failed: %v", err)
	}

	if _, err := dn.Stat("keep.txt"); err != nil {
		t.Fatalf("keep.txt not found: %v", err)
	}
	if _, err := dn.Stat("skip.log"); err == nil {
		t.Fatal("expected *.log to be excluded")
	}
}

func TestAddPath_Bad(t *testing.T) {
	dn := New()
	err := dn.AddPath("/nonexistent/dir", AddPathOptions{})
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestAddPath_ValidSymlink_Good(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "real.txt"), []byte("real"), 0644)
	os.Symlink(filepath.Join(dir, "real.txt"), filepath.Join(dir, "valid_link"))

	dn := New()
	err := dn.AddPath(dir, AddPathOptions{SkipBrokenSymlinks: true})
	if err != nil {
		t.Fatalf("AddPath failed: %v", err)
	}

	// Valid symlink inside root should be stored as symlink
	if _, err := dn.Stat("valid_link"); err != nil {
		t.Fatalf("valid_link not found: %v", err)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test -run TestAddPath -v ./pkg/datanode/`
Expected: FAIL — `AddPathOptions` and `AddPath` undefined

**Step 3: Implement AddPath**

Add to `pkg/datanode/datanode.go`:

```go
// AddPathOptions configures AddPath behaviour.
type AddPathOptions struct {
	SkipBrokenSymlinks bool     // Skip broken symlinks instead of erroring (default: false)
	FollowSymlinks     bool     // Follow symlinks and store target content (default: false = store as symlinks)
	ExcludePatterns    []string // Glob patterns to exclude
}

// AddPath walks a filesystem directory and adds all files to the DataNode.
// Paths are stored relative to dir.
func (dn *DataNode) AddPath(dir string, opts AddPathOptions) error {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)

		// Check exclude patterns
		for _, pat := range opts.ExcludePatterns {
			if matched, _ := filepath.Match(pat, filepath.Base(path)); matched {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Handle symlinks
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				if opts.SkipBrokenSymlinks {
					return nil
				}
				return err
			}
			// Check if target exists
			absTarget := target
			if !filepath.IsAbs(absTarget) {
				absTarget = filepath.Join(filepath.Dir(path), target)
			}
			if _, err := os.Stat(absTarget); err != nil {
				if opts.SkipBrokenSymlinks {
					return nil
				}
				return fmt.Errorf("broken symlink %s -> %s: %w", rel, target, err)
			}
			if opts.FollowSymlinks {
				content, err := os.ReadFile(absTarget)
				if err != nil {
					return err
				}
				dn.AddData(rel, content)
			} else {
				dn.AddSymlink(rel, target)
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		dn.AddData(rel, content)
		return nil
	})
}
```

**Important:** `filepath.WalkDir` does NOT follow symlinks by default and reports them as `d.Type()&fs.ModeSymlink != 0`. However, `d.Info()` may fail for broken symlinks. The implementation needs to use `os.Lstat` to detect symlinks since `WalkDir` may not report the symlink type. Check the actual behaviour and adjust — you may need to call `os.Lstat(path)` directly instead of relying on `d.Info()`.

**Step 4: Run tests**

Run: `go test -run TestAddPath -v ./pkg/datanode/`
Expected: ALL PASS

**Step 5: Run full datanode test suite**

Run: `go test -v ./pkg/datanode/`
Expected: ALL PASS

**Step 6: Commit**

```bash
git add pkg/datanode/datanode.go pkg/datanode/addpath_test.go
git commit -m "feat(datanode): add AddPath for filesystem directory collection"
```

---

## Task 5: DataNode ToTarWriter — Streaming Tar Output

Add `ToTarWriter(w io.Writer) error` method so DataNode contents can be streamed to a tar writer without buffering the entire archive in memory.

**Files:**
- Modify: `pkg/datanode/datanode.go`
- Modify: `pkg/datanode/datanode_test.go`

**Step 1: Write the failing test**

Add to `pkg/datanode/datanode_test.go`:

```go
func TestToTarWriter_Good(t *testing.T) {
	dn := New()
	dn.AddData("a.txt", []byte("aaa"))
	dn.AddData("b.txt", []byte("bbb"))

	var buf bytes.Buffer
	err := dn.ToTarWriter(&buf)
	if err != nil {
		t.Fatalf("ToTarWriter failed: %v", err)
	}

	// Should be valid tar — parse it back
	restored, err := FromTar(buf.Bytes())
	if err != nil {
		t.Fatalf("FromTar failed: %v", err)
	}

	f, err := restored.Open("a.txt")
	if err != nil {
		t.Fatalf("a.txt not found: %v", err)
	}
	content, _ := io.ReadAll(f)
	if string(content) != "aaa" {
		t.Fatalf("expected 'aaa', got '%s'", content)
	}
}

func TestToTarWriter_Symlinks_Good(t *testing.T) {
	dn := New()
	dn.AddData("real.txt", []byte("content"))
	dn.AddSymlink("link.txt", "real.txt")

	var buf bytes.Buffer
	err := dn.ToTarWriter(&buf)
	if err != nil {
		t.Fatalf("ToTarWriter failed: %v", err)
	}

	restored, err := FromTar(buf.Bytes())
	if err != nil {
		t.Fatalf("FromTar failed: %v", err)
	}

	info, err := restored.Stat("link.txt")
	if err != nil {
		t.Fatalf("link.txt not found: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("expected symlink after round-trip through ToTarWriter")
	}
}

func TestToTarWriter_Empty_Good(t *testing.T) {
	dn := New()
	var buf bytes.Buffer
	err := dn.ToTarWriter(&buf)
	if err != nil {
		t.Fatalf("ToTarWriter failed on empty DataNode: %v", err)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test -run TestToTarWriter -v ./pkg/datanode/`
Expected: FAIL — `ToTarWriter` undefined

**Step 3: Implement ToTarWriter**

Add to `pkg/datanode/datanode.go`:

```go
// ToTarWriter writes the DataNode contents as a tar stream to w.
// Unlike ToTar(), this does not buffer the entire archive in memory.
func (dn *DataNode) ToTarWriter(w io.Writer) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	// Sort keys for deterministic output
	names := make([]string, 0, len(dn.files))
	for name := range dn.files {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		f := dn.files[name]
		if f.isSymlink() {
			hdr := &tar.Header{
				Name:     name,
				Typeflag: tar.TypeSymlink,
				Linkname: f.symlink,
				ModTime:  f.modTime,
				Mode:     0777,
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
		} else {
			hdr := &tar.Header{
				Name:     name,
				Size:     int64(len(f.content)),
				Mode:     0600,
				ModTime:  f.modTime,
				Typeflag: tar.TypeReg,
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			if _, err := tw.Write(f.content); err != nil {
				return err
			}
		}
	}
	return nil
}
```

Also add `import "sort"` if not already present.

**Step 4: Run tests**

Run: `go test -run TestToTarWriter -v ./pkg/datanode/`
Expected: ALL PASS

**Step 5: Run full test suite**

Run: `go test -v ./pkg/datanode/`
Expected: ALL PASS

**Step 6: Commit**

```bash
git add pkg/datanode/datanode.go pkg/datanode/datanode_test.go
git commit -m "feat(datanode): add ToTarWriter for streaming tar output"
```

---

## Task 6: Argon2id Key Derivation

Replace bare SHA-256 key derivation with Argon2id. Keep backward compatibility by detecting old format.

**Files:**
- Modify: `pkg/trix/trix.go`
- Create: `pkg/trix/trix_test.go` (if not exists, or modify existing)

**Step 1: Write the failing tests**

Create/add to `pkg/trix/trix_test.go`:

```go
package trix

import (
	"bytes"
	"testing"
)

func TestDeriveKeyArgon2_Good(t *testing.T) {
	salt := make([]byte, 16)
	key := DeriveKeyArgon2("password", salt)
	if len(key) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(key))
	}
}

func TestDeriveKeyArgon2_Deterministic_Good(t *testing.T) {
	salt := []byte("0123456789abcdef")
	key1 := DeriveKeyArgon2("password", salt)
	key2 := DeriveKeyArgon2("password", salt)
	if !bytes.Equal(key1, key2) {
		t.Fatal("same password+salt should produce same key")
	}
}

func TestDeriveKeyArgon2_DifferentSalt_Good(t *testing.T) {
	salt1 := []byte("0123456789abcdef")
	salt2 := []byte("fedcba9876543210")
	key1 := DeriveKeyArgon2("password", salt1)
	key2 := DeriveKeyArgon2("password", salt2)
	if bytes.Equal(key1, key2) {
		t.Fatal("different salts should produce different keys")
	}
}

func TestDeriveKeyLegacy_Good(t *testing.T) {
	// Existing DeriveKey should still work (backward compat)
	key := DeriveKey("password")
	if len(key) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(key))
	}
}

func TestArgon2Params_Good(t *testing.T) {
	p := DefaultArgon2Params()
	if p.Time == 0 || p.Memory == 0 || p.Threads == 0 {
		t.Fatal("params should have non-zero values")
	}
	encoded := p.Encode()
	if len(encoded) != 12 {
		t.Fatalf("expected 12-byte encoded params, got %d", len(encoded))
	}
	decoded := DecodeArgon2Params(encoded)
	if decoded.Time != p.Time || decoded.Memory != p.Memory || decoded.Threads != p.Threads {
		t.Fatal("round-trip params mismatch")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test -run "TestDeriveKeyArgon2|TestArgon2Params" -v ./pkg/trix/`
Expected: FAIL — `DeriveKeyArgon2`, `DefaultArgon2Params`, etc. undefined

**Step 3: Add Argon2id dependency**

Run: `go get golang.org/x/crypto/argon2`

**Step 4: Implement Argon2id key derivation**

Add to `pkg/trix/trix.go` (keep existing `DeriveKey` for backward compat):

```go
import (
	"encoding/binary"

	"golang.org/x/crypto/argon2"
)

// Argon2Params holds Argon2id parameters stored in STIM headers.
type Argon2Params struct {
	Time    uint32
	Memory  uint32 // in KiB
	Threads uint32
}

// DefaultArgon2Params returns recommended Argon2id parameters.
func DefaultArgon2Params() Argon2Params {
	return Argon2Params{
		Time:    3,
		Memory:  64 * 1024, // 64 MiB
		Threads: 4,
	}
}

// Encode serialises params as 12 bytes (3 x uint32 LE).
func (p Argon2Params) Encode() []byte {
	buf := make([]byte, 12)
	binary.LittleEndian.PutUint32(buf[0:4], p.Time)
	binary.LittleEndian.PutUint32(buf[4:8], p.Memory)
	binary.LittleEndian.PutUint32(buf[8:12], p.Threads)
	return buf
}

// DecodeArgon2Params reads 12 bytes into Argon2Params.
func DecodeArgon2Params(data []byte) Argon2Params {
	return Argon2Params{
		Time:    binary.LittleEndian.Uint32(data[0:4]),
		Memory:  binary.LittleEndian.Uint32(data[4:8]),
		Threads: binary.LittleEndian.Uint32(data[8:12]),
	}
}

// DeriveKeyArgon2 derives a 32-byte key using Argon2id.
func DeriveKeyArgon2(password string, salt []byte) []byte {
	p := DefaultArgon2Params()
	return argon2.IDKey([]byte(password), salt, p.Time, p.Memory, uint8(p.Threads), 32)
}
```

**Step 5: Run tests**

Run: `go test -v ./pkg/trix/`
Expected: ALL PASS

**Step 6: Commit**

```bash
git add pkg/trix/trix.go pkg/trix/trix_test.go go.mod go.sum
git commit -m "feat(trix): add Argon2id key derivation alongside legacy SHA-256"
```

---

## Task 7: Chunked AEAD Streaming Encryption

Implement streaming encrypt/decrypt using 1 MiB blocks, each with its own ChaCha20-Poly1305 nonce and tag. New STIM v2 format with magic header, salt, and Argon2 params.

**Files:**
- Create: `pkg/tim/stream.go`
- Create: `pkg/tim/stream_test.go`

**Step 1: Write the failing tests**

Create `pkg/tim/stream_test.go`:

```go
package tim

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"
)

func TestStreamRoundTrip_Good(t *testing.T) {
	plaintext := []byte("hello, streaming encryption!")
	password := "test-password-123"

	var encrypted bytes.Buffer
	err := StreamEncrypt(bytes.NewReader(plaintext), &encrypted, password)
	if err != nil {
		t.Fatalf("StreamEncrypt failed: %v", err)
	}

	var decrypted bytes.Buffer
	err = StreamDecrypt(bytes.NewReader(encrypted.Bytes()), &decrypted, password)
	if err != nil {
		t.Fatalf("StreamDecrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted.Bytes(), plaintext) {
		t.Fatalf("round-trip mismatch: got %q, want %q", decrypted.Bytes(), plaintext)
	}
}

func TestStreamRoundTrip_Large_Good(t *testing.T) {
	// 3 MiB — spans multiple 1 MiB blocks
	plaintext := make([]byte, 3*1024*1024)
	rand.Read(plaintext)
	password := "large-test"

	var encrypted bytes.Buffer
	err := StreamEncrypt(bytes.NewReader(plaintext), &encrypted, password)
	if err != nil {
		t.Fatalf("StreamEncrypt failed: %v", err)
	}

	var decrypted bytes.Buffer
	err = StreamDecrypt(bytes.NewReader(encrypted.Bytes()), &decrypted, password)
	if err != nil {
		t.Fatalf("StreamDecrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted.Bytes(), plaintext) {
		t.Fatal("large round-trip mismatch")
	}
}

func TestStreamEncrypt_Empty_Good(t *testing.T) {
	var encrypted bytes.Buffer
	err := StreamEncrypt(bytes.NewReader(nil), &encrypted, "pass")
	if err != nil {
		t.Fatalf("StreamEncrypt failed on empty input: %v", err)
	}

	var decrypted bytes.Buffer
	err = StreamDecrypt(bytes.NewReader(encrypted.Bytes()), &decrypted, "pass")
	if err != nil {
		t.Fatalf("StreamDecrypt failed on empty: %v", err)
	}
	if decrypted.Len() != 0 {
		t.Fatalf("expected empty output, got %d bytes", decrypted.Len())
	}
}

func TestStreamDecrypt_WrongPassword_Bad(t *testing.T) {
	plaintext := []byte("secret data")
	var encrypted bytes.Buffer
	StreamEncrypt(bytes.NewReader(plaintext), &encrypted, "correct")

	var decrypted bytes.Buffer
	err := StreamDecrypt(bytes.NewReader(encrypted.Bytes()), &decrypted, "wrong")
	if err == nil {
		t.Fatal("expected error with wrong password")
	}
}

func TestStreamDecrypt_Truncated_Bad(t *testing.T) {
	plaintext := []byte("some data here")
	var encrypted bytes.Buffer
	StreamEncrypt(bytes.NewReader(plaintext), &encrypted, "pass")

	// Truncate the encrypted data
	truncated := encrypted.Bytes()[:encrypted.Len()/2]
	var decrypted bytes.Buffer
	err := StreamDecrypt(bytes.NewReader(truncated), &decrypted, "pass")
	if err == nil || err == io.EOF {
		// Should be an actual error, not silent success
		t.Fatal("expected error on truncated data")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test -run TestStream -v ./pkg/tim/`
Expected: FAIL — `StreamEncrypt`, `StreamDecrypt` undefined

**Step 3: Implement streaming encryption**

Create `pkg/tim/stream.go`:

```go
package tim

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	borgtrix "github.com/Snider/Borg/pkg/trix"
	"golang.org/x/crypto/chacha20poly1305"
)

// STIM v2 streaming format constants.
const (
	stimMagic      = "STIM"
	stimVersion    = 2
	blockSize      = 1024 * 1024 // 1 MiB plaintext per block
	saltSize       = 16
	argon2ParamLen = 12
	nonceSize      = chacha20poly1305.NonceSize // 12 bytes
	lengthSize     = 4                          // uint32 LE for ciphertext length
	tagSize        = chacha20poly1305.Overhead  // 16 bytes (Poly1305 tag)
)

// headerSize is magic(4) + version(1) + salt(16) + argon2params(12) = 33 bytes.
const headerSize = 4 + 1 + saltSize + argon2ParamLen

// StreamEncrypt reads plaintext from r and writes chunked-AEAD encrypted data to w.
//
// Format:
//
//	[magic: 4 bytes "STIM"]
//	[version: 1 byte = 2]
//	[salt: 16 bytes]
//	[argon2 params: 12 bytes]
//	Per block:
//	  [nonce: 12 bytes]
//	  [length: 4 bytes LE] — ciphertext length (plaintext + 16-byte tag)
//	  [ciphertext: length bytes]
//	EOF block:
//	  [nonce: 12 bytes]
//	  [length: 4 bytes LE = 0]
func StreamEncrypt(r io.Reader, w io.Writer, password string) error {
	// Generate salt
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("generating salt: %w", err)
	}

	// Derive key
	params := borgtrix.DefaultArgon2Params()
	key := borgtrix.DeriveKeyArgon2(password, salt)

	// Create AEAD cipher
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return fmt.Errorf("creating cipher: %w", err)
	}

	// Write header
	header := make([]byte, headerSize)
	copy(header[0:4], stimMagic)
	header[4] = stimVersion
	copy(header[5:21], salt)
	copy(header[21:33], params.Encode())
	if _, err := w.Write(header); err != nil {
		return fmt.Errorf("writing header: %w", err)
	}

	// Encrypt blocks
	plainBuf := make([]byte, blockSize)
	nonce := make([]byte, nonceSize)
	lenBuf := make([]byte, lengthSize)

	for {
		n, readErr := io.ReadFull(r, plainBuf)
		if n > 0 {
			// Generate random nonce
			if _, err := rand.Read(nonce); err != nil {
				return fmt.Errorf("generating nonce: %w", err)
			}

			// Encrypt block
			ciphertext := aead.Seal(nil, nonce, plainBuf[:n], nil)

			// Write nonce + length + ciphertext
			if _, err := w.Write(nonce); err != nil {
				return err
			}
			binary.LittleEndian.PutUint32(lenBuf, uint32(len(ciphertext)))
			if _, err := w.Write(lenBuf); err != nil {
				return err
			}
			if _, err := w.Write(ciphertext); err != nil {
				return err
			}
		}

		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("reading plaintext: %w", readErr)
		}
	}

	// Write EOF marker: nonce + length=0
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	if _, err := w.Write(nonce); err != nil {
		return err
	}
	binary.LittleEndian.PutUint32(lenBuf, 0)
	_, err = w.Write(lenBuf)
	return err
}

// StreamDecrypt reads chunked-AEAD encrypted data from r and writes plaintext to w.
func StreamDecrypt(r io.Reader, w io.Writer, password string) error {
	// Read and validate header
	header := make([]byte, headerSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return fmt.Errorf("reading header: %w", err)
	}

	if string(header[0:4]) != stimMagic {
		return errors.New("not a STIM v2 file (bad magic)")
	}
	if header[4] != stimVersion {
		return fmt.Errorf("unsupported STIM version: %d", header[4])
	}

	salt := header[5:21]
	params := borgtrix.DecodeArgon2Params(header[21:33])

	// Derive key with stored params
	key := deriveKeyWithParams(password, salt, params)

	// Create AEAD cipher
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return fmt.Errorf("creating cipher: %w", err)
	}

	// Decrypt blocks
	nonce := make([]byte, nonceSize)
	lenBuf := make([]byte, lengthSize)

	for {
		// Read nonce
		if _, err := io.ReadFull(r, nonce); err != nil {
			return fmt.Errorf("reading block nonce: %w", err)
		}

		// Read ciphertext length
		if _, err := io.ReadFull(r, lenBuf); err != nil {
			return fmt.Errorf("reading block length: %w", err)
		}
		ctLen := binary.LittleEndian.Uint32(lenBuf)

		// EOF marker
		if ctLen == 0 {
			return nil
		}

		// Read ciphertext
		ciphertext := make([]byte, ctLen)
		if _, err := io.ReadFull(r, ciphertext); err != nil {
			return fmt.Errorf("reading block ciphertext: %w", err)
		}

		// Decrypt
		plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return fmt.Errorf("decrypting block: %w", err)
		}

		if _, err := w.Write(plaintext); err != nil {
			return err
		}
	}
}

// deriveKeyWithParams uses specific Argon2id params (from STIM header).
func deriveKeyWithParams(password string, salt []byte, params borgtrix.Argon2Params) []byte {
	return argon2idKey([]byte(password), salt, params.Time, params.Memory, uint8(params.Threads), 32)
}
```

**Note:** You'll need to import `golang.org/x/crypto/argon2` and create the `argon2idKey` wrapper, or call `argon2.IDKey` directly. The simplest approach:

```go
import "golang.org/x/crypto/argon2"

func argon2idKey(password, salt []byte, time, memory uint32, threads uint8, keyLen uint32) []byte {
	return argon2.IDKey(password, salt, time, memory, threads, keyLen)
}
```

**Step 4: Run tests**

Run: `go test -run TestStream -v ./pkg/tim/`
Expected: ALL PASS

**Step 5: Run full tim test suite**

Run: `go test -v ./pkg/tim/`
Expected: ALL PASS (new + existing)

**Step 6: Commit**

```bash
git add pkg/tim/stream.go pkg/tim/stream_test.go
git commit -m "feat(tim): add chunked AEAD streaming encryption (STIM v2)"
```

---

## Task 8: Streaming Compression Wrappers

Add streaming compress/decompress that work with `io.Writer`/`io.Reader` instead of `[]byte`.

**Files:**
- Modify: `pkg/compress/compress.go`
- Modify: `pkg/compress/compress_test.go`

**Step 1: Write the failing tests**

Add to `pkg/compress/compress_test.go`:

```go
func TestNewCompressWriter_Gzip_Good(t *testing.T) {
	var buf bytes.Buffer
	wc, err := NewCompressWriter(&buf, "gz")
	if err != nil {
		t.Fatalf("NewCompressWriter failed: %v", err)
	}
	wc.Write([]byte("hello gzip stream"))
	wc.Close()

	// Decompress and verify
	result, err := Decompress(buf.Bytes())
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	if string(result) != "hello gzip stream" {
		t.Fatalf("expected 'hello gzip stream', got '%s'", result)
	}
}

func TestNewCompressWriter_Xz_Good(t *testing.T) {
	var buf bytes.Buffer
	wc, err := NewCompressWriter(&buf, "xz")
	if err != nil {
		t.Fatalf("NewCompressWriter failed: %v", err)
	}
	wc.Write([]byte("hello xz stream"))
	wc.Close()

	result, err := Decompress(buf.Bytes())
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	if string(result) != "hello xz stream" {
		t.Fatalf("expected 'hello xz stream', got '%s'", result)
	}
}

func TestNewCompressWriter_None_Good(t *testing.T) {
	var buf bytes.Buffer
	wc, err := NewCompressWriter(&buf, "none")
	if err != nil {
		t.Fatalf("NewCompressWriter failed: %v", err)
	}
	wc.Write([]byte("passthrough"))
	wc.Close()

	if buf.String() != "passthrough" {
		t.Fatalf("expected passthrough, got '%s'", buf.String())
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test -run TestNewCompressWriter -v ./pkg/compress/`
Expected: FAIL — `NewCompressWriter` undefined

**Step 3: Implement streaming compress writer**

Add to `pkg/compress/compress.go`:

```go
// NewCompressWriter returns a WriteCloser that compresses to w.
// Caller MUST call Close() to flush and finalise the stream.
// format: "gz", "xz", or "none" (passthrough).
func NewCompressWriter(w io.Writer, format string) (io.WriteCloser, error) {
	switch format {
	case "gz":
		return gzip.NewWriter(w), nil
	case "xz":
		xw, err := xz.NewWriter(w)
		if err != nil {
			return nil, err
		}
		return xw, nil
	case "none", "":
		return &nopCloser{w}, nil
	default:
		return nil, fmt.Errorf("unsupported compression format: %s", format)
	}
}

// nopCloser wraps an io.Writer with a no-op Close.
type nopCloser struct {
	io.Writer
}

func (n *nopCloser) Close() error { return nil }
```

Add `import "fmt"` if not already present.

**Step 4: Run tests**

Run: `go test -run TestNewCompressWriter -v ./pkg/compress/`
Expected: ALL PASS

**Step 5: Run full compress test suite**

Run: `go test -v ./pkg/compress/`
Expected: ALL PASS

**Step 6: Commit**

```bash
git add pkg/compress/compress.go pkg/compress/compress_test.go
git commit -m "feat(compress): add NewCompressWriter for streaming compression"
```

---

## Task 9: Rewrite Collect Local — Streaming Pipeline

Rewrite `collect local` to use the streaming pipeline: `walk → tar → compress → encrypt → file`. Falls back to DataNode for non-stim formats (datanode, tim, trix).

**Files:**
- Modify: `cmd/collect_local.go`
- Create: `cmd/collect_local_test.go`

**Step 1: Write the failing test**

Create `cmd/collect_local_test.go`:

```go
package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Snider/Borg/pkg/tim"
)

func TestCollectLocalStreaming_Good(t *testing.T) {
	// Create temp source directory
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("aaa"), 0644)
	os.MkdirAll(filepath.Join(srcDir, "sub"), 0755)
	os.WriteFile(filepath.Join(srcDir, "sub", "b.txt"), []byte("bbb"), 0644)

	// Create temp output file
	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "test.stim")

	err := CollectLocalStreaming(srcDir, outFile, "xz", "test-password")
	if err != nil {
		t.Fatalf("CollectLocalStreaming failed: %v", err)
	}

	// Verify output file exists and is non-empty
	info, err := os.Stat(outFile)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("output file is empty")
	}
}

func TestCollectLocalStreaming_Decrypt_Good(t *testing.T) {
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "secret.txt"), []byte("secret data"), 0644)

	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "test.stim")
	password := "round-trip-test"

	err := CollectLocalStreaming(srcDir, outFile, "gz", password)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	// Decrypt and verify contents
	decrypted, err := DecryptStimV2(outFile, password)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}

	if _, err := decrypted.Stat("secret.txt"); err != nil {
		t.Fatalf("secret.txt not found after decrypt: %v", err)
	}
}

func TestCollectLocalStreaming_BrokenSymlink_Good(t *testing.T) {
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "real.txt"), []byte("real"), 0644)
	os.Symlink("/nonexistent", filepath.Join(srcDir, "broken"))

	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "test.stim")

	// Should succeed, skipping broken symlink
	err := CollectLocalStreaming(srcDir, outFile, "none", "pass")
	if err != nil {
		t.Fatalf("expected success with broken symlink skip, got: %v", err)
	}
}

func TestCollectLocalStreaming_Bad(t *testing.T) {
	err := CollectLocalStreaming("/nonexistent", "/tmp/out.stim", "none", "pass")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test -run TestCollectLocalStreaming -v ./cmd/`
Expected: FAIL — `CollectLocalStreaming`, `DecryptStimV2` undefined

**Step 3: Implement streaming collection pipeline**

Add a new function in `cmd/collect_local.go` (keep existing `CollectLocal` for non-streaming formats):

```go
// CollectLocalStreaming collects a directory into an encrypted STIM v2 file
// using streaming I/O. Memory usage is constant (~2 MiB) regardless of input size.
//
// Pipeline: walk → tar → compress → encrypt → file
func CollectLocalStreaming(dir, output, compression, password string) error {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("source directory: %w", err)
	}

	// Create output file
	outFile, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("creating output: %w", err)
	}
	defer func() {
		outFile.Close()
		// Clean up partial file on error
		if err != nil {
			os.Remove(output)
		}
	}()

	// Build pipeline: tar → compress → encrypt → file
	// We build it in reverse order (outermost writer first)

	// Encryption layer (outermost)
	pr, pw := io.Pipe()

	// Start encryption in a goroutine (reads from pipe, writes to file)
	encErrCh := make(chan error, 1)
	go func() {
		encErrCh <- tim.StreamEncrypt(pr, outFile, password)
	}()

	// Compression layer
	compWriter, err := compress.NewCompressWriter(pw, compression)
	if err != nil {
		pw.Close()
		return fmt.Errorf("creating compressor: %w", err)
	}

	// Tar layer (innermost)
	tw := tar.NewWriter(compWriter)

	// Walk directory and write to tar
	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		// Check for symlinks
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return nil // skip broken
			}
			absTarget := target
			if !filepath.IsAbs(absTarget) {
				absTarget = filepath.Join(filepath.Dir(path), target)
			}
			if _, err := os.Stat(absTarget); err != nil {
				return nil // skip broken symlink
			}
			hdr := &tar.Header{
				Name:     filepath.ToSlash(rel),
				Typeflag: tar.TypeSymlink,
				Linkname: target,
				ModTime:  info.ModTime(),
				Mode:     0777,
			}
			return tw.WriteHeader(hdr)
		}

		if d.IsDir() {
			hdr := &tar.Header{
				Name:     filepath.ToSlash(rel) + "/",
				Typeflag: tar.TypeDir,
				Mode:     0755,
				ModTime:  info.ModTime(),
			}
			return tw.WriteHeader(hdr)
		}

		// Regular file — stream content
		hdr := &tar.Header{
			Name:     filepath.ToSlash(rel),
			Size:     info.Size(),
			Mode:     int64(info.Mode().Perm()),
			ModTime:  info.ModTime(),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(tw, f)
		return err
	})

	// Close pipeline layers in order
	tw.Close()
	compWriter.Close()
	pw.Close()

	// Wait for encryption to complete
	encErr := <-encErrCh

	if walkErr != nil {
		return fmt.Errorf("walking directory: %w", walkErr)
	}
	if encErr != nil {
		return fmt.Errorf("encrypting: %w", encErr)
	}

	return nil
}

// DecryptStimV2 decrypts a STIM v2 file back to a DataNode.
func DecryptStimV2(path, password string) (*datanode.DataNode, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Decrypt stream
	var tarBuf bytes.Buffer
	if err := tim.StreamDecrypt(f, &tarBuf, password); err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}

	// Decompress
	decompressed, err := compress.Decompress(tarBuf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("decompressing: %w", err)
	}

	// Parse tar into DataNode
	return datanode.FromTar(decompressed)
}
```

You'll need these imports at the top of `collect_local.go`:

```go
import (
	"archive/tar"
	"bytes"
	"io"
	"io/fs"

	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/tim"
)
```

**Step 4: Wire streaming into the collect local command**

In the existing `CollectLocal` function, add a check: if format is "stim", use the streaming path instead:

```go
// At the beginning of CollectLocal, after validation:
if format == "stim" {
	return CollectLocalStreaming(dir, output, compression, password)
}
```

The rest of the function continues to handle datanode/tim/trix formats via DataNode (existing path).

**Step 5: Run tests**

Run: `go test -run TestCollectLocalStreaming -v ./cmd/`
Expected: ALL PASS

**Step 6: Run full test suite**

Run: `go test -race ./...`
Expected: ALL PASS

**Step 7: Commit**

```bash
git add cmd/collect_local.go cmd/collect_local_test.go
git commit -m "feat(collect): add streaming pipeline for STIM v2 output"
```

---

## Task 10: Integration Test — Full Pipeline End-to-End

Create a comprehensive integration test that exercises the entire pipeline with realistic data.

**Files:**
- Create: `cmd/integration_test.go`

**Step 1: Write the integration test**

Create `cmd/integration_test.go`:

```go
package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFullPipeline_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a realistic directory structure
	srcDir := t.TempDir()
	password := "integration-test-2026"

	// Regular files
	os.WriteFile(filepath.Join(srcDir, "readme.md"), []byte("# Hello"), 0644)
	os.WriteFile(filepath.Join(srcDir, "config.json"), []byte(`{"key":"value"}`), 0644)

	// Nested directories
	os.MkdirAll(filepath.Join(srcDir, "src", "pkg"), 0755)
	os.WriteFile(filepath.Join(srcDir, "src", "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(srcDir, "src", "pkg", "lib.go"), []byte("package pkg"), 0644)

	// Large-ish file (1 MiB + 1 byte to cross block boundary)
	largeData := []byte(strings.Repeat("x", 1024*1024+1))
	os.WriteFile(filepath.Join(srcDir, "large.bin"), largeData, 0644)

	// Valid symlink
	os.Symlink(filepath.Join(srcDir, "readme.md"), filepath.Join(srcDir, "link-to-readme"))

	// Broken symlink (should be skipped)
	os.Symlink("/nonexistent/target", filepath.Join(srcDir, "broken-link"))

	// Hidden file (should be included with --hidden, excluded without)
	os.WriteFile(filepath.Join(srcDir, ".hidden"), []byte("hidden"), 0644)

	outDir := t.TempDir()

	// Test each compression mode
	for _, comp := range []string{"none", "gz", "xz"} {
		t.Run("compression_"+comp, func(t *testing.T) {
			outFile := filepath.Join(outDir, "test-"+comp+".stim")

			err := CollectLocalStreaming(srcDir, outFile, comp, password)
			if err != nil {
				t.Fatalf("CollectLocalStreaming(%s) failed: %v", comp, err)
			}

			// Verify output exists
			info, err := os.Stat(outFile)
			if err != nil {
				t.Fatalf("output not found: %v", err)
			}
			if info.Size() == 0 {
				t.Fatal("output is empty")
			}

			// Decrypt and verify
			dn, err := DecryptStimV2(outFile, password)
			if err != nil {
				t.Fatalf("decrypt failed: %v", err)
			}

			// Check files exist
			for _, name := range []string{"readme.md", "config.json", "src/main.go", "src/pkg/lib.go", "large.bin"} {
				if _, err := dn.Stat(name); err != nil {
					t.Errorf("file %s not found after round-trip: %v", name, err)
				}
			}

			// Check large file content
			f, err := dn.Open("large.bin")
			if err != nil {
				t.Fatalf("open large.bin: %v", err)
			}
			content := make([]byte, 100)
			n, _ := f.Read(content)
			if n == 0 || content[0] != 'x' {
				t.Fatal("large.bin content mismatch")
			}

			// Check broken symlink was skipped (no error during collection)
			if _, err := dn.Stat("broken-link"); err == nil {
				t.Error("broken symlink should have been skipped")
			}
		})
	}
}

func TestFullPipeline_WrongPassword_Bad(t *testing.T) {
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "secret.txt"), []byte("secret"), 0644)

	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "encrypted.stim")

	CollectLocalStreaming(srcDir, outFile, "none", "correct-password")

	_, err := DecryptStimV2(outFile, "wrong-password")
	if err == nil {
		t.Fatal("expected error with wrong password")
	}
}
```

**Step 2: Run integration tests**

Run: `go test -run TestFullPipeline -v ./cmd/`
Expected: ALL PASS

**Step 3: Run entire test suite with race detector**

Run: `go test -race ./...`
Expected: ALL PASS

**Step 4: Commit**

```bash
git add cmd/integration_test.go
git commit -m "test: add full pipeline integration tests for streaming collect"
```

---

## Task 11: Update Existing Command to Use Streaming + Docs

Wire the streaming path into the Cobra command, update documentation.

**Files:**
- Modify: `cmd/collect_local.go`
- Modify: `docs/plans/2026-02-21-borg-upgrade-design.md` (mark completed)

**Step 1: Update the collect local command runner**

In `cmd/collect_local.go`, modify the command's `RunE` function so that when `--format=stim` is selected, it calls `CollectLocalStreaming` instead of the old in-memory path. The existing `CollectLocal` function should have the early return added in Task 9 Step 4.

Also update the command's help text to mention the streaming mode:

```go
Long: `Collect local files into a portable container.

For STIM format, uses streaming I/O — memory usage is constant
(~2 MiB) regardless of input directory size. Other formats
(datanode, tim, trix) load files into memory.`,
```

**Step 2: Add --quiet flag usage**

In the `CollectLocal` function, replace direct `ui.NewProgressBar()` calls with `ProgressFromCmd(cmd)`:

```go
progress := ProgressFromCmd(cmd)
progress.Start("collecting " + dir)
// ... during walk ...
progress.Update(int64(fileCount), 0)
// ... after completion ...
progress.Finish(fmt.Sprintf("collected %d files", fileCount))
```

**Step 3: Run full test suite**

Run: `go test -race ./...`
Expected: ALL PASS

**Step 4: Mark design doc section as complete**

In `docs/plans/2026-02-21-borg-upgrade-design.md`, add at the top:

```markdown
**Status:** Implemented (2026-02-21)
```

**Step 5: Commit**

```bash
git add cmd/collect_local.go docs/plans/2026-02-21-borg-upgrade-design.md
git commit -m "feat(collect): wire streaming pipeline into CLI, add quiet mode support"
```

---

## Task 12: Manual Smoke Test + Final Verification

Build the binary, test it against a real directory, verify the backup workflow works end-to-end.

**Files:** None (verification only)

**Step 1: Build**

Run: `go build -o borg ./`
Expected: Clean build, no errors

**Step 2: Smoke test — collect + decrypt**

```bash
# Create test directory with some data
mkdir -p /tmp/borg-smoke/sub
echo "hello" > /tmp/borg-smoke/hello.txt
echo "nested" > /tmp/borg-smoke/sub/nested.txt
ln -s /tmp/borg-smoke/hello.txt /tmp/borg-smoke/link

# Collect with streaming encryption
./borg collect local /tmp/borg-smoke --format stim --compression xz --password "smoke-test" --output /tmp/borg-smoke-out.stim

# Verify output exists
ls -la /tmp/borg-smoke-out.stim

# Decode to verify
./borg decode /tmp/borg-smoke-out.stim -p "smoke-test" -o /tmp/borg-smoke-decoded.tar

# Cleanup
rm -rf /tmp/borg-smoke /tmp/borg-smoke-out.stim /tmp/borg-smoke-decoded.tar
```

**Step 3: Quiet mode test**

```bash
./borg collect local /tmp/borg-smoke --format stim --compression gz --password "test" --output /tmp/test.stim -q 2>&1 | head
# Should show minimal output (no spinner/progress bar)
```

**Step 4: Run full test suite one final time**

```bash
go test -race -cover ./...
go vet ./...
```
Expected: ALL PASS, no vet warnings, coverage >80%

**Step 5: Final commit if any cleanup needed**

```bash
git add -A
git commit -m "chore: final cleanup after smoke test"
```

---

## Dependency Summary

**New dependencies to add:**
```bash
go get golang.org/x/crypto/argon2
go get golang.org/x/term
go get golang.org/x/crypto/chacha20poly1305
```

**Note:** `golang.org/x/crypto/chacha20poly1305` may already be available transitively through Enchantrix. Check `go.sum` before adding.

## Task Order

Tasks MUST be executed in order — each builds on the previous:

```
Task 1  (Progress interface)
  ↓
Task 2  (--quiet flag)
  ↓
Task 3  (DataNode symlinks)
  ↓
Task 4  (DataNode AddPath)
  ↓
Task 5  (DataNode ToTarWriter)
  ↓
Task 6  (Argon2id key derivation)
  ↓
Task 7  (Chunked AEAD streaming encryption)
  ↓
Task 8  (Streaming compression)
  ↓
Task 9  (Streaming collect local pipeline)
  ↓
Task 10 (Integration tests)
  ↓
Task 11 (CLI wiring + docs)
  ↓
Task 12 (Smoke test + final verification)
```
