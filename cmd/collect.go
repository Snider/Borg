package cmd

import (
	"time"

	"github.com/spf13/cobra"
)

// collectCmd represents the collect command
var collectCmd = NewCollectCmd()

func init() {
	RootCmd.AddCommand(GetCollectCmd())
}
func NewCollectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collect",
		Short: "Collect a resource from a URI.",
		Long:  `Collect a resource from a URI and store it in a DataNode.`,
	}

	cmd.PersistentFlags().Duration("timeout", 0, "Total request timeout (e.g., 60s). 0 means no timeout.")
	cmd.PersistentFlags().Duration("connect-timeout", 10*time.Second, "TCP connection establishment timeout (e.g., 10s)")
	cmd.PersistentFlags().Duration("tls-timeout", 10*time.Second, "TLS handshake timeout (e.g., 10s)")
	cmd.PersistentFlags().Duration("header-timeout", 30*time.Second, "Time to receive response headers timeout (e.g., 30s)")

	return cmd
}

func GetCollectCmd() *cobra.Command {
	return collectCmd
}
