package cmd

import (
	"fmt"
	"path/filepath"
	"time"
	"github.com/Snider/Borg/pkg/wayback"
	"github.com/spf13/cobra"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
)

// waybackCmd represents the wayback command
var waybackCmd = NewWaybackCmd()
var waybackListCmd = NewWaybackListCmd()
var waybackCollectCmd = NewWaybackCollectCmd()

func init() {
	RootCmd.AddCommand(GetWaybackCmd())
	GetWaybackCmd().AddCommand(GetWaybackListCmd())
	GetWaybackCmd().AddCommand(GetWaybackCollectCmd())
}

func GetWaybackCmd() *cobra.Command {
	return waybackCmd
}

func GetWaybackListCmd() *cobra.Command {
	return waybackListCmd
}

func GetWaybackCollectCmd() *cobra.Command {
	return waybackCollectCmd
}

func NewWaybackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wayback",
		Short: "Interact with the Internet Archive Wayback Machine.",
		Long:  `List and collect historical snapshots of websites from the Internet Archive Wayback Machine.`,
	}
	return cmd
}

func NewWaybackListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [url]",
		Short: "List available snapshots for a URL.",
		Long:  `Queries the Wayback Machine CDX API to find all available snapshots for a given URL.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			snapshots, err := wayback.ListSnapshots(url)
			if err != nil {
				return fmt.Errorf("failed to list snapshots: %w", err)
			}

			if len(snapshots) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No snapshots found.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "TIMESTAMP\tMIMETYPE\tSTATUS\tLENGTH\tURL")
			for _, s := range snapshots {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", s.Timestamp, s.MimeType, s.StatusCode, s.Length, s.Original)
			}
			return w.Flush()
		},
	}
	return cmd
}

func NewWaybackCollectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collect [url]",
		Short: "Collect a snapshot of a website.",
		Long:  `Collects a snapshot of a website from the Wayback Machine.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			urlArg := args[0]
			outputDir, _ := cmd.Flags().GetString("output")
			latest, _ := cmd.Flags().GetBool("latest")
			all, _ := cmd.Flags().GetBool("all")
			date, _ := cmd.Flags().GetString("date")

			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}

			baseURL, err := url.Parse(urlArg)
			if err != nil {
				return fmt.Errorf("failed to parse URL: %w", err)
			}

			snapshots, err := wayback.ListSnapshots(urlArg)
			if err != nil {
				return fmt.Errorf("failed to list snapshots: %w", err)
			}
			if len(snapshots) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No snapshots found.")
				return nil
			}

			var timestamps []string
			if latest {
				timestamps = []string{snapshots[len(snapshots)-1].Timestamp}
			} else if all {
				for _, s := range snapshots {
					timestamps = append(timestamps, s.Timestamp)
				}
			} else if date != "" {
				filtered := filterSnapshotsByDate(snapshots, date)
				if len(filtered) == 0 {
					return fmt.Errorf("no snapshots found for date: %s", date)
				}
				for _, s := range filtered {
					timestamps = append(timestamps, s.Timestamp)
				}
			} else {
				return fmt.Errorf("either --latest, --all, or --date must be specified")
			}

			timeline := ""
			downloadedDigests := make(map[string]bool)

			assets, err := wayback.ListSnapshots(fmt.Sprintf("%s/*", urlArg))
			if err != nil {
				return fmt.Errorf("failed to list assets: %w", err)
			}

			for _, ts := range timestamps {
				fmt.Fprintf(cmd.OutOrStdout(), "Collecting snapshot from %s...\n", ts)
				snapshotDir := filepath.Join(outputDir, ts)
				if err := os.MkdirAll(snapshotDir, 0755); err != nil {
					return fmt.Errorf("failed to create snapshot directory: %w", err)
				}

				rootSnapshot := wayback.Snapshot{Timestamp: ts, Original: urlArg}
				if err := downloadAndProcess(rootSnapshot, snapshotDir, baseURL, downloadedDigests); err != nil {
					return err
				}

				timeline += fmt.Sprintf("- %s: %s\n", ts, urlArg)
			}

func downloadAndProcess(snapshot wayback.Snapshot, snapshotDir string, baseURL *url.URL, downloadedDigests map[string]bool) error {
	if downloadedDigests[snapshot.Digest] {
		return nil
	}
	time.Sleep(200 * time.Millisecond) // Simple rate-limiting
	fmt.Printf("  Downloading %s\n", snapshot.Original)
	data, err := wayback.DownloadSnapshot(snapshot)
	if err != nil {
		return fmt.Errorf("failed to download asset %s: %w", snapshot.Original, err)
	}
	downloadedDigests[snapshot.Digest] = true

	assetURL, err := url.Parse(snapshot.Original)
	if err != nil {
		return fmt.Errorf("failed to parse asset URL %s: %w", snapshot.Original, err)
	}
	path := assetURL.Path
	if strings.HasSuffix(path, "/") {
		path = filepath.Join(path, "index.html")
	}
	filePath := filepath.Join(snapshotDir, path)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create asset directory for %s: %w", filePath, err)
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write asset %s: %w", filePath, err)
	}

	if strings.HasPrefix(snapshot.MimeType, "text/html") {
		rewrittenData, err := wayback.RewriteLinks(data, baseURL)
		if err != nil {
			return fmt.Errorf("failed to rewrite links for %s: %w", snapshot.Original, err)
		}
		if err := os.WriteFile(filePath, rewrittenData, 0644); err != nil {
			return fmt.Errorf("failed to write rewritten asset %s: %w", filePath, err)
		}

		links, err := wayback.ExtractLinks(data)
		if err != nil {
			return fmt.Errorf("failed to extract links from %s: %w", snapshot.Original, err)
		}

		for _, link := range links {
			absoluteURL := assetURL.ResolveReference(&url.URL{Path: link})
			assetSnapshot := wayback.Snapshot{Timestamp: snapshot.Timestamp, Original: absoluteURL.String()}
			if err := downloadAndProcess(assetSnapshot, snapshotDir, baseURL, downloadedDigests); err != nil {
				fmt.Printf("Warning: failed to process asset %s: %v\n", absoluteURL.String(), err)
			}
		}
	}
	return nil

			timelineFile := filepath.Join(outputDir, "TIMELINE.md")
			if err := os.WriteFile(timelineFile, []byte(timeline), 0644); err != nil {
				return fmt.Errorf("failed to write timeline file: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Snapshots saved to %s\n", outputDir)
			return nil
		},
	}
	cmd.Flags().Bool("latest", false, "Collect the latest available snapshot.")
	cmd.Flags().Bool("all", false, "Collect all available snapshots.")
	cmd.Flags().String("date", "", "Collect a snapshot from a specific date (YYYY-MM-DD).")
	cmd.Flags().String("output", "", "Output directory for the collected snapshots.")
	cmd.MarkFlagRequired("output")
	return cmd
}

func filterSnapshotsByDate(snapshots []wayback.Snapshot, date string) []wayback.Snapshot {
	var filtered []wayback.Snapshot
	for _, s := range snapshots {
		if len(s.Timestamp) >= 8 && s.Timestamp[:8] == date[:4]+date[5:7]+date[8:10] {
			filtered = append(filtered, s)
		}
	}
	return filtered
}
