package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	borg_github "github.com/Snider/Borg/pkg/github"
	"bytes"
	"github.com/google/go-github/v39/github"
	"github.com/spf13/cobra"
)

func NewCollectGithubReleaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release <owner/repo> [tag]",
		Short: "Download a release from GitHub",
		Long:  `Download a specific release from GitHub. If no tag is specified, the latest release will be downloaded.`,
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			logVal := cmd.Context().Value("logger")
			log, ok := logVal.(*slog.Logger)
			if !ok || log == nil {
				return errors.New("logger not properly initialised")
			}
			repoURL := args[0]
			outputDir, _ := cmd.Flags().GetString("output")
			assetsOnly, _ := cmd.Flags().GetBool("assets-only")
			pattern, _ := cmd.Flags().GetString("pattern")
			verifyChecksums, _ := cmd.Flags().GetBool("verify-checksums")

			owner, repo, err := borg_github.ParseRepoFromURL(repoURL)
			if err != nil {
				return fmt.Errorf("failed to parse repository url: %w", err)
			}

			var release *github.RepositoryRelease
			if len(args) == 2 {
				tag := args[1]
				log.Info("getting release by tag", "tag", tag)
				release, err = borg_github.GetReleaseByTag(owner, repo, tag)
				if err != nil {
					return fmt.Errorf("failed to get release '%s': %w", tag, err)
				}
			} else {
				log.Info("getting latest release")
				release, err = borg_github.GetLatestRelease(owner, repo)
				if err != nil {
					return fmt.Errorf("failed to get latest release: %w", err)
				}
			}
			if release == nil {
				return errors.New("release not found")
			}

			log.Info("found release", "tag", release.GetTagName())

			tag := release.GetTagName()
			releaseDir := filepath.Join(outputDir, tag)
			if err := os.MkdirAll(releaseDir, 0755); err != nil {
				return fmt.Errorf("failed to create release directory: %w", err)
			}

			if !assetsOnly {
				releaseNotes := release.GetBody()
				if err := os.WriteFile(filepath.Join(releaseDir, "RELEASE.md"), []byte(releaseNotes), 0644); err != nil {
					return fmt.Errorf("failed to write release notes: %w", err)
				}
			}

			for _, asset := range release.Assets {
				if pattern != "" {
					matched, err := filepath.Match(pattern, asset.GetName())
					if err != nil {
						return fmt.Errorf("invalid pattern: %w", err)
					}
					if !matched {
						continue
					}
				}

				log.Info("downloading asset", "name", asset.GetName())
					assetPath := filepath.Join(releaseDir, asset.GetName())
					file, err := os.Create(assetPath)
					if err != nil {
						return fmt.Errorf("failed to create asset file: %w", err)
					}
					defer file.Close()

					var checksumData []byte
				if verifyChecksums {
						checksumAsset := findChecksumAsset(release.Assets)
						if checksumAsset == nil {
							log.Warn("checksum file not found in release", "tag", tag)
						} else {
							buf := new(bytes.Buffer)
							err := borg_github.DownloadReleaseAsset(checksumAsset, buf)
							if err != nil {
								log.Error("failed to download checksum file", "name", checksumAsset.GetName(), "err", err)
							} else {
								checksumData = buf.Bytes()
							}
						}
					}

					if verifyChecksums && checksumData != nil {
						err = borg_github.DownloadReleaseAssetWithChecksum(asset, checksumData, file)
				} else {
						err = borg_github.DownloadReleaseAsset(asset, file)
				}

				if err != nil {
					log.Error("failed to download asset", "name", asset.GetName(), "err", err)
					continue
				}
			}
			return nil
		},
	}
	cmd.Flags().String("output", "releases", "Output directory for the releases")
	cmd.Flags().Bool("assets-only", false, "Only download assets, skip release notes")
	cmd.Flags().String("pattern", "", "Filter assets by filename pattern")
	cmd.Flags().Bool("verify-checksums", false, "Verify checksums after download")
	return cmd
}

func init() {
	collectGithubCmd.AddCommand(NewCollectGithubReleaseCmd())
}
