// mkdemo-v3 creates a v3 chunked SMSG file for streaming demos
package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Snider/Borg/pkg/smsg"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: mkdemo-v3 <input-media-file> <output-smsg-file> [license] [chunk-size-kb]")
		fmt.Println("")
		fmt.Println("Creates a v3 chunked SMSG file for streaming demos.")
		fmt.Println("V3 uses rolling keys derived from: LTHN(date:license:fingerprint)")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  license       The license key (default: auto-generated)")
		fmt.Println("  chunk-size-kb Chunk size in KB (default: 512)")
		fmt.Println("")
		fmt.Println("Note: V3 files work for 24-48 hours from creation (rolling keys).")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]

	// Read input file
	content, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("Failed to read input file: %v\n", err)
		os.Exit(1)
	}

	// License (acts as password in v3)
	var license string
	if len(os.Args) > 3 {
		license = os.Args[3]
	} else {
		// Generate cryptographically secure license
		licenseBytes := make([]byte, 24)
		if _, err := rand.Read(licenseBytes); err != nil {
			fmt.Printf("Failed to generate license: %v\n", err)
			os.Exit(1)
		}
		license = base64.RawURLEncoding.EncodeToString(licenseBytes)
	}

	// Chunk size (default 512KB for good streaming granularity)
	chunkSize := 512 * 1024
	if len(os.Args) > 4 {
		var chunkKB int
		if _, err := fmt.Sscanf(os.Args[4], "%d", &chunkKB); err == nil && chunkKB > 0 {
			chunkSize = chunkKB * 1024
		}
	}

	// Create manifest
	title := filepath.Base(inputFile)
	ext := filepath.Ext(title)
	if ext != "" {
		title = title[:len(title)-len(ext)]
	}
	manifest := smsg.NewManifest(title)
	manifest.LicenseType = "streaming"
	manifest.Format = "dapp.fm/v3-chunked"

	// Detect MIME type
	mimeType := "video/mp4"
	switch ext {
	case ".mp3":
		mimeType = "audio/mpeg"
	case ".wav":
		mimeType = "audio/wav"
	case ".flac":
		mimeType = "audio/flac"
	case ".webm":
		mimeType = "video/webm"
	case ".ogg":
		mimeType = "audio/ogg"
	}

	// Create message with attachment
	msg := smsg.NewMessage("dapp.fm V3 Streaming Demo - Decrypt-while-downloading enabled")
	msg.Subject = "V3 Chunked Streaming"
	msg.From = "dapp.fm"
	msg.AddBinaryAttachment(
		filepath.Base(inputFile),
		content,
		mimeType,
	)

	// Create stream params with chunking enabled
	params := &smsg.StreamParams{
		License:     license,
		Fingerprint: "", // Empty for demo (works for any device)
		Cadence:     smsg.CadenceDaily,
		ChunkSize:   chunkSize,
	}

	// Encrypt with v3 chunked format
	encrypted, err := smsg.EncryptV3(msg, params, manifest)
	if err != nil {
		fmt.Printf("Failed to encrypt: %v\n", err)
		os.Exit(1)
	}

	// Write output
	if err := os.WriteFile(outputFile, encrypted, 0644); err != nil {
		fmt.Printf("Failed to write output: %v\n", err)
		os.Exit(1)
	}

	// Calculate chunk count
	numChunks := (len(content) + chunkSize - 1) / chunkSize

	fmt.Printf("Created: %s (%d bytes)\n", outputFile, len(encrypted))
	fmt.Printf("Format: v3 chunked\n")
	fmt.Printf("Chunk Size: %d KB\n", chunkSize/1024)
	fmt.Printf("Total Chunks: ~%d\n", numChunks)
	fmt.Printf("License: %s\n", license)
	fmt.Println("")
	fmt.Println("This license works for 24-48 hours from creation.")
	fmt.Println("Use the license in the streaming demo to decrypt.")
}
