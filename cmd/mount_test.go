package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/hanwen/go-fuse/v2/fuse"
)

func TestMountCommand(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping mount test on non-linux systems")
	}
	if _, err := exec.LookPath("fusermount"); err != nil {
		t.Skip("fusermount not found, skipping mount test")
	}

	// Create a temporary directory for the mount point
	mountDir := t.TempDir()

	// Create a dummy archive
	dn := datanode.New()
	dn.AddData("test.txt", []byte("hello"))
	tarball, err := dn.ToTar()
	if err != nil {
		t.Fatal(err)
	}

	archiveFile := filepath.Join(t.TempDir(), "test.dat")
	if err := os.WriteFile(archiveFile, tarball, 0644); err != nil {
		t.Fatal(err)
	}

	// Run the mount command in the background
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := NewMountCmd()
	cmd.SetArgs([]string{archiveFile, mountDir})

	cmdErrCh := make(chan error, 1)
	go func() {
		cmdErrCh <- cmd.ExecuteContext(ctx)
	}()

	// Give the FUSE server a moment to start up and stabilize
	time.Sleep(3 * time.Second)

	// Check that the mount is active.
	mounted, err := isMounted(mountDir)
	if err != nil {
		t.Fatalf("Failed to check if mounted: %v", err)
	}
	if !mounted {
		t.Errorf("Mount directory does not appear to be mounted.")
	}


	// Cancel the context to stop the command
	cancel()

	// Wait for the command to exit
	select {
	case err := <-cmdErrCh:
		if err != nil && err != context.Canceled {
			t.Errorf("Mount command returned an unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Mount command did not exit after context was canceled.")
	}

	// Give the filesystem a moment to unmount.
	time.Sleep(1 * time.Second)


	// Verify that the mount point is now unmounted.
	mounted, err = isMounted(mountDir)
	if err != nil {
		t.Fatalf("Failed to check if unmounted: %v", err)
	}
	if mounted {
		// As a fallback, try to unmount it manually.
		server, _ := fuse.NewServer(nil, mountDir, nil)
		server.Unmount()
		t.Errorf("Mount directory was not unmounted cleanly.")
	}
}

// isMounted checks if a directory is a FUSE mount point.
// This is a simple heuristic and might not be 100% reliable.
func isMounted(path string) (bool, error) {
	// On Linux, we can check the mountinfo.
	if runtime.GOOS == "linux" {
		data, err := os.ReadFile("/proc/self/mountinfo")
		if err != nil {
			return false, err
		}
		return strings.Contains(string(data), path) && strings.Contains(string(data), "fuse"), nil
	}
	return false, nil // Not implemented for other OSes.
}
