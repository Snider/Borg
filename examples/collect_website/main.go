package main

import (
	"log"
	"os"

	"github.com/Snider/Borg/pkg/website"
)

func main() {
	log.Println("Collecting website...")

	// Download and package the website.
	opts := website.DownloadOptions{
		URL:                  "https://example.com",
		MaxDepth:             2,
		EnableCircuitBreaker: true,
	}
	dn, err := website.DownloadAndPackageWebsite(opts)
	if err != nil {
		log.Fatalf("Failed to collect website: %v", err)
	}

	// Serialize the DataNode to a tarball.
	tarball, err := dn.ToTar()
	if err != nil {
		log.Fatalf("Failed to serialize datanode to tar: %v", err)
	}

	// Write the tarball to a file.
	err = os.WriteFile("website.dat", tarball, 0644)
	if err != nil {
		log.Fatalf("Failed to write datanode file: %v", err)
	}

	log.Println("Successfully created website.dat")
}
