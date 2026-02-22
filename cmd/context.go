package cmd

import (
	"os"

	"forge.lthn.ai/Snider/Borg/pkg/ui"
	"github.com/spf13/cobra"
)

// ProgressFromCmd returns a Progress based on --quiet flag and TTY detection.
func ProgressFromCmd(cmd *cobra.Command) ui.Progress {
	quiet, _ := cmd.Flags().GetBool("quiet")
	if quiet {
		return ui.NewQuietProgress(os.Stderr)
	}
	return ui.DefaultProgress()
}
