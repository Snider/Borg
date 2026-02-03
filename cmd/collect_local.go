package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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
		Long: `Collect files from a local directory and store them in a DataNode.

If no directory is specified, the current working directory is used.

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

			finalPath, err := CollectLocal(directory, outputFile, format, compression, password, excludes, includeHidden, respectGitignore)
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
func CollectLocal(directory string, outputFile string, format string, compression string, password string, excludes []string, includeHidden bool, respectGitignore bool) (string, error) {
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

	// Load gitignore patterns if enabled
	var gitignorePatterns []string
	if respectGitignore {
		gitignorePatterns = loadGitignore(absDir)
	}

	// Create DataNode and collect files
	dn := datanode.New()
	var fileCount int

	bar := ui.NewProgressBar(-1, "Scanning files")
	defer bar.Finish()

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
		bar.Describe(fmt.Sprintf("Collected %d files", fileCount))

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error walking directory: %w", err)
	}

	if fileCount == 0 {
		return "", fmt.Errorf("no files found in %s", directory)
	}

	bar.Describe(fmt.Sprintf("Packaging %d files", fileCount))

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
