package cmd

import (
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/Snider/Borg/pkg/storage"
	"github.com/spf13/cobra"
)

func NewPullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull [remote-url] [local-path]",
		Short: "Pull a remote file from a remote storage URL",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			remoteURL := args[0]
			localPath := args[1]

			u, err := url.Parse(remoteURL)
			if err != nil {
				return fmt.Errorf("invalid remote URL: %w", err)
			}

			s, err := storage.NewStorage(u)
			if err != nil {
				return err
			}

			r, err := s.Read(u.Path)
			if err != nil {
				return fmt.Errorf("error downloading file: %w", err)
			}
			defer r.Close()

			f, err := os.Create(localPath)
			if err != nil {
				return fmt.Errorf("error creating local file: %w", err)
			}
			defer f.Close()

			_, err = io.Copy(f, r)
			if err != nil {
				return fmt.Errorf("error writing to local file: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "File pulled successfully")
			return nil
		},
	}
}

func init() {
	RootCmd.AddCommand(NewPullCmd())
}
