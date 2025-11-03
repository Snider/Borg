package main

import (
	"os"

	"github.com/Snider/Borg/cmd"
	"github.com/Snider/Borg/pkg/logger"
)

// main is the entry point for the Borg CLI application.
func main() {
	verbose, _ := cmd.RootCmd.PersistentFlags().GetBool("verbose")
	log := logger.New(verbose)
	if err := cmd.Execute(log); err != nil {
		log.Error("fatal error", "err", err)
		os.Exit(1)
	}
}
