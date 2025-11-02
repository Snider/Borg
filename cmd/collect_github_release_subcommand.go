package cmd

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Snider/Borg/pkg/datanode"
	borg_github "github.com/Snider/Borg/pkg/github"
	gh "github.com/google/go-github/v39/github"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

// collectGithubReleaseCmd represents the collect github-release command
var collectGithubReleaseCmd = &cobra.Command{
	Use:   "release [repository-url]",
	Short: "Download the latest release of a file from GitHub releases",
	Long:  `Download the latest release of a file from GitHub releases. If the file or URL has a version number, it will check for a higher version and download it if found.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logVal := cmd.Context().Value("logger")
		log, ok := logVal.(*slog.Logger)
		if !ok || log == nil {
			fmt.Fprintln(os.Stderr, "Error: logger not properly initialised")
			return
		}
		repoURL := args[0]
		outputDir, _ := cmd.Flags().GetString("output")
		pack, _ := cmd.Flags().GetBool("pack")
		file, _ := cmd.Flags().GetString("file")
		version, _ := cmd.Flags().GetString("version")

		owner, repo, err := borg_github.ParseRepoFromURL(repoURL)
		if err != nil {
			log.Error("failed to parse repository url", "err", err)
			return
		}

		release, err := borg_github.GetLatestRelease(owner, repo)
		if err != nil {
			log.Error("failed to get latest release", "err", err)
			return
		}

		log.Info("found latest release", "tag", release.GetTagName())

		if version != "" {
			if !semver.IsValid(version) {
				log.Error("invalid version string", "version", version)
				return
			}
			if semver.Compare(release.GetTagName(), version) <= 0 {
				log.Info("latest release is not newer than the provided version", "latest", release.GetTagName(), "provided", version)
				return
			}
		}

		if pack {
			dn := datanode.New()
			for _, asset := range release.Assets {
				log.Info("downloading asset", "name", asset.GetName())
				resp, err := http.Get(asset.GetBrowserDownloadURL())
				if err != nil {
					log.Error("failed to download asset", "name", asset.GetName(), "err", err)
					continue
				}
				defer resp.Body.Close()
				var buf bytes.Buffer
				_, err = io.Copy(&buf, resp.Body)
				if err != nil {
					log.Error("failed to read asset", "name", asset.GetName(), "err", err)
					continue
				}
				dn.AddData(asset.GetName(), buf.Bytes())
			}
			tar, err := dn.ToTar()
			if err != nil {
				log.Error("failed to create datanode", "err", err)
				return
			}
			outputFile := outputDir
			if !strings.HasSuffix(outputFile, ".dat") {
				outputFile = outputFile + ".dat"
			}
			err = os.WriteFile(outputFile, tar, 0644)
			if err != nil {
				log.Error("failed to write datanode", "err", err)
				return
			}
			log.Info("datanode saved", "path", outputFile)
		} else {
			if len(release.Assets) == 0 {
				log.Info("no assets found in the latest release")
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
					log.Error("asset not found in the latest release", "asset", file)
					return
				}
			} else {
				assetToDownload = release.Assets[0]
			}
			outputPath := filepath.Join(outputDir, assetToDownload.GetName())
			log.Info("downloading asset", "name", assetToDownload.GetName())
			err = borg_github.DownloadReleaseAsset(assetToDownload, outputPath)
			if err != nil {
				log.Error("failed to download asset", "name", assetToDownload.GetName(), "err", err)
				return
			}
			log.Info("asset downloaded", "path", outputPath)
		}
	},
}

func init() {
	collectGithubCmd.AddCommand(collectGithubReleaseCmd)
	collectGithubReleaseCmd.PersistentFlags().String("output", ".", "Output directory for the downloaded file")
	collectGithubReleaseCmd.PersistentFlags().Bool("pack", false, "Pack all assets into a DataNode")
	collectGithubReleaseCmd.PersistentFlags().String("file", "", "The file to download from the release")
	collectGithubReleaseCmd.PersistentFlags().String("version", "", "The version to check against")
}
