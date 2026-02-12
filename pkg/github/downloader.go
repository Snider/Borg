package github

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"strings"
	"sync"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/vcs"
	"github.com/schollz/progressbar/v3"
)

// Downloader manages a pool of workers for cloning repositories.
type Downloader struct {
	parallel int
	bar      *progressbar.ProgressBar
	cloner   vcs.GitCloner
}

// NewDownloader creates a new Downloader.
func NewDownloader(parallel int, bar *progressbar.ProgressBar) *Downloader {
	return &Downloader{
		parallel: parallel,
		bar:      bar,
		cloner:   vcs.NewGitCloner(),
	}
}

// DownloadRepositories downloads a list of repositories in parallel.
func (d *Downloader) DownloadRepositories(ctx context.Context, repos []string) (*datanode.DataNode, error) {
	var wg sync.WaitGroup
	repoChan := make(chan string, len(repos))
	errChan := make(chan error, len(repos))
	mergedDN := datanode.New()
	var mu sync.Mutex

	for i := 0; i < d.parallel; i++ {
		wg.Add(1)
		go d.worker(ctx, &wg, repoChan, mergedDN, &mu, errChan)
	}

	for _, repo := range repos {
		select {
		case repoChan <- repo:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	close(repoChan)

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("errors cloning repositories: %v", errs)
	}

	return mergedDN, nil
}

func (d *Downloader) worker(ctx context.Context, wg *sync.WaitGroup, repoChan <-chan string, mergedDN *datanode.DataNode, mu *sync.Mutex, errChan chan<- error) {
	defer wg.Done()
	for repoURL := range repoChan {
		select {
		case <-ctx.Done():
			return
		default:
		}

		repoName, err := GetRepoNameFromURL(repoURL)
		if err != nil {
			errChan <- err
			continue
		}

		dn, err := d.cloner.CloneGitRepository(repoURL, nil)
		if err != nil {
			errChan <- fmt.Errorf("error cloning %s: %w", repoURL, err)
			continue
		}

		err = dn.Walk(".", func(path string, de fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !de.IsDir() {
				file, err := dn.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()
				content, err := io.ReadAll(file)
				if err != nil {
					return err
				}
				mu.Lock()
				mergedDN.AddData(fmt.Sprintf("%s/%s", repoName, path), content)
				mu.Unlock()
			}
			return nil
		})
		if err != nil {
			errChan <- err
		}

		if d.bar != nil {
			d.bar.Add(1)
		}
	}
}

// GetRepoNameFromURL extracts the repository name from a Git URL.
func GetRepoNameFromURL(repoURL string) (string, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", err
	}
	path := strings.TrimSuffix(u.Path, ".git")
	return strings.TrimPrefix(path, "/"), nil
}
