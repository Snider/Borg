package main

import (
	"log"

	"github.com/Snider/Borg/pkg/tim"
)

func main() {
	log.Println("Executing tim with Borg...")

	// Execute the tim using the Borg package.
	if err := tim.Run("programmatic.tim"); err != nil {
		log.Fatalf("Failed to run tim: %v", err)
	}
}
