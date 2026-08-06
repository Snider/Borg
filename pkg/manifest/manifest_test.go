package manifest

import (
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	dn := datanode.New()
	dn.AddData("file1.txt", []byte("hello"))
	dn.AddData("file2.txt", []byte("world"))
	dn.AddData("dir/file3.go", []byte("package main"))

	manifest, err := Generate(dn, "test", "datanode", false)
	assert.NoError(t, err)

	assert.Equal(t, "test", manifest.Source)
	assert.Equal(t, "datanode", manifest.Format)
	assert.False(t, manifest.Encrypted)
	assert.Len(t, manifest.Files, 3)
	assert.Equal(t, 3, manifest.Stats.TotalFiles)
	assert.Equal(t, "22 B", manifest.Stats.TotalSize)
	assert.Equal(t, map[string]int{"txt": 2, "go": 1}, manifest.Stats.ByType)
}
