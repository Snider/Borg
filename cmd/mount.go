package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/fusefs"
	"github.com/Snider/Borg/pkg/tim"
	"github.com/Snider/Borg/pkg/trix"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/spf13/cobra"
)

var mountCmd = NewMountCmd()

func NewMountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mount [archive] [mountpoint]",
		Short: "Mount an archive as a read-only filesystem",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			archiveFile := args[0]
			mountpoint := args[1]
			password, _ := cmd.Flags().GetString("password")

			data, err := os.ReadFile(archiveFile)
			if err != nil {
				return err
			}

			var dn *datanode.DataNode
			if strings.HasSuffix(archiveFile, ".stim") || (len(data) >= 4 && string(data[:4]) == "STIM") {
				if password == "" {
					return fmt.Errorf("password required for .stim files")
				}
				m, err := tim.FromSigil(data, password)
				if err != nil {
					return err
				}
				tarball, err := m.ToTar()
				if err != nil {
					return err
				}
				dn, err = datanode.FromTar(tarball)
				if err != nil {
					return err
				}
			} else {
				// This handles .dat, .tar, .trix, and .tim files
				dn, err = trix.FromTrix(data, password)
				if err != nil {
					// If FromTrix fails, try FromTar as a fallback for plain tarballs
					if dn, err = datanode.FromTar(data); err != nil {
						return err
					}
				}
			}

			root := fusefs.NewDataNodeFs(dn)
			server, err := fs.Mount(mountpoint, root, &fs.Options{
				MountOptions: fuse.MountOptions{
					Debug: true,
				},
			})
			if err != nil {
				return err
			}

			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-c
				server.Unmount()
			}()

			fmt.Fprintf(cmd.OutOrStdout(), "Archive mounted at %s. Press Ctrl+C to unmount.\n", mountpoint)
			server.Wait()

			return nil
		},
	}
	cmd.Flags().StringP("password", "p", "", "Password for encrypted archives")
	return cmd
}

func GetMountCmd() *cobra.Command {
	return mountCmd
}

func init() {
	RootCmd.AddCommand(GetMountCmd())
}
