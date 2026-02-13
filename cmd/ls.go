package cmd

import (
	"fmt"
	"net/url"

	"github.com/Snider/Borg/pkg/storage"
	"github.com/spf13/cobra"
)

func NewLsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls [remote-url]",
		Short: "List the contents of a remote storage path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			remoteURL := args[0]

			u, err := url.Parse(remoteURL)
			if err != nil {
				return fmt.Errorf("invalid remote URL: %w", err)
			}

			s, err := storage.NewStorage(u)
			if err != nil {
				return err
			}

			paths, err := s.List(u.Path)
			if err != nil {
				return fmt.Errorf("error listing contents: %w", err)
			}

			for _, path := range paths {
				fmt.Fprintln(cmd.OutOrStdout(), path)
			}

			return nil
		},
	}
}

func init() {
	RootCmd.AddCommand(NewLsCmd())
}
