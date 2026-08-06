package main

import (
	"log"
	"os"

	"github.com/Snider/Borg/pkg/github"
)

func main() {
	log.Println("Collecting GitHub release...")

	owner, repo, err := github.ParseRepoFromURL("https://github.com/Snider/Borg")
	if err != nil {
		log.Fatalf("Failed to parse repo from URL: %v", err)
	}

	release, err := github.GetLatestRelease(owner, repo)
	if err != nil {
		log.Fatalf("Failed to get latest release: %v", err)
	}

	if len(release.Assets) == 0 {
		log.Println("No assets found in the latest release.")
		return
	}

	asset := release.Assets[0]
	log.Printf("Downloading asset: %s", asset.GetName())

	file, err := os.Create(asset.GetName())
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	err = github.DownloadReleaseAsset(asset, file)
	if err != nil {
		log.Fatalf("Failed to download asset: %v", err)
	}

	log.Printf("Successfully downloaded asset to %s", asset.GetName())
}
