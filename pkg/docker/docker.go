package docker

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// Collect pulls a Docker image and saves it as an OCI tarball.
func Collect(imageRef, outputFile string, allTags bool, platform, registry string) error {
	if outputFile == "" && !allTags {
		return fmt.Errorf("output file not specified")
	}

	if registry != "" {
		imageRef = fmt.Sprintf("%s/%s", registry, imageRef)
	}

	var options []crane.Option
	options = append(options, crane.WithAuthFromKeychain(authn.DefaultKeychain))
	if platform != "" {
		p, err := v1.ParsePlatform(platform)
		if err != nil {
			return fmt.Errorf("parsing platform %q: %w", platform, err)
		}
		options = append(options, crane.WithPlatform(p))
	}

	if allTags {
		repo := imageRef
		if strings.Contains(repo, ":") {
			repo = strings.Split(repo, ":")[0]
		}
		tags, err := crane.ListTags(repo, options...)
		if err != nil {
			return fmt.Errorf("listing tags for %q: %w", repo, err)
		}

		for _, tag := range tags {
			taggedImageRef := fmt.Sprintf("%s:%s", repo, tag)
			img, err := crane.Pull(taggedImageRef, options...)
			if err != nil {
				return fmt.Errorf("pulling image %q: %w", taggedImageRef, err)
			}

			// a tar file can't have a colon in the name
			safeTag := strings.ReplaceAll(tag, ":", "_")
			outputFile := fmt.Sprintf("%s_%s.tar", repo, safeTag)
			f, err := os.Create(outputFile)
			if err != nil {
				return fmt.Errorf("creating file %q: %w", outputFile, err)
			}
			defer f.Close()
			if err := crane.Export(img, f); err != nil {
				return fmt.Errorf("saving image %q to %q: %w", taggedImageRef, outputFile, err)
			}
		}
	} else {
		img, err := crane.Pull(imageRef, options...)
		if err != nil {
			return fmt.Errorf("pulling image %q: %w", imageRef, err)
		}
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("creating file %q: %w", outputFile, err)
		}
		defer f.Close()

		if err := crane.Export(img, f); err != nil {
			return fmt.Errorf("saving image %q to %q: %w", imageRef, outputFile, err)
		}
	}

	return nil
}
