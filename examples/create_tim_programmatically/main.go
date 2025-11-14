package main

import (
	"log"
	"os"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/tim"
)

func main() {
	// Create a new DataNode and add a file to it.
	dn := datanode.New()
	dn.AddData("hello.txt", []byte("Hello from within the tim!"))

	// Create a new TerminalIsolationMatrix from the DataNode.
	m, err := tim.FromDataNode(dn)
	if err != nil {
		log.Fatalf("Failed to create tim: %v", err)
	}

	// Serialize the tim to a tarball.
	tarball, err := m.ToTar()
	if err != nil {
		log.Fatalf("Failed to serialize tim to tar: %v", err)
	}

	// Write the tarball to a file.
	err = os.WriteFile("programmatic.tim", tarball, 0644)
	if err != nil {
		log.Fatalf("Failed to write tim file: %v", err)
	}

	log.Println("Successfully created programmatic.tim")
}
