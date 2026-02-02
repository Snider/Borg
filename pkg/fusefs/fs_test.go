package fusefs

import (
	"context"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
)

func TestDataNodeFs_Readdir(t *testing.T) {
	dn := datanode.New()
	dn.AddData("file1.txt", []byte("hello"))
	dn.AddData("dir1/file2.txt", []byte("world"))

	root := &DataNodeFs{dn: dn}
	stream, errno := root.Readdir(context.Background())
	if errno != 0 {
		t.Fatalf("Readdir failed: %v", errno)
	}
	if stream == nil {
		t.Fatal("Readdir returned a nil stream")
	}
}
