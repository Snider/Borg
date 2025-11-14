package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Snider/Borg/pkg/github"
	"github.com/Snider/Borg/pkg/vcs"
)

func main() {
	log.Println("Collecting all repositories for a user...")

	repos, err := github.NewGithubClient().GetPublicRepos(context.Background(), "Snider")
	if err != nil {
		log.Fatalf("Failed to get public repos: %v", err)
	}

	cloner := vcs.NewGitCloner()

	for _, repo := range repos {
		log.Printf("Cloning %s...", repo)
		dn, err := cloner.CloneGitRepository(fmt.Sprintf("https://github.com/%s", repo), nil)
		if err != nil {
			log.Printf("Failed to clone %s: %v", repo, err)
			continue
		}

		tarball, err := dn.ToTar()
		if err != nil {
			log.Printf("Failed to serialize %s to tar: %v", repo, err)
			continue
		}

		err = os.WriteFile(fmt.Sprintf("%s.dat", repo), tarball, 0644)
		if err != nil {
			log.Printf("Failed to write %s.dat: %v", repo, err)
			continue
		}
		log.Printf("Successfully created %s.dat", repo)
	}
}
