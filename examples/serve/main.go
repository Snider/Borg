package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Snider/Borg/pkg/compress"
	"github.com/Snider/Borg/pkg/tarfs"
)

func main() {
	log.Println("Serving datanode...")

	// Create a dummy datanode for serving.
	err := os.WriteFile("serve.dat", []byte{}, 0644)
	if err != nil {
		log.Fatalf("Failed to create dummy datanode: %v", err)
	}

	data, err := os.ReadFile("serve.dat")
	if err != nil {
		log.Fatalf("Failed to read datanode: %v", err)
	}

	decompressed, err := compress.Decompress(data)
	if err != nil {
		log.Fatalf("Failed to decompress datanode: %v", err)
	}

	fs, err := tarfs.New(decompressed)
	if err != nil {
		log.Fatalf("Failed to create tarfs: %v", err)
	}

	http.Handle("/", http.FileServer(fs))
	log.Println("Listening on :8080...")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
