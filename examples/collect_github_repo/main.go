package main

import (
	"log"
	"os"

	"github.com/Snider/Borg/pkg/vcs"
)

func main() {
	log.Println("Collecting GitHub repo...")

	cloner := vcs.NewGitCloner()
	dn, err := cloner.CloneGitRepository("https://github.com/Snider/Borg", nil)
	if err != nil {
		log.Fatalf("Failed to clone repository: %v", err)
	}

	tarball, err := dn.ToTar()
	if err != nil {
		log.Fatalf("Failed to serialize datanode to tar: %v", err)
	}

	err = os.WriteFile("repo.dat", tarball, 0644)
	if err != nil {
		log.Fatalf("Failed to write datanode file: %v", err)
	}

	log.Println("Successfully created repo.dat")
}
