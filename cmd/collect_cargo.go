package cmd

import (
	"fmt"
	"os"

	"github.com/Snider/Borg/pkg/collect"
	"github.com/spf13/cobra"
)

// collectCargoCmd represents the collect cargo command
var collectCargoCmd = NewCollectCargoCmd()

func init() {
	GetCollectCmd().AddCommand(GetCollectCargoCmd())
}

func GetCollectCargoCmd() *cobra.Command {
	return collectCargoCmd
}

func NewCollectCargoCmd() *cobra.Command {
	collectCargoCmd := &cobra.Command{
		Use:   "cargo [package]",
		Short: "Collect a single cargo package",
		Long:  `Collect a single cargo package and store it in a DataNode.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := args[0]
			outputFile, err := cmd.Flags().GetString("output")
			if err != nil {
				return fmt.Errorf("could not get output flag: %w", err)
			}

			collector := collect.NewCargoCollector()
			dn, err := collector.Collect(packageName)
			if err != nil {
				return fmt.Errorf("error collecting cargo package: %w", err)
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
				return fmt.Errorf("error writing cargo package to file: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Cargo package saved to", outputFile)
			return nil
		},
	}
	collectCargoCmd.PersistentFlags().String("output", "", "Output file for the DataNode")
	return collectCargoCmd
}
