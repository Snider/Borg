package cmd

import (
	"fmt"
	"os"

	"github.com/Snider/Borg/pkg/collect"
	"github.com/spf13/cobra"
)

// collectNpmCmd represents the collect npm command
var collectNpmCmd = NewCollectNpmCmd()

func init() {
	GetCollectCmd().AddCommand(GetCollectNpmCmd())
}

func GetCollectNpmCmd() *cobra.Command {
	return collectNpmCmd
}

func NewCollectNpmCmd() *cobra.Command {
	collectNpmCmd := &cobra.Command{
		Use:   "npm [package]",
		Short: "Collect a single npm package",
		Long:  `Collect a single npm package and store it in a DataNode.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := args[0]
			outputFile, err := cmd.Flags().GetString("output")
			if err != nil {
				return fmt.Errorf("could not get output flag: %w", err)
			}

			collector := collect.NewNPMCollector()
			dn, err := collector.Collect(packageName)
			if err != nil {
				return fmt.Errorf("error collecting npm package: %w", err)
			}

			data, err := dn.ToTar()
			if err != nil {
				return fmt.Errorf("error serializing DataNode: %w", err)
			}

			if outputFile == "" {
				outputFile = packageName + ".dat"
			}

			err = os.WriteFile(outputFile, data, 0644)
			if err != nil {
				return fmt.Errorf("error writing npm package to file: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "NPM package saved to", outputFile)
			return nil
		},
	}
	collectNpmCmd.PersistentFlags().String("output", "", "Output file for the DataNode")
	return collectNpmCmd
}
