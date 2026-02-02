package cmd

import (
	"fmt"
	"strings"

	"github.com/Snider/Borg/pkg/docker"
	"github.com/spf13/cobra"
)

// collectDockerCmd represents the collect docker command
var collectDockerCmd = NewCollectDockerCmd()

func init() {
	GetCollectCmd().AddCommand(GetCollectDockerCmd())
}

func GetCollectDockerCmd() *cobra.Command {
	return collectDockerCmd
}

func NewCollectDockerCmd() *cobra.Command {
	collectDockerCmd := &cobra.Command{
		Use:   "docker [image]",
		Short: "Collect a Docker image",
		Long:  `Collect a Docker image and save it as an OCI tarball.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			imageRef := args[0]
			outputFile, err := cmd.Flags().GetString("output")
			if err != nil {
				return fmt.Errorf("error getting output flag: %w", err)
			}
			allTags, err := cmd.Flags().GetBool("all-tags")
			if err != nil {
				return fmt.Errorf("error getting all-tags flag: %w", err)
			}
			platform, err := cmd.Flags().GetString("platform")
			if err != nil {
				return fmt.Errorf("error getting platform flag: %w", err)
			}
			registry, err := cmd.Flags().GetString("registry")
			if err != nil {
				return fmt.Errorf("error getting registry flag: %w", err)
			}

			if outputFile == "" && !allTags {
				// Create a default output file name from the image ref
				// by replacing slashes and colons with underscores.
				// e.g., letheanmovement/chain:v1.0.0 -> letheanmovement_chain_v1.0.0.tar
				safeRef := strings.ReplaceAll(imageRef, "/", "_")
				safeRef = strings.ReplaceAll(safeRef, ":", "_")
				outputFile = safeRef + ".tar"
			}

			err := docker.Collect(imageRef, outputFile, allTags, platform, registry)
			if err != nil {
				return fmt.Errorf("error collecting docker image: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Docker image saved to", outputFile)
			return nil
		},
	}
	collectDockerCmd.Flags().StringP("output", "o", "", "Output file for the OCI tarball")
	collectDockerCmd.Flags().Bool("all-tags", false, "Collect all available tags")
	collectDockerCmd.Flags().String("platform", "", "Specific platform (e.g., linux/amd64)")
	collectDockerCmd.Flags().String("registry", "", "Custom registry URL")

	return collectDockerCmd
}
