package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/spf13/cobra"
)

var unmountCmd = NewUnmountCmd()

func NewUnmountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unmount [mountpoint]",
		Short: "Unmount a filesystem",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mountpoint := args[0]
			server, _ := fuse.NewServer(nil, mountpoint, nil)
			err := server.Unmount()
			if err == nil {
				return nil
			}

			// Fallback to system commands
			var unmountCmd *exec.Cmd
			switch runtime.GOOS {
			case "linux":
				unmountCmd = exec.Command("fusermount", "-u", mountpoint)
			case "darwin":
				unmountCmd = exec.Command("umount", mountpoint)
			default:
				return fmt.Errorf("unmount not supported on %s: %v", runtime.GOOS, err)
			}

			return unmountCmd.Run()
		},
	}
	return cmd
}

func GetUnmountCmd() *cobra.Command {
	return unmountCmd
}

func init() {
	RootCmd.AddCommand(GetUnmountCmd())
}
