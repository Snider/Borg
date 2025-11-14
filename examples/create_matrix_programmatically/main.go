package main

import (
	"log"
	"os"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/tim"
)

func main() {
	// Create a new DataNode to hold the root filesystem.
	dn := datanode.New()
	dn.AddData("hello.txt", []byte("Hello from within the TIM!"))

	// Create a new TerminalIsolationMatrix from the DataNode.
	m, err := tim.FromDataNode(dn)
	if err != nil {
		log.Fatalf("Failed to create TIM: %v", err)
	}

	// Serialize the TIM to a tarball.
	tarball, err := m.ToTar()
	if err != nil {
		log.Fatalf("Failed to serialize TIM to tar: %v", err)
	}

	// Write the tarball to a file.
	err = os.WriteFile("programmatic.tim", tarball, 0644)
	if err != nil {
		log.Fatalf("Failed to write TIM file: %v", err)
	}

	log.Println("Successfully created programmatic.tim")
}
