package main

import (
	"log"
	"os"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/matrix"
)

func main() {
	// Create a new DataNode to hold the root filesystem.
	dn := datanode.New()
	dn.AddData("hello.txt", []byte("Hello from within the matrix!"))

	// Create a new TerminalIsolationMatrix from the DataNode.
	m, err := matrix.FromDataNode(dn)
	if err != nil {
		log.Fatalf("Failed to create matrix: %v", err)
	}

	// Serialize the matrix to a tarball.
	tarball, err := m.ToTar()
	if err != nil {
		log.Fatalf("Failed to serialize matrix to tar: %v", err)
	}

	// Write the tarball to a file.
	err = os.WriteFile("programmatic.matrix", tarball, 0644)
	if err != nil {
		log.Fatalf("Failed to write matrix file: %v", err)
	}

	log.Println("Successfully created programmatic.matrix")
}
