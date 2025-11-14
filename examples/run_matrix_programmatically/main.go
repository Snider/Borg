package main

import (
	"log"

	"github.com/Snider/Borg/pkg/matrix"
)

func main() {
	log.Println("Executing matrix with Borg...")

	// Execute the matrix using the Borg package.
	if err := matrix.Run("programmatic.matrix"); err != nil {
		log.Fatalf("Failed to run matrix: %v", err)
	}

	log.Println("Matrix execution finished.")
}
