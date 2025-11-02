package main

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/Snider/Borg/pkg/datanode"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run inspect_datanode.go <path to .dat file>")
		os.Exit(1)
	}

	datFile := os.Args[1]

	data, err := os.ReadFile(datFile)
	if err != nil {
		fmt.Printf("Error reading .dat file: %v\n", err)
		os.Exit(1)
	}

	dn, err := datanode.FromTar(data)
	if err != nil {
		fmt.Printf("Error creating DataNode from tarball: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Contents of %s:\n", datFile)
	err = dn.Walk(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	})
	if err != nil {
		fmt.Printf("Error walking DataNode: %v\n", err)
		os.Exit(1)
	}
}
