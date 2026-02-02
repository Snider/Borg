package cmd

import (
	"github.com/Snider/Borg/pkg/config"
	"github.com/spf13/cobra"
)

func NewRemoteAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [name] [url]",
		Short: "Add a new remote storage configuration",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			url := args[1]
			accessKey, _ := cmd.Flags().GetString("access-key")
			secretKey, _ := cmd.Flags().GetString("secret-key")

			endpoint, _ := cmd.Flags().GetString("endpoint")
			remote := config.Remote{
				Name:      name,
				URL:       url,
				AccessKey: accessKey,
				SecretKey: secretKey,
				Endpoint:  endpoint,
			}

			remotes, err := config.LoadRemotes()
			if err != nil {
				return err
			}

			remotes = append(remotes, remote)

			return config.SaveRemotes(remotes)
		},
	}

	cmd.Flags().String("access-key", "", "Access key for the remote")
	cmd.Flags().String("secret-key", "", "Secret key for the remote")
	cmd.Flags().String("endpoint", "", "Custom endpoint for S3-compatible storage")

	return cmd
}

func init() {
	remoteCmd.AddCommand(NewRemoteAddCmd())
}
