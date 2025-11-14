package main

import (
	"log"

	"github.com/Snider/Borg/pkg/tim"
)

func main() {
	log.Println("Executing TIM with Borg...")

	// Execute the TIM using the Borg package.
	if err := tim.Run("programmatic.tim"); err != nil {
		log.Fatalf("Failed to run TIM: %v", err)
	}

	log.Println("TIM execution finished.")
}
