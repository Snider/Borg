package cmd

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/tim"
	"github.com/Snider/Borg/pkg/trix"
	"github.com/Snider/Borg/pkg/ui"

	"github.com/spf13/cobra"
)

type CollectLocalCmd struct {
	cobra.Command
}

// NewCollectLocalCmd creates a new collect local command
func NewCollectLocalCmd() *CollectLocalCmd {
	c := &CollectLocalCmd{}
	c.Command = cobra.Command{
		Use:   "local [directory]",
		Short: "Collect files from a local directory",
		Long: `Collect local files into a portable container.

For STIM format, uses streaming I/O — memory usage is constant
(~2 MiB) regardless of input directory size. Other formats
(datanode, tim, trix) load files into memory.

Examples:
  borg collect local
  borg collect local ./src
  borg collect local /path/to/project --output project.tar
  borg collect local . --format stim --password secret
  borg collect local . --exclude "*.log" --exclude "node_modules"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			directory := "."
			if len(args) > 0 {
				directory = args[0]
			}

			outputFile, _ := cmd.Flags().GetString("output")
			format, _ := cmd.Flags().GetString("format")
			compression, _ := cmd.Flags().GetString("compression")
			password, _ := cmd.Flags().GetString("password")
			excludes, _ := cmd.Flags().GetStringSlice("exclude")
			includeHidden, _ := cmd.Flags().GetBool("hidden")
			respectGitignore, _ := cmd.Flags().GetBool("gitignore")

			progress := ProgressFromCmd(cmd)
			finalPath, err := CollectLocal(directory, outputFile, format, compression, password, excludes, includeHidden, respectGitignore, progress)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Files saved to", finalPath)
			return nil
		},
	}
	c.Flags().String("output", "", "Output file for the DataNode")
	c.Flags().String("format", "datanode", "Output format (datanode, tim, trix, or stim)")
	c.Flags().String("compression", "none", "Compression format (none, gz, or xz)")
	c.Flags().String("password", "", "Password for encryption (required for stim/trix format)")
	c.Flags().StringSlice("exclude", nil, "Patterns to exclude (can be specified multiple times)")
	c.Flags().Bool("hidden", false, "Include hidden files and directories")
	c.Flags().Bool("gitignore", true, "Respect .gitignore files (default: true)")
	return c
}

func init() {
	collectCmd.AddCommand(&NewCollectLocalCmd().Command)
}

// CollectLocal collects files from a local directory into a DataNode
func CollectLocal(directory string, outputFile string, format string, compression string, password string, excludes []string, includeHidden bool, respectGitignore bool, progress ui.Progress) (string, error) {
	// Validate format
	if format != "datanode" && format != "tim" && format != "trix" && format != "stim" {
		return "", fmt.Errorf("invalid format: %s (must be 'datanode', 'tim', 'trix', or 'stim')", format)
	}
	if (format == "stim" || format == "trix") && password == "" {
		return "", fmt.Errorf("password is required for %s format", format)
	}
	if compression != "none" && compression != "gz" && compression != "xz" {
		return "", fmt.Errorf("invalid compression: %s (must be 'none', 'gz', or 'xz')", compression)
	}

	// Resolve directory path
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return "", fmt.Errorf("error resolving directory path: %w", err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return "", fmt.Errorf("error accessing directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", absDir)
	}

	// Use streaming pipeline for STIM v2 format
	if format == "stim" {
		if outputFile == "" {
			baseName := filepath.Base(absDir)
			if baseName == "." || baseName == "/" {
				baseName = "local"
			}
			outputFile = baseName + ".stim"
		}
		if err := CollectLocalStreaming(absDir, outputFile, compression, password); err != nil {
			return "", err
		}
		return outputFile, nil
	}

	// Load gitignore patterns if enabled
	var gitignorePatterns []string
	if respectGitignore {
		gitignorePatterns = loadGitignore(absDir)
	}

	// Create DataNode and collect files
	dn := datanode.New()
	var fileCount int

	progress.Start("collecting " + directory)

	err = filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(absDir, path)
		if err != nil {
			return err
		}

		// Skip root
		if relPath == "." {
			return nil
		}

		// Skip hidden files/dirs unless explicitly included
		if !includeHidden && isHidden(relPath) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check gitignore patterns
		if respectGitignore && matchesGitignore(relPath, d.IsDir(), gitignorePatterns) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check exclude patterns
		if matchesExclude(relPath, excludes) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories (they're implicit in DataNode)
		if d.IsDir() {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading %s: %w", relPath, err)
		}

		// Add to DataNode with forward slashes (tar convention)
		dn.AddData(filepath.ToSlash(relPath), content)
		fileCount++
		progress.Update(int64(fileCount), 0)

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error walking directory: %w", err)
	}

	if fileCount == 0 {
		return "", fmt.Errorf("no files found in %s", directory)
	}

	progress.Finish(fmt.Sprintf("collected %d files", fileCount))

	// Convert to output format
	var data []byte
	if format == "tim" {
		t, err := tim.FromDataNode(dn)
		if err != nil {
			return "", fmt.Errorf("error creating tim: %w", err)
		}
		data, err = t.ToTar()
		if err != nil {
			return "", fmt.Errorf("error serializing tim: %w", err)
		}
	} else if format == "stim" {
		t, err := tim.FromDataNode(dn)
		if err != nil {
			return "", fmt.Errorf("error creating tim: %w", err)
		}
		data, err = t.ToSigil(password)
		if err != nil {
			return "", fmt.Errorf("error encrypting stim: %w", err)
		}
	} else if format == "trix" {
		data, err = trix.ToTrix(dn, password)
		if err != nil {
			return "", fmt.Errorf("error serializing trix: %w", err)
		}
	} else {
		data, err = dn.ToTar()
		if err != nil {
			return "", fmt.Errorf("error serializing DataNode: %w", err)
		}
	}

	// Apply compression
	compressedData, err := compress.Compress(data, compression)
	if err != nil {
		return "", fmt.Errorf("error compressing data: %w", err)
	}

	// Determine output filename
	if outputFile == "" {
		baseName := filepath.Base(absDir)
		if baseName == "." || baseName == "/" {
			baseName = "local"
		}
		outputFile = baseName + "." + format
		if compression != "none" {
			outputFile += "." + compression
		}
	}

	err = os.WriteFile(outputFile, compressedData, 0644)
	if err != nil {
		return "", fmt.Errorf("error writing output file: %w", err)
	}

	return outputFile, nil
}

// isHidden checks if a path component starts with a dot
func isHidden(path string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}

// loadGitignore loads patterns from .gitignore if it exists
func loadGitignore(dir string) []string {
	var patterns []string

	gitignorePath := filepath.Join(dir, ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		return patterns
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}

	return patterns
}

// matchesGitignore checks if a path matches any gitignore pattern
func matchesGitignore(path string, isDir bool, patterns []string) bool {
	for _, pattern := range patterns {
		// Handle directory-only patterns
		if strings.HasSuffix(pattern, "/") {
			if !isDir {
				continue
			}
			pattern = strings.TrimSuffix(pattern, "/")
		}

		// Handle negation (simplified - just skip negated patterns)
		if strings.HasPrefix(pattern, "!") {
			continue
		}

		// Match against path components
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		if matched {
			return true
		}

		// Also try matching the full path
		matched, _ = filepath.Match(pattern, path)
		if matched {
			return true
		}

		// Handle ** patterns (simplified)
		if strings.Contains(pattern, "**") {
			simplePattern := strings.ReplaceAll(pattern, "**", "*")
			matched, _ = filepath.Match(simplePattern, path)
			if matched {
				return true
			}
		}
	}
	return false
}

// matchesExclude checks if a path matches any exclude pattern
func matchesExclude(path string, excludes []string) bool {
	for _, pattern := range excludes {
		// Match against basename
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		if matched {
			return true
		}

		// Match against full path
		matched, _ = filepath.Match(pattern, path)
		if matched {
			return true
		}
	}
	return false
}

// CollectLocalStreaming collects files from a local directory using a streaming
// pipeline: walk -> tar -> compress -> encrypt -> file.
// The encryption runs in a goroutine, consuming from an io.Pipe that the
// tar/compress writes feed into synchronously.
func CollectLocalStreaming(dir, output, compression, password string) error {
	// Resolve to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("error resolving directory path: %w", err)
	}

	// Validate directory exists
	info, err := os.Stat(absDir)
	if err != nil {
		return fmt.Errorf("error accessing directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", absDir)
	}

	// Create output file
	outFile, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}

	// cleanup removes partial output on error
	cleanup := func() {
		outFile.Close()
		os.Remove(output)
	}

	// Build streaming pipeline:
	//   tar.Writer -> compressWriter -> pipeWriter -> pipeReader -> StreamEncrypt -> outFile
	pr, pw := io.Pipe()

	// Start encryption goroutine
	var encErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		encErr = tim.StreamEncrypt(pr, outFile, password)
	}()

	// Create compression writer wrapping the pipe writer
	compWriter, err := compress.NewCompressWriter(pw, compression)
	if err != nil {
		pw.Close()
		wg.Wait()
		cleanup()
		return fmt.Errorf("error creating compression writer: %w", err)
	}

	// Create tar writer wrapping the compression writer
	tw := tar.NewWriter(compWriter)

	// Walk directory and write tar entries
	walkErr := filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(absDir, path)
		if err != nil {
			return err
		}

		// Skip root
		if relPath == "." {
			return nil
		}

		// Normalize to forward slashes for tar
		relPath = filepath.ToSlash(relPath)

		// Check if entry is a symlink using Lstat
		linfo, err := os.Lstat(path)
		if err != nil {
			return err
		}
		isSymlink := linfo.Mode()&fs.ModeSymlink != 0

		if isSymlink {
			// Read symlink target
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}

			// Resolve to check if target exists
			absTarget := linkTarget
			if !filepath.IsAbs(absTarget) {
				absTarget = filepath.Join(filepath.Dir(path), linkTarget)
			}
			_, statErr := os.Stat(absTarget)
			if statErr != nil {
				// Broken symlink - skip silently
				return nil
			}

			// Write valid symlink as tar entry
			hdr := &tar.Header{
				Typeflag: tar.TypeSymlink,
				Name:     relPath,
				Linkname: linkTarget,
				Mode:     0777,
			}
			return tw.WriteHeader(hdr)
		}

		if d.IsDir() {
			// Write directory header
			hdr := &tar.Header{
				Typeflag: tar.TypeDir,
				Name:     relPath + "/",
				Mode:     0755,
			}
			return tw.WriteHeader(hdr)
		}

		// Regular file: write header + content
		finfo, err := d.Info()
		if err != nil {
			return err
		}

		hdr := &tar.Header{
			Name: relPath,
			Mode: 0644,
			Size: finfo.Size(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("error opening %s: %w", relPath, err)
		}
		defer f.Close()

		if _, err := io.Copy(tw, f); err != nil {
			return fmt.Errorf("error streaming %s: %w", relPath, err)
		}

		return nil
	})

	// Close pipeline layers in order: tar -> compress -> pipe
	// We must close even on error to unblock the encryption goroutine.
	twCloseErr := tw.Close()
	compCloseErr := compWriter.Close()

	if walkErr != nil {
		pw.CloseWithError(walkErr)
		wg.Wait()
		cleanup()
		return fmt.Errorf("error walking directory: %w", walkErr)
	}

	if twCloseErr != nil {
		pw.CloseWithError(twCloseErr)
		wg.Wait()
		cleanup()
		return fmt.Errorf("error closing tar writer: %w", twCloseErr)
	}

	if compCloseErr != nil {
		pw.CloseWithError(compCloseErr)
		wg.Wait()
		cleanup()
		return fmt.Errorf("error closing compression writer: %w", compCloseErr)
	}

	// Signal EOF to encryption goroutine
	pw.Close()

	// Wait for encryption to finish
	wg.Wait()

	if encErr != nil {
		cleanup()
		return fmt.Errorf("error encrypting data: %w", encErr)
	}

	// Close output file
	if err := outFile.Close(); err != nil {
		os.Remove(output)
		return fmt.Errorf("error closing output file: %w", err)
	}

	return nil
}

// DecryptStimV2 decrypts a STIM v2 file back into a DataNode.
// It opens the file, runs StreamDecrypt, decompresses the result,
// and parses the tar archive into a DataNode.
func DecryptStimV2(path, password string) (*datanode.DataNode, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer f.Close()

	// Decrypt
	var decrypted bytes.Buffer
	if err := tim.StreamDecrypt(f, &decrypted, password); err != nil {
		return nil, fmt.Errorf("error decrypting: %w", err)
	}

	// Decompress
	decompressed, err := compress.Decompress(decrypted.Bytes())
	if err != nil {
		return nil, fmt.Errorf("error decompressing: %w", err)
	}

	// Parse tar into DataNode
	dn, err := datanode.FromTar(decompressed)
	if err != nil {
		return nil, fmt.Errorf("error parsing tar: %w", err)
	}

	return dn, nil
}
