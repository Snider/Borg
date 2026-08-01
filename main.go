package main

import (
	"os"

	"github.com/Snider/Borg/cmd"
	"github.com/Snider/Borg/pkg/logger"
)

var osExit = os.Exit

func main() {
	Main()
}
func Main() {
	verbose, _ := cmd.RootCmd.PersistentFlags().GetBool("verbose")
	log := logger.New(verbose)
	if err := cmd.Execute(log); err != nil {
		log.Error("fatal error", "err", err)
		osExit(1)
	}
}
