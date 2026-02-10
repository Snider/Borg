package main

import (
	"log"
	"os"

	"github.com/Snider/Borg/pkg/hooks"
	"github.com/Snider/Borg/pkg/pwa"
)

func main() {
	log.Println("Collecting PWA...")

	client := pwa.NewPWAClient()
	pwaURL := "https://squoosh.app"

	manifestURL, err := client.FindManifest(pwaURL)
	if err != nil {
		log.Fatalf("Failed to find manifest: %v", err)
	}

	hookRunner, err := hooks.NewHookRunner("")
	if err != nil {
		log.Fatalf("Failed to create hook runner: %v", err)
	}

	dn, err := client.DownloadAndPackagePWA(pwaURL, manifestURL, nil, hookRunner)
	if err != nil {
		log.Fatalf("Failed to download and package PWA: %v", err)
	}

	tarball, err := dn.ToTar()
	if err != nil {
		log.Fatalf("Failed to serialize datanode to tar: %v", err)
	}

	err = os.WriteFile("pwa.dat", tarball, 0644)
	if err != nil {
		log.Fatalf("Failed to write datanode file: %v", err)
	}

	log.Println("Successfully created pwa.dat")
}
