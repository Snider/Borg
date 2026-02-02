package cmd

import "github.com/spf13/cobra"

var remoteCmd = NewRemoteCmd()

func init() {
	RootCmd.AddCommand(GetRemoteCmd())
}

func NewRemoteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remote",
		Short: "Manage remote storage configurations",
		Long:  `Add, remove, and list remote storage configurations for S3, R2, B2, etc.`,
	}
}

func GetRemoteCmd() *cobra.Command {
	return remoteCmd
}
