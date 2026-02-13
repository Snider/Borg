package cmd

import (
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/storage"
	"github.com/Snider/Borg/pkg/tim"
	"github.com/Snider/Borg/pkg/trix"
	"github.com/Snider/Borg/pkg/ui"
	"github.com/Snider/Borg/pkg/vcs"

	"github.com/spf13/cobra"
)

const (
	defaultFilePermission = 0644
)

var (
	// GitCloner is the git cloner used by the command. It can be replaced for testing.
	GitCloner = vcs.NewGitCloner()
)

// NewCollectGithubRepoCmd creates a new cobra command for collecting a single git repository.
func NewCollectGithubRepoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo [repository-url]",
		Short: "Collect a single Git repository",
		Long:  `Collect a single Git repository and store it in a DataNode.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoURL := args[0]
			outputFile, _ := cmd.Flags().GetString("output")
			format, _ := cmd.Flags().GetString("format")
			compression, _ := cmd.Flags().GetString("compression")
			password, _ := cmd.Flags().GetString("password")

			if format != "datanode" && format != "tim" && format != "trix" && format != "stim" {
				return fmt.Errorf("invalid format: %s (must be 'datanode', 'tim', 'trix', or 'stim')", format)
			}
			if compression != "none" && compression != "gz" && compression != "xz" {
				return fmt.Errorf("invalid compression: %s (must be 'none', 'gz', or 'xz')", compression)
			}

			prompter := ui.NewNonInteractivePrompter(ui.GetVCSQuote)
			prompter.Start()
			defer prompter.Stop()

			var progressWriter io.Writer
			if prompter.IsInteractive() {
				bar := ui.NewProgressBar(-1, "Cloning repository")
				progressWriter = ui.NewProgressWriter(bar)
			}

			dn, err := GitCloner.CloneGitRepository(repoURL, progressWriter)
			if err != nil {
				return fmt.Errorf("error cloning repository: %w", err)
			}

			var data []byte
			if format == "tim" {
				t, err := tim.FromDataNode(dn)
				if err != nil {
					return fmt.Errorf("error creating tim: %w", err)
				}
				data, err = t.ToTar()
				if err != nil {
					return fmt.Errorf("error serializing tim: %w", err)
				}
			} else if format == "stim" {
				if password == "" {
					return fmt.Errorf("password required for stim format")
				}
				t, err := tim.FromDataNode(dn)
				if err != nil {
					return fmt.Errorf("error creating tim: %w", err)
				}
				data, err = t.ToSigil(password)
				if err != nil {
					return fmt.Errorf("error encrypting stim: %w", err)
				}
			} else if format == "trix" {
				data, err = trix.ToTrix(dn, password)
				if err != nil {
					return fmt.Errorf("error serializing trix: %w", err)
				}
			} else {
				data, err = dn.ToTar()
				if err != nil {
					return fmt.Errorf("error serializing DataNode: %w", err)
				}
			}

			if outputFile == "" {
				outputFile = "repo." + format
				if compression != "none" {
					outputFile += "." + compression
				}
			}

			u, err := url.Parse(outputFile)
			if err != nil {
				return fmt.Errorf("invalid output URL: %w", err)
			}

			pr, pw := io.Pipe()

			go func() {
				defer pw.Close()
				compressWriter, err := compress.NewWriter(pw, compression)
				if err != nil {
					pw.CloseWithError(fmt.Errorf("error creating compress writer: %w", err))
					return
				}
				defer compressWriter.Close()
				_, err = compressWriter.Write(data)
				if err != nil {
					pw.CloseWithError(fmt.Errorf("error writing compressed data: %w", err))
					return
				}
			}()

			if u.Scheme == "" || u.Scheme == "file" {
				f, err := os.Create(outputFile)
				if err != nil {
					return fmt.Errorf("error creating file: %w", err)
				}
				defer f.Close()
				_, err = io.Copy(f, pr)
				if err != nil {
					return fmt.Errorf("error writing to file: %w", err)
				}
			} else {
				s, err := storage.NewStorage(u)
				if err != nil {
					return err
				}

				err = s.Write(u.Path, pr)
				if err != nil {
					return fmt.Errorf("error uploading file: %w", err)
				}
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Repository saved to", outputFile)
			return nil
		},
	}
	cmd.Flags().String("output", "", "Output file for the DataNode")
	cmd.Flags().String("format", "datanode", "Output format (datanode, tim, trix, or stim)")
	cmd.Flags().String("compression", "none", "Compression format (none, gz, or xz)")
	cmd.Flags().String("password", "", "Password for encryption (required for trix/stim)")
	return cmd
}

func init() {
	collectGithubCmd.AddCommand(NewCollectGithubRepoCmd())
}
