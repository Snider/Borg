package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Snider/Borg/pkg/changelog"
	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/tim"
	"github.com/Snider/Borg/pkg/trix"
	"github.com/Snider/Borg/pkg/vcs"
	"github.com/spf13/cobra"
)

var changelogCmd = NewChangelogCmd()

func NewChangelogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "changelog [old] [new]",
		Short: "Generate a changelog between two archives",
		Args: func(cmd *cobra.Command, args []string) error {
			source, _ := cmd.Flags().GetString("source")
			if source != "" {
				if len(args) != 1 {
					return errors.New("accepts one archive when --source is set")
				}
			} else {
				if len(args) != 2 {
					return errors.New("accepts two archives when --source is not set")
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")
			password, _ := cmd.Flags().GetString("password")
			source, _ := cmd.Flags().GetString("source")

			var oldNode, newNode *datanode.DataNode
			var err error

			if source != "" {
				oldNode, err = getDataNode(args[0], password)
				if err != nil {
					return fmt.Errorf("failed to read old archive: %w", err)
				}
				newNode, err = getDataNodeFromSource(source)
				if err != nil {
					return fmt.Errorf("failed to read source: %w", err)
				}
			} else {
				oldNode, err = getDataNode(args[0], password)
				if err != nil {
					return fmt.Errorf("failed to read old archive: %w", err)
				}
				newNode, err = getDataNode(args[1], password)
				if err != nil {
					return fmt.Errorf("failed to read new archive: %w", err)
				}
			}

			report, err := changelog.GenerateReport(oldNode, newNode)
			if err != nil {
				return fmt.Errorf("failed to generate changelog: %w", err)
			}

			var output string
			switch format {
			case "markdown":
				output, err = changelog.FormatAsMarkdown(report)
			case "json":
				output, err = changelog.FormatAsJSON(report)
			default:
				output, err = changelog.FormatAsText(report)
			}

			if err != nil {
				return fmt.Errorf("failed to format changelog: %w", err)
			}

			fmt.Fprint(os.Stdout, output)

			return nil
		},
	}

	cmd.Flags().String("format", "text", "Output format (text, markdown, json)")
	cmd.Flags().String("password", "", "Password for encrypted archives")
	cmd.Flags().String("source", "", "Remote source to compare against (e.g., github:org/repo)")

	return cmd
}

func getDataNode(filePath, password string) (*datanode.DataNode, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Handle .stim files
	if strings.HasSuffix(filePath, ".stim") || (len(data) >= 4 && string(data[:4]) == "STIM") {
		if password == "" {
			return nil, fmt.Errorf("password required for .stim files")
		}
		m, err := tim.FromSigil(data, password)
		if err != nil {
			return nil, err
		}
		tarball, err := m.ToTar()
		if err != nil {
			return nil, err
		}
		return datanode.FromTar(tarball)
	}

	// Handle .trix files
	if strings.HasSuffix(filePath, ".trix") {
		dn, err := trix.FromTrix(data, password)
		if err != nil {
			return nil, err
		}
		return dn, nil
	}

	// Assume it's a tarball
	return datanode.FromTar(data)
}

func getDataNodeFromSource(source string) (*datanode.DataNode, error) {
	parts := strings.SplitN(source, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid source format: %s", source)
	}

	sourceType := parts[0]
	sourcePath := parts[1]

	switch sourceType {
	case "github":
		url := "https://" + sourceType + ".com/" + sourcePath
		gitCloner := vcs.NewGitCloner()
		return gitCloner.CloneGitRepository(url, os.Stdout)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", sourceType)
	}
}


func GetChangelogCmd() *cobra.Command {
	return changelogCmd
}

func init() {
	RootCmd.AddCommand(GetChangelogCmd())
}
