package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/itchyny/gojq"
	"github.com/mattn/go-isatty"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

// collectBatchCmd represents the collect batch command
var collectBatchCmd = NewCollectBatchCmd()

func init() {
	GetCollectCmd().AddCommand(GetCollectBatchCmd())
}

func GetCollectBatchCmd() *cobra.Command {
	return collectBatchCmd
}

func NewCollectBatchCmd() *cobra.Command {
	collectBatchCmd := &cobra.Command{
		Use:   "batch [file|-]",
		Short: "Batch collect from a list of URLs",
		Long:  `Collect multiple resources from a list of URLs provided in a file or via stdin.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inputFile := args[0]
			outputDir, _ := cmd.Flags().GetString("output-dir")
			delay, _ := cmd.Flags().GetString("delay")
			skipExisting, _ := cmd.Flags().GetBool("continue")
			parallel, _ := cmd.Flags().GetInt("parallel")
			jqFilter, _ := cmd.Flags().GetString("jq")

			delayDuration, err := time.ParseDuration(delay)
			if err != nil {
				return fmt.Errorf("invalid delay duration: %w", err)
			}

			var reader io.Reader

			if inputFile == "-" {
				reader = os.Stdin
			} else {
				file, err := os.Open(inputFile)
				if err != nil {
					return fmt.Errorf("error opening input file: %w", err)
				}
				defer file.Close()
				reader = file
			}

			urls, err := readURLs(reader, jqFilter)
			if err != nil {
				return fmt.Errorf("error reading urls: %w", err)
			}

			if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
				return fmt.Errorf("error creating output directory: %w", err)
			}

			urlsChan := make(chan string, len(urls))
			var wg sync.WaitGroup
			var bar *progressbar.ProgressBar
			var outMutex sync.Mutex

			if isatty.IsTerminal(os.Stdout.Fd()) {
				bar = progressbar.Default(int64(len(urls)))
			}

			for i := 0; i < parallel; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for u := range urlsChan {
						downloadURL(cmd, u, outputDir, skipExisting, delayDuration, bar, &outMutex)
						if bar != nil {
							bar.Add(1)
						}
					}
				}()
			}

			for _, u := range urls {
				urlsChan <- u
			}
			close(urlsChan)

			wg.Wait()

			return nil
		},
	}

	collectBatchCmd.Flags().IntP("parallel", "p", 1, "Number of concurrent downloads")
	collectBatchCmd.Flags().String("delay", "0s", "Delay between requests")
	collectBatchCmd.Flags().StringP("output-dir", "o", ".", "Base output directory")
	collectBatchCmd.Flags().Bool("continue", false, "Skip already collected files")
	collectBatchCmd.Flags().String("jq", "", "jq filter to extract URLs from JSON input")

	return collectBatchCmd
}

func downloadURL(cmd *cobra.Command, u, outputDir string, skipExisting bool, delayDuration time.Duration, bar *progressbar.ProgressBar, outMutex *sync.Mutex) {
	fileName, err := getFileNameFromURL(u)
	if err != nil {
		logMessage(cmd, fmt.Sprintf("Skipping invalid URL %s: %v", u, err), bar, outMutex)
		return
	}
	filePath := filepath.Join(outputDir, fileName)

	if skipExisting {
		if _, err := os.Stat(filePath); err == nil {
			logMessage(cmd, fmt.Sprintf("Skipping already downloaded file: %s", filePath), bar, outMutex)
			return
		}
	}

	resp, err := http.Get(u)
	if err != nil {
		logMessage(cmd, fmt.Sprintf("Error downloading %s: %v", u, err), bar, outMutex)
		return
	}
	defer resp.Body.Close()

	out, err := os.Create(filePath)
	if err != nil {
		logMessage(cmd, fmt.Sprintf("Error creating file for %s: %v", u, err), bar, outMutex)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		logMessage(cmd, fmt.Sprintf("Error saving content for %s: %v", u, err), bar, outMutex)
		return
	}

	logMessage(cmd, fmt.Sprintf("Downloaded %s to %s", u, filePath), bar, outMutex)

	if delayDuration > 0 {
		time.Sleep(delayDuration)
	}
}

func logMessage(cmd *cobra.Command, msg string, bar *progressbar.ProgressBar, outMutex *sync.Mutex) {
	if bar != nil {
		bar.Describe(msg)
	} else {
		outMutex.Lock()
		defer outMutex.Unlock()
		fmt.Fprintln(cmd.OutOrStdout(), msg)
	}
}

func readURLs(reader io.Reader, jqFilter string) ([]string, error) {
	if jqFilter == "" {
		var urls []string
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				urls = append(urls, line)
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return urls, nil
	}

	query, err := gojq.Parse(jqFilter)
	if err != nil {
		return nil, fmt.Errorf("error parsing jq filter: %w", err)
	}

	var input interface{}
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&input); err != nil {
		return nil, fmt.Errorf("error decoding json: %w", err)
	}

	var urls []string
	iter := query.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, fmt.Errorf("error executing jq filter: %w", err)
		}
		if s, ok := v.(string); ok {
			urls = append(urls, s)
		}
	}
	return urls, nil
}

func getFileNameFromURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("invalid URL scheme: %s", parsedURL.Scheme)
	}
	if parsedURL.Path == "" || parsedURL.Path == "/" {
		return "index.html", nil
	}
	return filepath.Base(parsedURL.Path), nil
}
