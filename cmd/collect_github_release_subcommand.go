package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Snider/Borg/pkg/datanode"
	borg_github "github.com/Snider/Borg/pkg/github"
	"github.com/google/go-github/v39/github"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

// collectGithubReleaseCmd represents the collect github-release command
var collectGithubReleaseCmd = &cobra.Command{
	Use:   "release [repository-url]",
	Short: "Download the latest release of a file from GitHub releases",
	Long:  `Download the latest release of a file from GitHub releases. If the file or URL has a version number, it will check for a higher version and download it if found.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logVal := cmd.Context().Value("logger")
		log, ok := logVal.(*slog.Logger)
		if !ok || log == nil {
			return errors.New("logger not properly initialised")
		}
		repoURL := args[0]
		outputDir, _ := cmd.Flags().GetString("output")
		pack, _ := cmd.Flags().GetBool("pack")
		file, _ := cmd.Flags().GetString("file")
		version, _ := cmd.Flags().GetString("version")

		_, err := GetRelease(log, repoURL, outputDir, pack, file, version)
		return err
	},
}

// init registers the 'collect github release' subcommand and its flags.
func init() {
	collectGithubCmd.AddCommand(collectGithubReleaseCmd)
	collectGithubReleaseCmd.PersistentFlags().String("output", ".", "Output directory for the downloaded file")
	collectGithubReleaseCmd.PersistentFlags().Bool("pack", false, "Pack all assets into a DataNode")
	collectGithubReleaseCmd.PersistentFlags().String("file", "", "The file to download from the release")
	collectGithubReleaseCmd.PersistentFlags().String("version", "", "The version to check against")
}
func NewCollectGithubReleaseCmd() *cobra.Command {
	return collectGithubReleaseCmd
}
func GetRelease(log *slog.Logger, repoURL string, outputDir string, pack bool, file string, version string) (*github.RepositoryRelease, error) {
	owner, repo, err := borg_github.ParseRepoFromURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository url: %w", err)
	}

	release, err := borg_github.GetLatestRelease(owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}

	log.Info("found latest release", "tag", release.GetTagName())

	if version != "" {
		if !semver.IsValid(version) {
			return nil, fmt.Errorf("invalid version string: %s", version)
		}
		if semver.Compare(release.GetTagName(), version) <= 0 {
			log.Info("latest release is not newer than the provided version", "latest", release.GetTagName(), "provided", version)
			return nil, nil
		}
	}

	if pack {
		dn := datanode.New()
		for _, asset := range release.Assets {
			log.Info("downloading asset", "name", asset.GetName())
			data, err := borg_github.DownloadReleaseAsset(asset)
			if err != nil {
				log.Error("failed to download asset", "name", asset.GetName(), "err", err)
				continue
			}
			dn.AddData(asset.GetName(), data)
		}
		tar, err := dn.ToTar()
		if err != nil {
			return nil, fmt.Errorf("failed to create datanode: %w", err)
		}
		outputFile := outputDir
		if !strings.HasSuffix(outputFile, ".dat") {
			outputFile = outputFile + ".dat"
		}
		err = os.WriteFile(outputFile, tar, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to write datanode: %w", err)
		}
		log.Info("datanode saved", "path", outputFile)
	} else {
		if len(release.Assets) == 0 {
			log.Info("no assets found in the latest release")
			return nil, nil
		}
		var assetToDownload *github.ReleaseAsset
		if file != "" {
			for _, asset := range release.Assets {
				if asset.GetName() == file {
					assetToDownload = asset
					break
				}
			}
			if assetToDownload == nil {
				return nil, fmt.Errorf("asset not found in the latest release: %s", file)
			}
		} else {
			assetToDownload = release.Assets[0]
		}
		outputPath := filepath.Join(outputDir, assetToDownload.GetName())
		log.Info("downloading asset", "name", assetToDownload.GetName())
		data, err := borg_github.DownloadReleaseAsset(assetToDownload)
		if err != nil {
			return nil, fmt.Errorf("failed to download asset: %w", err)
		}
		err = os.WriteFile(outputPath, data, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to write asset to file: %w", err)
		}
		log.Info("asset downloaded", "path", outputPath)
	}
	return release, nil
}
