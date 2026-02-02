package changelog

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/pmezard/go-difflib/difflib"
)

// ChangeReport represents the differences between two archives.
type ChangeReport struct {
	Added    []string       `json:"added"`
	Modified []ModifiedFile `json:"modified"`
	Removed  []string       `json:"removed"`
}

// ModifiedFile represents a file that has been modified.
type ModifiedFile struct {
	Path      string   `json:"path"`
	Additions int      `json:"additions"`
	Deletions int      `json:"deletions"`
	Commits   []string `json:"commits,omitempty"`
}

// GenerateReport compares two DataNodes and returns a ChangeReport.
func GenerateReport(oldNode, newNode *datanode.DataNode) (*ChangeReport, error) {
	report := &ChangeReport{}

	walkOptions := datanode.WalkOptions{
		SkipErrors: true,
		Filter: func(path string, d fs.DirEntry) bool {
			return !strings.HasPrefix(path, ".git")
		},
	}

	oldFiles := make(map[string][]byte)
	if err := oldNode.Walk(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || path == "." {
			return nil
		}
		file, err := oldNode.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		content, err := io.ReadAll(file)
		if err != nil {
			return err
		}
		oldFiles[path] = content
		return nil
	}, walkOptions); err != nil {
		return nil, err
	}

	newFiles := make(map[string][]byte)
	if err := newNode.Walk(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || path == "." {
			return nil
		}
		file, err := newNode.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		content, err := io.ReadAll(file)
		if err != nil {
			return err
		}
		newFiles[path] = content
		return nil
	}, walkOptions); err != nil {
		return nil, err
	}

	commitMap, err := getCommitMap(newNode)
	if err != nil {
		// Not a git repo, so we can ignore this error
	}

	for path, newContent := range newFiles {
		oldContent, ok := oldFiles[path]
		if !ok {
			report.Added = append(report.Added, path)
			continue
		}

		if !bytes.Equal(oldContent, newContent) {
			diff := difflib.UnifiedDiff{
				A:        difflib.SplitLines(string(oldContent)),
				B:        difflib.SplitLines(string(newContent)),
				FromFile: "old/" + path,
				ToFile:   "new/" + path,
				Context:  0,
			}
			diffString, err := difflib.GetUnifiedDiffString(diff)
			if err != nil {
				return nil, err
			}

			var additions, deletions int
			for _, line := range strings.Split(diffString, "\n") {
				if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
					additions++
				}
				if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
					deletions++
				}
			}
			report.Modified = append(report.Modified, ModifiedFile{
				Path:      path,
				Additions: additions,
				Deletions: deletions,
				Commits:   commitMap[path],
			})
		}
	}

	for path := range oldFiles {
		if _, ok := newFiles[path]; !ok {
			report.Removed = append(report.Removed, path)
		}
	}

	return report, nil
}

func getCommitMap(node *datanode.DataNode) (map[string][]string, error) {
	exists, err := node.Exists(".git", datanode.ExistsOptions{WantType: fs.ModeDir})
	if err != nil || !exists {
		return nil, os.ErrNotExist
	}

	memFS := memfs.New()
	err = node.Walk(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}
		if d.IsDir() {
			return memFS.MkdirAll(path, 0755)
		}
		file, err := node.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		content, err := io.ReadAll(file)
		if err != nil {
			return err
		}
		f, err := memFS.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write(content)
		return err
	})
	if err != nil {
		return nil, err
	}

	dotGitFS, err := memFS.Chroot(".git")
	if err != nil {
		return nil, err
	}
	storer := filesystem.NewStorage(dotGitFS, nil)

	repo, err := git.Open(storer, nil)
	if err != nil {
		return nil, err
	}

	commitIter, err := repo.Log(&git.LogOptions{All: true})
	if err != nil {
		return nil, err
	}

	commitMap := make(map[string][]string)
	err = commitIter.ForEach(func(c *object.Commit) error {
		if c.NumParents() == 0 {
			tree, err := c.Tree()
			if err != nil {
				return err
			}
			walker := object.NewTreeWalker(tree, true, nil)
			defer walker.Close()
			for {
				name, _, err := walker.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}
				commitMap[name] = append(commitMap[name], c.Message)
			}
		} else {
			parent, err := c.Parent(0)
			if err != nil {
				return err
			}
			patch, err := parent.Patch(c)
			if err != nil {
				return err
			}
			for _, filePatch := range patch.FilePatches() {
				from, to := filePatch.Files()
				paths := make(map[string]struct{})
				if from != nil {
					paths[from.Path()] = struct{}{}
				}
				if to != nil {
					paths[to.Path()] = struct{}{}
				}
				for p := range paths {
					commitMap[p] = append(commitMap[p], c.Message)
				}
			}
		}
		return nil
	})

	return commitMap, err
}
