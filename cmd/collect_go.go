package cmd

import (
	"fmt"
	"os"

	"github.com/Snider/Borg/pkg/collect"
	"github.com/spf13/cobra"
)

// collectGoCmd represents the collect go command
var collectGoCmd = NewCollectGoCmd()

func init() {
	GetCollectCmd().AddCommand(GetCollectGoCmd())
}

func GetCollectGoCmd() *cobra.Command {
	return collectGoCmd
}

func NewCollectGoCmd() *cobra.Command {
	collectGoCmd := &cobra.Command{
		Use:   "go [module]",
		Short: "Collect a single Go module",
		Long:  `Collect a single Go module and store it in a DataNode.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			modulePath := args[0]
			outputFile, err := cmd.Flags().GetString("output")
			if err != nil {
				return fmt.Errorf("could not get output flag: %w", err)
			}

			collector := collect.NewGoCollector()
			dn, err := collector.Collect(modulePath)
			if err != nil {
				return fmt.Errorf("error collecting go module: %w", err)
			}

			data, err := dn.ToTar()
			if err != nil {
				return fmt.Errorf("error serializing DataNode: %w", err)
			}

			if outputFile == "" {
				outputFile = modulePath + ".dat"
			}

			err = os.WriteFile(outputFile, data, 0644)
			if err != nil {
				return fmt.Errorf("error writing go module to file: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Go module saved to", outputFile)
			return nil
		},
	}
	collectGoCmd.PersistentFlags().String("output", "", "Output file for the DataNode")
	return collectGoCmd
}
