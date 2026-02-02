package cmd

import (
	"github.com/google/go-github/v39/github"
)

func findChecksumAsset(assets []*github.ReleaseAsset) *github.ReleaseAsset {
	for _, asset := range assets {
		// A common convention for checksum files.
		// A more robust solution could be configured by the user.
		if asset.GetName() == "checksums.txt" || asset.GetName() == "SHA256SUMS" {
			return asset
		}
	}
	return nil
}
