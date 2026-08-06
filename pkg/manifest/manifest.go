package manifest

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/Snider/Borg/pkg/datanode"
)

type Manifest struct {
	CollectedAt string    `json:"collected_at"`
	Source      string    `json:"source"`
	Format      string    `json:"format"`
	Encrypted   bool      `json:"encrypted"`
	Files       []File    `json:"files"`
	Stats       Stats     `json:"stats"`
}

type File struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
	Type   string `json:"type"`
}

type Stats struct {
	TotalFiles int            `json:"total_files"`
	TotalSize  string         `json:"total_size"`
	ByType     map[string]int `json:"by_type"`
}

func Generate(dn *datanode.DataNode, source, format string, encrypted bool) (*Manifest, error) {
	manifest := &Manifest{
		CollectedAt: time.Now().UTC().Format(time.RFC3339),
		Source:      source,
		Format:      format,
		Encrypted:   encrypted,
		Files:       []File{},
		Stats: Stats{
			ByType: make(map[string]int),
		},
	}

	var totalSize int64
	err := dn.Walk(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		file, err := dn.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			return err
		}

		hasher := sha256.New()
		if _, err := hasher.Write(content); err != nil {
			return err
		}

		fileType := filepath.Ext(path)
		if fileType != "" {
			fileType = fileType[1:]
		}

		manifest.Files = append(manifest.Files, File{
			Path:   path,
			Size:   info.Size(),
			SHA256: fmt.Sprintf("%x", hasher.Sum(nil)),
			Type:   fileType,
		})

		totalSize += info.Size()
		manifest.Stats.ByType[fileType]++
		return nil
	})

	if err != nil {
		return nil, err
	}

	manifest.Stats.TotalFiles = len(manifest.Files)
	manifest.Stats.TotalSize = formatBytes(totalSize)

	return manifest, nil
}

func (m *Manifest) ToJSON() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
