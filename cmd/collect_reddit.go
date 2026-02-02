package cmd

import (
	"github.com/spf13/cobra"
)

// collectRedditCmd represents the collect reddit command
var collectRedditCmd = NewCollectRedditCmd()

func init() {
	GetCollectCmd().AddCommand(GetCollectRedditCmd())
}

func NewCollectRedditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reddit",
		Short: "Collect a resource from Reddit.",
		Long:  `Collect a resource from Reddit and store it in a DataNode.`,
	}
}

func GetCollectRedditCmd() *cobra.Command {
	return collectRedditCmd
}
