package cmd

import (
	"github.com/spf13/cobra"
)

// collectGithubCmd represents the collect github command
var collectGithubCmd = &cobra.Command{
	Use:   "github",
	Short: "Collect a resource from GitHub.",
	Long:  `Collect a resource from a GitHub repository, such as a repository or a release.`,
}

func init() {
	collectCmd.AddCommand(collectGithubCmd)
}
func NewCollectGithubCmd() *cobra.Command {
	return collectGithubCmd
}
