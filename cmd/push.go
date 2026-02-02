package cmd

import (
	"fmt"
	"net/url"
	"os"

	"github.com/Snider/Borg/pkg/storage"
	"github.com/spf13/cobra"
)

func NewPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push [local-path] [remote-url]",
		Short: "Push a local file to a remote storage URL",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			localPath := args[0]
			remoteURL := args[1]

			u, err := url.Parse(remoteURL)
			if err != nil {
				return fmt.Errorf("invalid remote URL: %w", err)
			}

			s, err := storage.NewStorage(u)
			if err != nil {
				return err
			}

			f, err := os.Open(localPath)
			if err != nil {
				return fmt.Errorf("error opening local file: %w", err)
			}
			defer f.Close()

			err = s.Write(u.Path, f)
			if err != nil {
				return fmt.Errorf("error uploading file: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "File pushed successfully")
			return nil
		},
	}
}

func init() {
	RootCmd.AddCommand(NewPushCmd())
}
