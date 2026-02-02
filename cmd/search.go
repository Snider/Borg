package cmd

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/datanode"
	"github.com/spf13/cobra"
)

// searchCmd represents the search command
var searchCmd = NewSearchCmd()

func init() {
	RootCmd.AddCommand(GetSearchCmd())
}

type searchResult struct {
	FilePath string
	LineNum  int
	Line     string
}

func NewSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <archive> <pattern>",
		Short: "Search for a pattern in an archive.",
		Long:  `Search for a pattern in a .dat, .tim, or .trix archive.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			archivePath, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("could not get absolute path for archive: %w", err)
			}
			pattern := args[1]

			// Read and decompress the archive
			compressedData, err := os.ReadFile(archivePath)
			if err != nil {
				return fmt.Errorf("failed to read archive: %w", err)
			}
			tarData, err := compress.Decompress(compressedData)
			if err != nil {
				return fmt.Errorf("failed to decompress archive: %w", err)
			}
			dn, err := datanode.FromTar(tarData)
			if err != nil {
				return fmt.Errorf("failed to load datanode: %w", err)
			}

			indexDir := filepath.Join(filepath.Dir(archivePath), ".borg-index")
			indexPath := filepath.Join(indexDir, "trigram.idx")

			var results []searchResult

			if _, err := os.Stat(indexPath); err == nil {
				results, err = searchWithIndex(dn, archivePath, pattern, cmd)
				if err != nil {
					return fmt.Errorf("error searching with index: %w", err)
				}
			} else {
				results, err = searchWithoutIndex(dn, pattern, cmd)
				if err != nil {
					return fmt.Errorf("error searching without index: %w", err)
				}
			}

			return printResults(cmd, dn, results)
		},
	}

	cmd.Flags().Bool("regex", false, "Use regex pattern")
	cmd.Flags().IntP("context", "C", 0, "Show N lines around match")
	cmd.Flags().String("type", "", "Filter by file extension")
	cmd.Flags().Int("max-results", 0, "Limit output to N results")

	return cmd
}

func printResults(cmd *cobra.Command, dn *datanode.DataNode, results []searchResult) error {
	contextLines, _ := cmd.Flags().GetInt("context")
	maxResults, _ := cmd.Flags().GetInt("max-results")

	if maxResults > 0 && len(results) > maxResults {
		results = results[:maxResults]
	}

	// Group results by file
	resultsByFile := make(map[string][]searchResult)
	for _, res := range results {
		resultsByFile[res.FilePath] = append(resultsByFile[res.FilePath], res)
	}

	// Process each file
	for filePath, fileResults := range resultsByFile {
		if contextLines > 0 {
			file, err := dn.Open(filePath)
			if err != nil {
				return fmt.Errorf("could not open file %s from archive: %w", filePath, err)
			}

			var lines []string
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}
			file.Close()

			for _, res := range fileResults {
				start := res.LineNum - 1 - contextLines
				if start < 0 {
					start = 0
				}

				end := res.LineNum + contextLines
				if end > len(lines) {
					end = len(lines)
				}

				for j := start; j < end; j++ {
					lineNum := j + 1
					line := lines[j]
					prefix := " "
					if lineNum == res.LineNum {
						prefix = ">"
					}
					fmt.Fprintf(cmd.OutOrStdout(), "%s %s:%d: %s\n", prefix, filePath, lineNum, line)
				}
				fmt.Fprintln(cmd.OutOrStdout(), "--")
			}
		} else {
			for _, res := range fileResults {
				fmt.Fprintf(cmd.OutOrStdout(), "%s:%d: %s\n", res.FilePath, res.LineNum, res.Line)
			}
		}
	}

	return nil
}

func searchWithoutIndex(dn *datanode.DataNode, pattern string, cmd *cobra.Command) ([]searchResult, error) {
	var results []searchResult

	useRegex, _ := cmd.Flags().GetBool("regex")
	fileType, _ := cmd.Flags().GetString("type")

	var re *regexp.Regexp
	var err error
	if useRegex {
		re, err = regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	err = dn.Walk(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if fileType != "" && !strings.HasSuffix(path, "."+fileType) {
			return nil
		}

		file, err := dn.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for lineNum := 1; scanner.Scan(); lineNum++ {
			line := scanner.Text()
			match := false
			if useRegex {
				if re.MatchString(line) {
					match = true
				}
			} else {
				if strings.Contains(line, pattern) {
					match = true
				}
			}

			if match {
				results = append(results, searchResult{
					FilePath: path,
					LineNum:  lineNum,
					Line:     strings.TrimSpace(line),
				})
			}
		}
		return scanner.Err()
	})

	if err != nil {
		return nil, fmt.Errorf("error walking datanode: %w", err)
	}

	return results, nil
}

func searchWithIndex(dn *datanode.DataNode, archivePath, pattern string, cmd *cobra.Command) ([]searchResult, error) {
	indexDir := filepath.Join(filepath.Dir(archivePath), ".borg-index")

	// Load file list
	fileListPath := filepath.Join(indexDir, "files.json")
	fileListData, err := os.ReadFile(fileListPath)
	if err != nil {
		return nil, fmt.Errorf("could not read file list: %w", err)
	}
	var fileList []string
	if err := json.Unmarshal(fileListData, &fileList); err != nil {
		return nil, fmt.Errorf("could not unmarshal file list: %w", err)
	}

	// Load trigram index
	trigramIndexPath := filepath.Join(indexDir, "trigram.idx")
	trigramIndexData, err := os.ReadFile(trigramIndexPath)
	if err != nil {
		return nil, fmt.Errorf("could not read trigram index: %w", err)
	}
	var trigramIndex map[[3]byte][]uint32
	decoder := gob.NewDecoder(bytes.NewReader(trigramIndexData))
	if err := decoder.Decode(&trigramIndex); err != nil {
		return nil, fmt.Errorf("could not decode trigram index: %w", err)
	}

	// Find candidate files
	candidateFiles := findCandidateFiles(pattern, trigramIndex, fileList)

	// Search within candidate files
	var results []searchResult
	useRegex, _ := cmd.Flags().GetBool("regex")
	fileType, _ := cmd.Flags().GetString("type")

	var re *regexp.Regexp
	if useRegex {
		re, err = regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	for path := range candidateFiles {
		if fileType != "" && !strings.HasSuffix(path, "."+fileType) {
			continue
		}

		file, err := dn.Open(path)
		if err != nil {
			return nil, fmt.Errorf("could not open file from archive: %w", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for lineNum := 1; scanner.Scan(); lineNum++ {
			line := scanner.Text()
			match := false
			if useRegex {
				if re.MatchString(line) {
					match = true
				}
			} else {
				if strings.Contains(line, pattern) {
					match = true
				}
			}

			if match {
				results = append(results, searchResult{
					FilePath: path,
					LineNum:  lineNum,
					Line:     strings.TrimSpace(line),
				})
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("error scanning file %s: %w", path, err)
		}
	}

	return results, nil
}

func findCandidateFiles(pattern string, trigramIndex map[[3]byte][]uint32, fileList []string) map[string]struct{} {
	if len(pattern) < 3 {
		// Fallback for short patterns
		candidateFiles := make(map[string]struct{})
		for _, file := range fileList {
			candidateFiles[file] = struct{}{}
		}
		return candidateFiles
	}

	// Generate trigrams from pattern
	var trigrams [][3]byte
	for i := 0; i <= len(pattern)-3; i++ {
		var trigram [3]byte
		copy(trigram[:], pattern[i:i+3])
		trigrams = append(trigrams, trigram)
	}

	// Find intersection of file IDs
	var intersection map[uint32]struct{}
	for i, trigram := range trigrams {
		postings := trigramIndex[trigram]
		if i == 0 {
			intersection = make(map[uint32]struct{})
			for _, fileID := range postings {
				intersection[fileID] = struct{}{}
			}
		} else {
			newIntersection := make(map[uint32]struct{})
			for _, fileID := range postings {
				if _, ok := intersection[fileID]; ok {
					newIntersection[fileID] = struct{}{}
				}
			}
			intersection = newIntersection
		}
	}

	candidateFiles := make(map[string]struct{})
	for fileID := range intersection {
		candidateFiles[fileList[fileID]] = struct{}{}
	}

	return candidateFiles
}

func GetSearchCmd() *cobra.Command {
	return searchCmd
}
