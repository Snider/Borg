package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Snider/Borg/pkg/datanode"
	borg_github "github.com/Snider/Borg/pkg/github"
	gh "github.com/google/go-github/v39/github"
	"github.com/spf13/cobra"
)

// collectGithubReleaseCmd represents the collect github-release command
var collectGithubReleaseCmd = &cobra.Command{
	Use:   "github-release [repository-url]",
	Short: "Download the latest release of a file from GitHub releases",
	Long:  `Download the latest release of a file from GitHub releases. If the file or URL has a version number, it will check for a higher version and download it if found.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoURL := args[0]
		outputDir, _ := cmd.Flags().GetString("output")
		pack, _ := cmd.Flags().GetBool("pack")
		file, _ := cmd.Flags().GetString("file")

		owner, repo, err := borg_github.ParseRepoFromURL(repoURL)
		if err != nil {
			fmt.Printf("Error parsing repository URL: %v\n", err)
			return
		}

		release, err := borg_github.GetLatestRelease(owner, repo)
		if err != nil {
			fmt.Printf("Error getting latest release: %v\n", err)
			return
		}

		fmt.Printf("Found latest release: %s\n", release.GetTagName())

		if pack {
			dn := datanode.New()
			for _, asset := range release.Assets {
				fmt.Printf("Downloading asset: %s\n", asset.GetName())
				resp, err := http.Get(asset.GetBrowserDownloadURL())
				if err != nil {
					fmt.Printf("Error downloading asset: %v\n", err)
					continue
				}
				defer resp.Body.Close()
				var buf bytes.Buffer
				_, err = io.Copy(&buf, resp.Body)
				if err != nil {
					fmt.Printf("Error reading asset: %v\n", err)
					continue
				}
				dn.AddData(asset.GetName(), buf.Bytes())
			}
			tar, err := dn.ToTar()
			if err != nil {
				fmt.Printf("Error creating DataNode: %v\n", err)
				return
			}
			outputFile := outputDir
			if !strings.HasSuffix(outputFile, ".dat") {
				outputFile = outputFile + ".dat"
			}
			err = os.WriteFile(outputFile, tar, 0644)
			if err != nil {
				fmt.Printf("Error writing DataNode: %v\n", err)
				return
			}
			fmt.Printf("DataNode saved to %s\n", outputFile)
		} else {
			if len(release.Assets) == 0 {
				fmt.Println("No assets found in the latest release.")
				return
			}
			var assetToDownload *gh.ReleaseAsset
			if file != "" {
				for _, asset := range release.Assets {
					if asset.GetName() == file {
						assetToDownload = asset
						break
					}
				}
				if assetToDownload == nil {
					fmt.Printf("Asset '%s' not found in the latest release.\n", file)
					return
				}
			} else {
				assetToDownload = release.Assets[0]
			}
			outputPath := filepath.Join(outputDir, assetToDownload.GetName())
			fmt.Printf("Downloading asset: %s\n", assetToDownload.GetName())
			err = borg_github.DownloadReleaseAsset(assetToDownload, outputPath)
			if err != nil {
				fmt.Printf("Error downloading asset: %v\n", err)
				return
			}
			fmt.Printf("Asset downloaded to %s\n", outputPath)
		}
	},
}

func init() {
	collectCmd.AddCommand(collectGithubReleaseCmd)
	collectGithubReleaseCmd.PersistentFlags().String("output", ".", "Output directory for the downloaded file")
	collectGithubReleaseCmd.PersistentFlags().Bool("pack", false, "Pack all assets into a DataNode")
	collectGithubReleaseCmd.PersistentFlags().String("file", "", "The file to download from the release")
}
